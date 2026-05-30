package tools

import (
	"context"
	"fmt"

	"memoryflow/internal/domain/model"
)

func GetMemoryDetail(ctx context.Context, memoryService MemoryService, input GetMemoryDetailInput) (*model.MemoryItem, error) {
	if input.MemoryID == 0 {
		return nil, fmt.Errorf("memory_id is required")
	}
	return memoryService.GetByID(ctx, input.MemoryID)
}
