package memory_chat

import (
	"context"
	"errors"

	"github.com/cloudwego/eino/compose"

	"memoryflow/internal/ai/component/retriever"
)

type Pipeline struct {
	memoryRetriever MemoryRetriever
	memoryReranker  MemoryReranker
	chatModel       ChatModel

	runnable compose.Runnable[ChatInput, *ChatOutput]
}

type MemoryRetriever interface {
	Retrieve(ctx context.Context, query string, opt retriever.RetrieveOptions) ([]retriever.RetrievedMemory, error)
}

type MemoryReranker interface {
	Rerank(query string, memories []retriever.RetrievedMemory, topK int) []retriever.RetrievedMemory
}

func NewPipeline(
	ctx context.Context,
	memoryRetriever MemoryRetriever,
	memoryReranker MemoryReranker,
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
