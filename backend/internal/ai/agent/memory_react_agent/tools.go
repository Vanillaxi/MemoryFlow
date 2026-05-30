package memory_react_agent

import (
	"context"
	"strings"
	"time"

	"memoryflow/internal/ai/agent/memory_chat_pipeline"
	"memoryflow/internal/ai/agent/memory_summary_pipeline"
	"memoryflow/internal/ai/retriever"
	memoryagenttools "memoryflow/internal/ai/tools"
	"memoryflow/internal/domain/model"
	"memoryflow/internal/domain/service"
)

type ToolName string

const (
	ToolAskMemory           ToolName = "ask_memory"
	ToolSearchMemory        ToolName = "search_memory"
	ToolListRecent          ToolName = "list_recent"
	ToolGetTimeline         ToolName = "get_timeline"
	ToolSummarizeMemory     ToolName = "summarize_memory"
	ToolGetCurrentTime      ToolName = "get_current_time"
	ToolQueryLongTermMemory ToolName = "query_long_term_memory"
	ToolGetMemoryDetail     ToolName = "get_memory_detail"

	longTermMemoryModeSemantic  = memoryagenttools.ModeSemantic
	longTermMemoryModeTimeline  = memoryagenttools.ModeTimeline
	longTermMemoryModeAggregate = memoryagenttools.ModeAggregate
	maxMemorySummaryLength      = memoryagenttools.MaxSummaryLength
)

type AskMemoryInput struct {
	Question  string
	TopK      int
	Type      string
	StartTime *time.Time
	EndTime   *time.Time
}

type SearchMemoryInput struct {
	Query     string
	TopK      int
	Type      string
	StartTime *time.Time
	EndTime   *time.Time
}

type RecentMemoryInput struct {
	Limit int
}

type TimelineInput struct {
	Start time.Time
	End   time.Time
}

type SummarizeMemoryInput struct {
	Start time.Time
	End   time.Time
	Limit int
}

type QueryLongTermMemoryInput = memoryagenttools.QueryLongTermMemoryInput

type GetMemoryDetailInput = memoryagenttools.GetMemoryDetailInput

type CurrentTimeOutput = memoryagenttools.CurrentTimeOutput

type LongTermMemoryItem = memoryagenttools.LongTermMemoryItem

type LongTermMemoryEvidence = memoryagenttools.LongTermMemoryEvidence

type LongTermMemoryAggregation = memoryagenttools.LongTermMemoryAggregation

type QueryLongTermMemoryOutput = memoryagenttools.QueryLongTermMemoryOutput

type ChatPipeline interface {
	Run(ctx context.Context, input memory_chat_pipeline.ChatInput) (*memory_chat_pipeline.ChatOutput, error)
}

type MemoryRetriever interface {
	Retrieve(ctx context.Context, query string, opt retriever.RetrieveOptions) ([]retriever.RetrievedMemory, error)
}

type MemoryService interface {
	ListRecent(ctx context.Context, limit int) ([]model.MemoryItem, error)
	GetTimeline(ctx context.Context, start, end time.Time) ([]service.TimelineGroup, error)
	ListByTimeRange(ctx context.Context, from, to time.Time, limit int) ([]*model.MemoryItem, error)
	GetByID(ctx context.Context, id uint) (*model.MemoryItem, error)
}

type SummaryPipeline interface {
	Invoke(ctx context.Context, input memory_summary_pipeline.SummaryInput) (*memory_summary_pipeline.SummaryOutput, error)
}

func (a *MemoryAgent) AskMemory(ctx context.Context, input AskMemoryInput) (*memory_chat_pipeline.ChatOutput, error) {
	question := strings.TrimSpace(input.Question)
	if question == "" {
		return &memory_chat_pipeline.ChatOutput{
			Answer: "问题不能为空。",
		}, nil
	}

	return a.chatPipeline.Run(ctx, memory_chat_pipeline.ChatInput{
		Question:  question,
		TopK:      input.TopK,
		Type:      input.Type,
		StartTime: input.StartTime,
		EndTime:   input.EndTime,
	})
}

func (a *MemoryAgent) SearchMemory(ctx context.Context, input SearchMemoryInput) ([]retriever.RetrievedMemory, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return []retriever.RetrievedMemory{}, nil
	}

	topK := input.TopK
	if topK <= 0 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}

	return a.memoryRetriever.Retrieve(ctx, query, retriever.RetrieveOptions{
		TopK:      topK,
		Type:      input.Type,
		StartTime: input.StartTime,
		EndTime:   input.EndTime,
	})
}

func (a *MemoryAgent) ListRecent(ctx context.Context, input RecentMemoryInput) ([]model.MemoryItem, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return a.memoryService.ListRecent(ctx, limit)
}

func (a *MemoryAgent) GetTimeline(ctx context.Context, input TimelineInput) (any, error) {
	return a.memoryService.GetTimeline(ctx, input.Start, input.End)
}

func (a *MemoryAgent) SummarizeMemory(ctx context.Context, input SummarizeMemoryInput) (*memory_summary_pipeline.SummaryOutput, error) {
	return a.summaryPipeline.Invoke(ctx, memory_summary_pipeline.SummaryInput{
		From:  input.Start,
		To:    input.End,
		Limit: input.Limit,
	})
}

func (a *MemoryAgent) GetCurrentTime() CurrentTimeOutput {
	return memoryagenttools.GetCurrentTime()
}

func (a *MemoryAgent) QueryLongTermMemory(ctx context.Context, input QueryLongTermMemoryInput) (*QueryLongTermMemoryOutput, error) {
	return memoryagenttools.QueryLongTermMemory(ctx, a.memoryRetriever, a.memoryService, input)
}

func (a *MemoryAgent) GetMemoryDetail(ctx context.Context, input GetMemoryDetailInput) (*model.MemoryItem, error) {
	return memoryagenttools.GetMemoryDetail(ctx, a.memoryService, input)
}
