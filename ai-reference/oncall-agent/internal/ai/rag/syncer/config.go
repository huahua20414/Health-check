package syncer

import (
	"context"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

func LoadConfig(ctx context.Context) Config {
	cfg := Config{
		Enabled:       false,
		Interval:      10 * time.Minute,
		MaxFileSizeMB: 50,
	}

	if value, err := g.Cfg().Get(ctx, "rag_sync.enabled"); err == nil {
		cfg.Enabled = value.Bool()
	}
	if value, err := g.Cfg().Get(ctx, "rag_sync.interval"); err == nil {
		if parsed, parseErr := time.ParseDuration(strings.TrimSpace(value.String())); parseErr == nil && parsed > 0 {
			cfg.Interval = parsed
		}
	}
	if value, err := g.Cfg().Get(ctx, "rag_sync.max_file_size_mb"); err == nil && value.Int64() > 0 {
		cfg.MaxFileSizeMB = value.Int64()
	}
	if value, err := g.Cfg().Get(ctx, "rag_sync.mysql_dsn"); err == nil {
		cfg.MySQLDSN = strings.TrimSpace(value.String())
	}
	if cfg.MySQLDSN == "" {
		if value, err := g.Cfg().Get(ctx, "memory_store.dsn"); err == nil {
			cfg.MySQLDSN = strings.TrimSpace(value.String())
		}
	}

	if value, err := g.Cfg().Get(ctx, "file_dir"); err == nil {
		fileDir := strings.TrimSpace(value.String())
		if fileDir != "" {
			cfg.Sources = append(cfg.Sources, SourceConfig{
				Name: "local_docs",
				Type: "local",
				URI:  fileDir,
			})
		}
	}
	return cfg
}

func StartFromConfig(ctx context.Context) error {
	cfg := LoadConfig(ctx)
	if !cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(cfg.MySQLDSN) == "" {
		return nil
	}
	store, err := NewMySQLMetadataStore(cfg.MySQLDSN)
	if err != nil {
		return err
	}
	New(cfg, store).Start(ctx)
	return nil
}
