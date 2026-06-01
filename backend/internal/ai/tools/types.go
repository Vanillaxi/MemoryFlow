package tools

import (
	"context"
)

type Tool interface {
	Name() string
	Description() string
	Call(ctx context.Context, args map[string]any) (string, error)
}
