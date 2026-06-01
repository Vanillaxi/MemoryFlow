package repository

import (
	"context"

	"memoryflow/internal/domain/model"

	"gorm.io/gorm"
)

type ProjectRepository interface {
	Create(ctx context.Context, project *model.Project) error
	List(ctx context.Context) ([]model.Project, error)
	FindByID(ctx context.Context, id uint) (*model.Project, error)
	FindByName(ctx context.Context, name string) (*model.Project, error)
}

type SQLiteProjectRepository struct {
	db *gorm.DB
}

func NewSQLiteProjectRepository(db *gorm.DB) *SQLiteProjectRepository {
	return &SQLiteProjectRepository{db: db}
}

func (r *SQLiteProjectRepository) Create(ctx context.Context, project *model.Project) error {
	return r.db.WithContext(ctx).Create(project).Error
}

func (r *SQLiteProjectRepository) List(ctx context.Context) ([]model.Project, error) {
	var projects []model.Project
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&projects).Error
	return projects, err
}

func (r *SQLiteProjectRepository) FindByID(ctx context.Context, id uint) (*model.Project, error) {
	var project model.Project
	if err := r.db.WithContext(ctx).First(&project, id).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *SQLiteProjectRepository) FindByName(ctx context.Context, name string) (*model.Project, error) {
	var project model.Project
	if err := r.db.WithContext(ctx).Where("LOWER(name) = LOWER(?)", name).First(&project).Error; err != nil {
		return nil, err
	}
	return &project, nil
}
