package chat_pipeline

import (
	"context"
	"errors"
	"strings"

	memorytools "memoryflow/internal/ai/tools"
)

const defaultSummaryLimit = 100

// Summarize keeps the existing HTTP summary behavior as a chat capability.
func (p *Pipeline) Summarize(ctx context.Context, input SummaryInput) (*SummaryOutput, error) {
	if input.From.IsZero() || input.To.IsZero() {
		return nil, errors.New("summary from and to are required")
	}
	if input.To.Before(input.From) {
		return nil, errors.New("summary to must not be before from")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = defaultSummaryLimit
	}
	if limit > 500 {
		limit = 500
	}

	memories, err := p.memoryService.ListByTimeRange(ctx, input.From, input.To, limit)
	if err != nil {
		return nil, err
	}

	aggregation := memorytools.AggregateMemories(memories)
	output := &SummaryOutput{
		From:       input.From,
		To:         input.To,
		Highlights: aggregation.Highlights,
		Tags:       aggregation.Tags,
		Moods:      aggregation.Moods,
		Count:      aggregation.Count,
	}
	if aggregation.Count == 0 {
		output.Summary = "这段时间没有记录到可供回顾的记忆。"
		return output, nil
	}

	summary, err := p.chatModel.Generate(ctx, BuildSummaryPrompt(input.From, input.To, aggregation))
	if err != nil {
		return nil, err
	}
	output.Summary = strings.TrimSpace(summary)
	return output, nil
}
