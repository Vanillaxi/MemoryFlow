package knowledge_pipeline

import "context"

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
			if err := p.indexer.Index(ctx, ToIndexDocument(item)); err != nil {
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
