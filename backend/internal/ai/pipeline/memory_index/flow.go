package memory_index

import (
	"context"

	"memoryflow/internal/domain/service"
)

type Pipeline struct {
	memoryService *service.MemoryService
	indexer       *Indexer
}

func NewPipeline(
	memoryService *service.MemoryService,
	indexer *Indexer,
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
