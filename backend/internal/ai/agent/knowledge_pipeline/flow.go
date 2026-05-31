package knowledge_pipeline

import (
	"context"

	"memoryflow/internal/ai/indexer"
	"memoryflow/internal/domain/model"
)

type Pipeline struct {
	memoryLoader MemoryLoader
	indexer      DocumentIndexer
}

type MemoryLoader interface {
	LoadForIndex(ctx context.Context, limit int, offset int) ([]model.MemoryItem, error)
}

type DocumentIndexer interface {
	Index(ctx context.Context, doc indexer.IndexDocument) error
}

func NewPipeline(
	memoryLoader MemoryLoader,
	indexer DocumentIndexer,
) *Pipeline {
	return &Pipeline{
		memoryLoader: memoryLoader,
		indexer:      indexer,
	}
}
