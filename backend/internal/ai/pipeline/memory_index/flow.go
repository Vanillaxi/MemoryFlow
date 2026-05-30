package memory_index

import (
	"context"

	"memoryflow/internal/domain/model"
)

type Pipeline struct {
	memoryService MemoryService
	indexer       DocumentIndexer
}

type MemoryService interface {
	ListForIndex(ctx context.Context, limit int, offset int) ([]model.MemoryItem, error)
}

type DocumentIndexer interface {
	Index(ctx context.Context, doc IndexDocument) error
}

func NewPipeline(
	memoryService MemoryService,
	indexer DocumentIndexer,
) *Pipeline {
	return &Pipeline{
		memoryService: memoryService,
		indexer:       indexer,
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
		items, err := p.memoryService.ListForIndex(ctx, batchSize, offset)
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
