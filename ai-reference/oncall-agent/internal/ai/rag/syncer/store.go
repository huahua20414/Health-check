package syncer

import (
	"context"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MetadataStore interface {
	GetBySourcePath(ctx context.Context, sourceName, path string) (*DocumentMetadata, error)
	Upsert(ctx context.Context, doc *DocumentMetadata) error
}

type MySQLMetadataStore struct {
	db *gorm.DB
}

func NewMySQLMetadataStore(dsn string) (*MySQLMetadataStore, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&DocumentMetadata{}); err != nil {
		return nil, err
	}
	return &MySQLMetadataStore{db: db}, nil
}

func (s *MySQLMetadataStore) GetBySourcePath(ctx context.Context, sourceName, path string) (*DocumentMetadata, error) {
	var doc DocumentMetadata
	err := s.db.WithContext(ctx).
		Where("source_name = ? AND path = ?", sourceName, path).
		First(&doc).Error
	if err == nil {
		return &doc, nil
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return nil, err
}

func (s *MySQLMetadataStore) Upsert(ctx context.Context, doc *DocumentMetadata) error {
	existing, err := s.GetBySourcePath(ctx, doc.SourceName, doc.Path)
	if err != nil {
		return err
	}
	if existing != nil {
		doc.ID = existing.ID
		doc.CreatedAt = existing.CreatedAt
	}
	return s.db.WithContext(ctx).Save(doc).Error
}
