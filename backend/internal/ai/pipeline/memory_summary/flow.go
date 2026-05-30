package memory_summary

import (
	"context"
	"errors"
	"strings"
	"time"

	"memoryflow/internal/domain/model"
)

const defaultLimit = 100

type MemoryService interface {
	ListByTimeRange(ctx context.Context, from, to time.Time, limit int) ([]*model.MemoryItem, error)
}

type Pipeline struct {
	memoryService MemoryService
	chatModel     ChatModel
}

func NewPipeline(memoryService MemoryService, chatModel ChatModel) *Pipeline {
	return &Pipeline{
		memoryService: memoryService,
		chatModel:     chatModel,
	}
}

func (p *Pipeline) Invoke(ctx context.Context, input SummaryInput) (*SummaryOutput, error) {
	if input.From.IsZero() || input.To.IsZero() {
		return nil, errors.New("summary from and to are required")
	}
	if input.To.Before(input.From) {
		return nil, errors.New("summary to must not be before from")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > 500 {
		limit = 500
	}

	memories, err := p.memoryService.ListByTimeRange(ctx, input.From, input.To, limit)
	if err != nil {
		return nil, err
	}

	aggregation := AggregateMemories(memories)
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
