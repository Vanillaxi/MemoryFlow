package memory_agent

import (
	"context"

	"memoryflow/internal/ai/pipelines/memory_chat_pipeline"
	"memoryflow/internal/ai/retriever"
	"memoryflow/internal/service"
)

type MemoryAgent struct {
	chatPipeline    *memory_chat_pipeline.Pipeline
	memoryRetriever *retriever.MemoryRetriever
	memoryService   *service.MemoryService
}

func NewMemoryAgent(
	chatPipeline *memory_chat_pipeline.Pipeline,
	memoryRetriever *retriever.MemoryRetriever,
	memoryService *service.MemoryService,
) *MemoryAgent {
	return &MemoryAgent{
		chatPipeline:    chatPipeline,
		memoryRetriever: memoryRetriever,
		memoryService:   memoryService,
	}
}

func (a *MemoryAgent) Chat(ctx context.Context, input ChatInput) (*ChatOutput, error) {
	return a.Orchestrate(ctx, input)
}
