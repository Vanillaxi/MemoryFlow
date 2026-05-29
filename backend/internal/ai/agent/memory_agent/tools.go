package memory_agent

import (
	"context"
	"strings"
	"time"

	"memoryflow/internal/ai/component/retriever"
	"memoryflow/internal/ai/pipeline/memory_chat"
	"memoryflow/internal/domain/model"
)

type ToolName string

const (
	ToolAskMemory    ToolName = "ask_memory"
	ToolSearchMemory ToolName = "search_memory"
	ToolListRecent   ToolName = "list_recent"
	ToolGetTimeline  ToolName = "get_timeline"
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

func (a *MemoryAgent) AskMemory(ctx context.Context, input AskMemoryInput) (*memory_chat.ChatOutput, error) {
	question := strings.TrimSpace(input.Question)
	if question == "" {
		return &memory_chat.ChatOutput{
			Answer: "问题不能为空。",
		}, nil
	}

	return a.chatPipeline.Run(ctx, memory_chat.ChatInput{
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
