package loader

import (
	"context"

	"memoryflow/internal/domain/model"
)

type MemoryService interface {
	ListForIndex(ctx context.Context, limit int, offset int) ([]model.MemoryItem, error)
}

type MemoryLoader struct {
	memoryService MemoryService
}

func NewMemoryLoader(memoryService MemoryService) *MemoryLoader {
	return &MemoryLoader{memoryService: memoryService}
}

func (l *MemoryLoader) LoadForIndex(ctx context.Context, limit int, offset int) ([]model.MemoryItem, error) {
	return l.memoryService.ListForIndex(ctx, limit, offset)
}
