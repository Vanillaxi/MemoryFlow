package memory_chat

import (
	"context"
	"errors"

	"github.com/cloudwego/eino/compose"

	"memoryflow/internal/ai/component/reranker"
	"memoryflow/internal/ai/component/retriever"
)

type Pipeline struct {
	memoryRetriever *retriever.MemoryRetriever
	memoryReranker  *reranker.MemoryReranker
	chatModel       ChatModel

	runnable compose.Runnable[ChatInput, *ChatOutput]
}

func NewPipeline(
	ctx context.Context,
	memoryRetriever *retriever.MemoryRetriever,
	memoryReranker *reranker.MemoryReranker,
	chatModel ChatModel,
) (*Pipeline, error) {
	p := &Pipeline{
		memoryRetriever: memoryRetriever,
		memoryReranker:  memoryReranker,
		chatModel:       chatModel,
	}

	if err := p.buildGraph(ctx); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *Pipeline) Run(ctx context.Context, input ChatInput) (*ChatOutput, error) {
	if p.runnable == nil {
		return nil, errors.New("memory chat pipeline is not initialized")
	}

	return p.runnable.Invoke(ctx, input)
}
