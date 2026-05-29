package memory_agent

import (
	"context"

	"memoryflow/internal/ai/pipelines/memory_chat_pipeline"
	"memoryflow/internal/ai/retriever"
	"memoryflow/internal/service"
)

type ChatModel interface {
	Generate(ctx context.Context, prompt string) (string, error)
	GenerateWithSystem(ctx context.Context, systemPrompt string, userPrompt string) (string, error)
}
type MemoryAgent struct {
	chatPipeline    *memory_chat_pipeline.Pipeline
	memoryRetriever *retriever.MemoryRetriever
	memoryService   *service.MemoryService
	chatModel       ChatModel
}

func NewMemoryAgent(
	chatPipeline *memory_chat_pipeline.Pipeline,
	memoryRetriever *retriever.MemoryRetriever,
	memoryService *service.MemoryService,
	chatModel ChatModel,
) *MemoryAgent {
	return &MemoryAgent{
		chatPipeline:    chatPipeline,
		memoryRetriever: memoryRetriever,
		memoryService:   memoryService,
		chatModel:       chatModel,
	}
}

func (a *MemoryAgent) Chat(ctx context.Context, input ChatInput) (*ChatOutput, error) {
	return a.Orchestrate(ctx, input)
}
