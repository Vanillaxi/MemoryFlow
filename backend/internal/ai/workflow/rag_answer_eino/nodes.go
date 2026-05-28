package rag_answer_eino

import (
	"context"
	"strings"
	"time"

	"memoryflow/internal/ai/reranker"
	"memoryflow/internal/ai/retriever"
)

type Nodes struct {
	memoryRetriever *retriever.MemoryRetriever
	memoryReranker  *reranker.MemoryReranker
	chatModel       ChatModel
}

type ChatModel interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

func NewNodes(
	memoryRetriever *retriever.MemoryRetriever,
	memoryReranker *reranker.MemoryReranker,
	chatModel ChatModel,
) *Nodes {
	return &Nodes{
		memoryRetriever: memoryRetriever,
		memoryReranker:  memoryReranker,
		chatModel:       chatModel,
	}
}

func (n *Nodes) Retrieve(ctx context.Context, input RAGAnswerInput) ([]retriever.RetrievedMemory, error) {
	question := strings.TrimSpace(input.Question)
	if question == "" {
		return []retriever.RetrievedMemory{}, nil
	}

	topK := input.TopK
	if topK <= 0 {
		topK = 20
	}

	return n.memoryRetriever.Retrieve(ctx, question, retriever.RetrieveOptions{
		TopK:      topK,
		Type:      input.Type,
		StartTime: input.StartTime,
		EndTime:   input.EndTime,
	})
}

func (n *Nodes) Rerank(ctx context.Context, input RAGAnswerInput, memories []retriever.RetrievedMemory) []retriever.RetrievedMemory {
	question := strings.TrimSpace(input.Question)
	return n.memoryReranker.Rerank(question, memories, 5)
}

func (n *Nodes) BuildContext(memories []retriever.RetrievedMemory) string {
	return BuildMemoryContext(memories)
}

func (n *Nodes) BuildPrompt(question string, memoryContext string) string {
	return BuildRAGAnswerPrompt(question, memoryContext)
}

func (n *Nodes) Generate(ctx context.Context, prompt string) (string, error) {
	return n.chatModel.Generate(ctx, prompt)
}

func BuildReferences(memories []retriever.RetrievedMemory) []MemoryReference {
	refs := make([]MemoryReference, 0, len(memories))

	for _, item := range memories {
		memory := item.Memory

		content := truncateRunes(memory.ContentText, 120)

		ref := MemoryReference{
			ID:       memory.ID,
			Summary:  memory.Summary,
			Content:  content,
			ImageURL: memory.ImageURL,
			Location: memory.Location,
			Mood:     memory.Mood,
			Score:    item.Score,
		}

		if !memory.OccurredAt.IsZero() {
			ref.OccurredAt = memory.OccurredAt.Format(time.RFC3339)
		}

		refs = append(refs, ref)
	}

	return refs
}

func truncateRunes(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
