package memory_chat

import "context"

type ChatModel interface {
	Generate(ctx context.Context, prompt string) (string, error)
}
