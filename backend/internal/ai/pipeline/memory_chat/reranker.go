package memory_chat

import (
	"context"
	"strings"
)

func (p *Pipeline) rerank(ctx context.Context, state *ChatState) (*ChatState, error) {
	if state.Answer != "" {
		return state, nil
	}

	question := strings.TrimSpace(state.Input.Question)
	state.Reranked = p.memoryReranker.Rerank(question, state.Retrieved, 5)

	return state, nil
}
