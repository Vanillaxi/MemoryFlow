package repository

import (
	"context"
	"memoryflow/internal/model"

	"gorm.io/gorm"
)

type TaskRepository interface {
	Create(ctx context.Context, task *model.Task) error
	FindByID(ctx context.Context, id uint) (*model.Task, error)
	UpdateStatus(ctx context.Context, id uint, status string, errorMessage string) error
	FindPending(ctx context.Context, limit int) ([]model.Task, error)
}

type SQLiteTaskRepository struct {
	db *gorm.DB
}

func NewSQLiteTaskRepository(db *gorm.DB) *SQLiteTaskRepository {
	return &SQLiteTaskRepository{db: db}
}

func (r *SQLiteTaskRepository) Create(ctx context.Context, task *model.Task) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *SQLiteTaskRepository) FindByID(ctx context.Context, id uint) (*model.Task, error) {
	var task model.Task
	if err := r.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *SQLiteTaskRepository) UpdateStatus(ctx context.Context, id uint, status string, errorMessage string) error {
	return r.db.WithContext(ctx).
		Model(&model.Task{}).
		Where("id=?", id).
		Updates(map[string]any{
			"status":        status,
			"error_message": errorMessage,
		}).Error
}

func (r *SQLiteTaskRepository) FindPending(ctx context.Context, limit int) ([]model.Task, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var tasks []model.Task
	err := r.db.WithContext(ctx).
		Where("status=?", "pending").
		Order("created_at ASC").
		Limit(limit).
		Find(&tasks).Error

	return tasks, err
}
