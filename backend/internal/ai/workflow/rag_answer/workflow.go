package rag_answer

import (
	"context"
	"strings"
	"time"

	"memoryflow/internal/ai/reranker"
	"memoryflow/internal/ai/retriever"
)

type ChatModel interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type RAGAnswerResult struct {
	Answer     string            `json:"answer"`
	References []MemoryReference `json:"references"`
}

type MemoryReference struct {
	ID         uint    `json:"id"`
	Summary    string  `json:"summary"`
	Content    string  `json:"content,omitempty"`
	ImageURL   string  `json:"image_url,omitempty"`
	OccurredAt string  `json:"occurred_at,omitempty"`
	Location   string  `json:"location,omitempty"`
	Mood       string  `json:"mood,omitempty"`
	Score      float32 `json:"score"`
}

type RAGAnswerWorkflow struct {
	memoryRetriever *retriever.MemoryRetriever
	memoryReranker  *reranker.MemoryReranker
	chatModel       ChatModel
}

func NewRAGAnswerWorkflow(
	memoryRetriever *retriever.MemoryRetriever,
	memoryReranker *reranker.MemoryReranker,
	chatModel ChatModel,
) *RAGAnswerWorkflow {
	return &RAGAnswerWorkflow{
		memoryRetriever: memoryRetriever,
		memoryReranker:  memoryReranker,
		chatModel:       chatModel,
	}
}

func (w *RAGAnswerWorkflow) Answer(ctx context.Context, question string) (*RAGAnswerResult, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return &RAGAnswerResult{
			Answer:     "问题不能为空。",
			References: []MemoryReference{},
		}, nil
	}

	// 1. 先多召回一点，给 rerank 留空间
	memories, err := w.memoryRetriever.Retrieve(ctx, question, retriever.RetrieveOptions{
		TopK: 20,
	})
	if err != nil {
		return nil, err
	}

	// 2. rerank 后取前 5 条
	memories = w.memoryReranker.Rerank(question, memories, 5)

	// 3. 没有相关记忆
	if len(memories) == 0 {
		return &RAGAnswerResult{
			Answer:     "抱歉，我没有在已有记忆中找到足够依据来回答这个问题。",
			References: []MemoryReference{},
		}, nil
	}

	// 4. 构建上下文
	memoryContext := BuildMemoryContext(memories)

	// 5. 构建 Prompt
	prompt := BuildRAGAnswerPrompt(question, memoryContext)

	// 6. 调用 LLM
	answer, err := w.chatModel.Generate(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return &RAGAnswerResult{
		Answer:     strings.TrimSpace(answer),
		References: buildReferences(memories),
	}, nil
}

func buildReferences(memories []retriever.RetrievedMemory) []MemoryReference {
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
