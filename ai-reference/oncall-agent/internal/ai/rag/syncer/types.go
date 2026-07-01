// Package syncer provides scheduled incremental RAG document indexing.
package syncer

import "time"

type Config struct {
	Enabled       bool
	Interval      time.Duration
	Sources       []SourceConfig
	MySQLDSN      string
	MaxFileSizeMB int64
}

type SourceConfig struct {
	Name string
	Type string
	URI  string
}

type DocumentMetadata struct {
	ID             uint   `gorm:"primaryKey"`
	SourceName     string `gorm:"size:128;index:idx_rag_doc_source_path,unique"`
	SourceType     string `gorm:"size:64"`
	Path           string `gorm:"size:1024;index:idx_rag_doc_source_path,unique"`
	ContentHash    string `gorm:"size:64;index"`
	FileSize       int64
	ModifiedAt     time.Time
	LastIndexedAt  time.Time
	LastStatus     string `gorm:"size:32"`
	LastError      string `gorm:"type:text"`
	IndexedVersion int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type SyncResult struct {
	Scanned int
	Skipped int
	Indexed int
	Failed  int
	Errors  []error
}
