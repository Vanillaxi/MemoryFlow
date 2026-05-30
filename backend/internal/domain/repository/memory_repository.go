package repository

import (
	"context"
	"memoryflow/internal/domain/model"
	"time"

	"gorm.io/gorm"
)

type MemoryRepository interface {
	Create(ctx context.Context, item *model.MemoryItem) error
	FindByID(ctx context.Context, id uint) (*model.MemoryItem, error)
	FindByIDs(ctx context.Context, ids []uint) ([]model.MemoryItem, error)
	FindRecent(ctx context.Context, limit int) ([]model.MemoryItem, error)
	FindByTimeRange(ctx context.Context, start, end time.Time, limit int) ([]model.MemoryItem, error)
	ListByTimeRange(ctx context.Context, from, to time.Time, limit int) ([]*model.MemoryItem, error)
	UpdateAnalysis(ctx context.Context, id uint, summary string, tags string, mood string, importanceScore float64) error
	ListForIndex(ctx context.Context, limit int, offset int) ([]model.MemoryItem, error)
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

// 从新到旧排序
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

// ListByTimeRange returns fact records ordered by occurrence time from oldest to newest.
func (r *SQLiteMemoryRepository) ListByTimeRange(ctx context.Context, from, to time.Time, limit int) ([]*model.MemoryItem, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	var items []*model.MemoryItem
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where("occurred_at >= ? AND occurred_at <= ?", from, to).
		Order("occurred_at ASC").
		Limit(limit).
		Find(&items).Error
	return items, err
}

func (r *SQLiteMemoryRepository) FindByID(ctx context.Context, id uint) (*model.MemoryItem, error) {
	var item model.MemoryItem
	if err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *SQLiteMemoryRepository) FindByIDs(ctx context.Context, ids []uint) ([]model.MemoryItem, error) {
	if len(ids) == 0 {
		return []model.MemoryItem{}, nil
	}

	var items []model.MemoryItem
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where("id IN (?)", ids).
		Find(&items).Error
	if err != nil {
		return nil, err
	}

	itemMap := make(map[uint]model.MemoryItem, len(items))
	for _, item := range items {
		itemMap[item.ID] = item
	}

	//按照Milvus的顺序重新排一遍
	ordered := make([]model.MemoryItem, 0, len(items))
	for _, id := range ids {
		if item, ok := itemMap[id]; ok {
			ordered = append(ordered, item)
		}
	}

	return ordered, nil
}

func (r *SQLiteMemoryRepository) UpdateAnalysis(ctx context.Context, id uint, summary string, tags string, mood string, importanceScore float64) error {
	return r.db.WithContext(ctx).
		Model(&model.MemoryItem{}).
		Where("id=? AND deleted_at IS NULL", id).
		Updates(map[string]any{
			"summary":          summary,
			"tags":             tags,
			"mood":             mood,
			"importance_score": importanceScore,
		}).Error
}

func (r *SQLiteMemoryRepository) ListForIndex(ctx context.Context, limit int, offset int) ([]model.MemoryItem, error) {
	var items []model.MemoryItem

	if limit <= 0 {
		limit = 100
	}

	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("id ASC").
		Limit(limit).
		Offset(offset).
		Find(&items).Error

	return items, err
}
