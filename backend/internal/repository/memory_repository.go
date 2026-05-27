package repository

import (
	"context"
	"memoryflow/internal/model"
	"time"

	"gorm.io/gorm"
)

type MemoryRepository interface {
	Create(ctx context.Context, item *model.MemoryItem) error
	FindRecent(ctx context.Context, limit int) ([]model.MemoryItem, error)
	FindByTimeRange(ctx context.Context, start, end time.Time, limit int) ([]model.MemoryItem, error)
}

type SQLiteMemoryRepository struct {
	db *gorm.DB
}

func NewSQLiteMemoryRepository(db *gorm.DB) *SQLiteMemoryRepository {
	return &SQLiteMemoryRepository{db: db}
}

func (r *SQLiteMemoryRepository) Create(ctx context.Context, item *model.MemoryItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *SQLiteMemoryRepository) FindRecent(ctx context.Context, limit int) ([]model.MemoryItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var items []model.MemoryItem
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Limit(limit).
		Find(&items).Error

	return items, err

}

func (r *SQLiteMemoryRepository) FindByTimeRange(ctx context.Context, start, end time.Time, limit int) ([]model.MemoryItem, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	var items []model.MemoryItem
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where("created_at >= ? AND created_at < ?", start, end).
		Order("created_at DESC").
		Limit(limit).
		Find(&items).Error
	return items, err
}
