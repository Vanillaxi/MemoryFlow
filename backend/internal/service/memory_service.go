package service

import (
	"context"
	"memoryflow/internal/model"
	"memoryflow/internal/repository"
	"time"
)

type CreateTextMemoryRequest struct {
	ContentText string    `json:"content_text"`
	OccurredAt  time.Time `json:"occurred_at"`
	Location    string    `json:"location"`
}

type MemoryService struct {
	repo repository.MemoryRepository
}

func NewMemoryService(repo repository.MemoryRepository) *MemoryService {
	return &MemoryService{repo: repo}
}

func (s *MemoryService) CreateTextMemory(ctx context.Context, req *CreateTextMemoryRequest) (*model.MemoryItem, error) {
	occurredAt := req.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}

	item := &model.MemoryItem{
		Type:        "text",
		ContentText: req.ContentText,
		OccurredAt:  occurredAt,
		Location:    req.Location,
		Tags:        "[]",
	}

	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *MemoryService) ListRecent(ctx context.Context, limit int) ([]model.MemoryItem, error) {
	return s.repo.FindRecent(ctx, limit)
}
