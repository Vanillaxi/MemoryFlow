package memory_summary

import "context"

type ChatModel interface {
	Generate(ctx context.Context, prompt string) (string, error)
}
