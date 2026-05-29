package memory_chat

import (
	"context"
	"strings"
	"time"

	"memoryflow/internal/ai/component/retriever"
)

func (p *Pipeline) buildContext(ctx context.Context, state *ChatState) (*ChatState, error) {
	if state.Answer != "" {
		return state, nil
	}

	if len(state.Reranked) == 0 {
		state.Answer = "抱歉，我没有在已有记忆中找到足够依据来回答这个问题。"
		return state, nil
	}

	state.MemoryContext = BuildMemoryContext(state.Reranked)
	return state, nil
}

func (p *Pipeline) buildPrompt(ctx context.Context, state *ChatState) (*ChatState, error) {
	if state.Answer != "" {
		return state, nil
	}

	state.Prompt = BuildAnswerPrompt(state.Input.Question, state.MemoryContext)
	return state, nil
}

func (p *Pipeline) generate(ctx context.Context, state *ChatState) (*ChatState, error) {
	if state.Answer != "" {
		return state, nil
	}

	answer, err := p.chatModel.Generate(ctx, state.Prompt)
	if err != nil {
		return nil, err
	}

	state.Answer = strings.TrimSpace(answer)
	return state, nil
}

func toOutput(state *ChatState) *ChatOutput {
	return &ChatOutput{
		Answer:     state.Answer,
		References: buildReferences(state.Reranked),
	}
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
