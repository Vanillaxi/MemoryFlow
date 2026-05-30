package memory_index_pipeline

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

func (p *Pipeline) ReindexAll(ctx context.Context, input ReindexInput) (*ReindexOutput, error) {
	batchSize := input.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}
	if batchSize > 500 {
		batchSize = 500
	}

	output := &ReindexOutput{}

	offset := 0

	for {
		items, err := p.memoryLoader.LoadForIndex(ctx, batchSize, offset)
		if err != nil {
			return nil, err
		}

		if len(items) == 0 {
			break
		}

		for _, item := range items {
			output.Total++

			doc := ToIndexDocument(item)
			if err := p.indexer.Index(ctx, doc); err != nil {
				output.Failed++
				continue
			}

			output.Succeeded++
		}

		offset += len(items)

		if len(items) < batchSize {
			break
		}
	}

	return output, nil
}
