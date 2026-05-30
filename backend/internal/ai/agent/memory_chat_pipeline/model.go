package memory_chat_pipeline

import "context"

type ChatModel interface {
	Generate(ctx context.Context, prompt string) (string, error)
}
