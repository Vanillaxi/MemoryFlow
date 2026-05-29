package memory_chat

import (
	"context"
	"strings"

	"memoryflow/internal/ai/component/retriever"
)

func (p *Pipeline) retrieve(ctx context.Context, state *ChatState) (*ChatState, error) {
	question := strings.TrimSpace(state.Input.Question)
	if question == "" {
		state.Answer = "问题不能为空。"
		return state, nil
	}

	topK := state.Input.TopK
	if topK <= 0 {
		topK = 20
	}

	memories, err := p.memoryRetriever.Retrieve(ctx, question, retriever.RetrieveOptions{
		TopK:      topK,
		Type:      state.Input.Type,
		StartTime: state.Input.StartTime,
		EndTime:   state.Input.EndTime,
	})
	if err != nil {
		return nil, err
	}

	state.Retrieved = memories
	return state, nil
}
