package syncer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"oncall-agent/internal/ai/agent/knowledge_index_pipeline"
	"oncall-agent/utility/client"
	"oncall-agent/utility/common"
	"oncall-agent/utility/log_call_back"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/compose"
)

type Syncer struct {
	cfg   Config
	store MetadataStore
}

func New(cfg Config, store MetadataStore) *Syncer {
	if cfg.Interval <= 0 {
		cfg.Interval = 10 * time.Minute
	}
	if cfg.MaxFileSizeMB <= 0 {
		cfg.MaxFileSizeMB = 50
	}
	return &Syncer{cfg: cfg, store: store}
}

func (s *Syncer) RunOnce(ctx context.Context) SyncResult {
	result := SyncResult{}
	for _, source := range s.cfg.Sources {
		if strings.TrimSpace(source.Type) == "" {
			source.Type = "local"
		}
		if source.Type != "local" {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Errorf("unsupported rag source type %q for %s", source.Type, source.Name))
			continue
		}
		s.scanLocalSource(ctx, source, &result)
	}
	return result
}

func (s *Syncer) Start(ctx context.Context) {
	if !s.cfg.Enabled {
		return
	}
	go func() {
		s.RunOnce(ctx)
		ticker := time.NewTicker(s.cfg.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.RunOnce(ctx)
			}
		}
	}()
}

func (s *Syncer) scanLocalSource(ctx context.Context, source SourceConfig, result *SyncResult) {
	root := strings.TrimSpace(source.URI)
	if root == "" {
		result.Failed++
		result.Errors = append(result.Errors, fmt.Errorf("empty local rag source uri: %s", source.Name))
		return
	}

	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err)
			return nil
		}
		if entry.IsDir() || !isSupportedDocument(path) {
			return nil
		}
		result.Scanned++
		status, err := s.syncFile(ctx, source, path)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err)
			return nil
		}
		switch status {
		case "indexed":
			result.Indexed++
		case "skipped":
			result.Skipped++
		}
		return nil
	})
}

func (s *Syncer) syncFile(ctx context.Context, source SourceConfig, path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "failed", err
	}
	if info.Size() > s.cfg.MaxFileSizeMB*1024*1024 {
		return "skipped", s.mark(ctx, source, path, "", info, "skipped", "file too large")
	}
	hash, err := fileSHA256(path)
	if err != nil {
		return "failed", err
	}
	existing, err := s.store.GetBySourcePath(ctx, source.Name, path)
	if err != nil {
		return "failed", err
	}
	if existing != nil && existing.ContentHash == hash && existing.FileSize == info.Size() && existing.ModifiedAt.Equal(info.ModTime()) {
		return "skipped", nil
	}

	r, err := knowledge_index_pipeline.BuildKnowledgeIndexing(ctx)
	if err != nil {
		_ = s.mark(ctx, source, path, hash, info, "failed", err.Error())
		return "failed", err
	}
	if err := deleteExistingChunks(ctx, path); err != nil {
		_ = s.mark(ctx, source, path, hash, info, "failed", err.Error())
		return "failed", err
	}
	_, err = r.Invoke(ctx, document.Source{URI: path}, compose.WithCallbacks(log_call_back.LogCallback(nil)))
	if err != nil {
		_ = s.mark(ctx, source, path, hash, info, "failed", err.Error())
		return "failed", err
	}
	return "indexed", s.mark(ctx, source, path, hash, info, "indexed", "")
}

func deleteExistingChunks(ctx context.Context, sourcePath string) error {
	cli, err := client.NewMilvusClient(ctx)
	if err != nil {
		return err
	}
	expr := fmt.Sprintf(`metadata["_source"] == "%s"`, escapeMilvusString(sourcePath))
	queryResult, err := cli.Query(ctx, common.MilvusCollectionName, []string{}, expr, []string{common.MilvusFieldID})
	if err != nil || len(queryResult) == 0 {
		return err
	}

	idsToDelete := make([]string, 0)
	for _, column := range queryResult {
		if column.Name() != common.MilvusFieldID {
			continue
		}
		for i := 0; i < column.Len(); i++ {
			id, getErr := column.GetAsString(i)
			if getErr == nil && id != "" {
				idsToDelete = append(idsToDelete, id)
			}
		}
	}
	if len(idsToDelete) == 0 {
		return nil
	}
	escapedIDs := make([]string, 0, len(idsToDelete))
	for _, id := range idsToDelete {
		escapedIDs = append(escapedIDs, escapeMilvusString(id))
	}
	deleteExpr := fmt.Sprintf(`id in ["%s"]`, strings.Join(escapedIDs, `","`))
	return cli.Delete(ctx, common.MilvusCollectionName, "", deleteExpr)
}

func escapeMilvusString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

func (s *Syncer) mark(ctx context.Context, source SourceConfig, path, hash string, info os.FileInfo, status, lastErr string) error {
	existing, err := s.store.GetBySourcePath(ctx, source.Name, path)
	if err != nil {
		return err
	}
	version := int64(1)
	if existing != nil {
		version = existing.IndexedVersion
		if status == "indexed" {
			version++
		}
	}
	return s.store.Upsert(ctx, &DocumentMetadata{
		SourceName:     source.Name,
		SourceType:     source.Type,
		Path:           path,
		ContentHash:    hash,
		FileSize:       info.Size(),
		ModifiedAt:     info.ModTime(),
		LastIndexedAt:  time.Now(),
		LastStatus:     status,
		LastError:      lastErr,
		IndexedVersion: version,
	})
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func isSupportedDocument(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".txt", ".pdf", ".docx", ".html":
		return true
	default:
		return false
	}
}
