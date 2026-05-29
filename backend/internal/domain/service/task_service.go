package service

import (
	"context"
	"memoryflow/internal/domain/model"
	"memoryflow/internal/domain/repository"
)

const (
	TaskStatusPending = "pending"
	TaskStatusRunning = "running"
	TaskStatusSuccess = "success"
	TaskStatusFailed  = "failed"

	TaskTypeTextAnalyze    = "text_analyze"
	TaskTypeImageAnalyze   = "image_analyze"
	TaskTypeEmbedding      = "embedding"
	TaskTypeReportGenerate = "report_generate"
)

type TaskService struct {
	repo repository.TaskRepository
}

func NewTaskService(repo repository.TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

func (s *TaskService) CreateTask(ctx context.Context, taskType string, targetID uint) (*model.Task, error) {
	task := &model.Task{
		TaskType: taskType,
		Status:   TaskStatusPending,
		TargetID: targetID,
	}

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}

	return task, nil

}

func (s *TaskService) GetTask(ctx context.Context, id uint) (*model.Task, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *TaskService) UpdateStatus(ctx context.Context, id uint, status string, errorMessage string) error {
	return s.repo.UpdateStatus(ctx, id, status, errorMessage)
}

func (s *TaskService) FindPending(ctx context.Context, limit int) ([]model.Task, error) {
	return s.repo.FindPending(ctx, limit)
}
