package web

import (
	"context"
	"errors"
	"testing"
)

func TestWebSearchRequiresProvider(t *testing.T) {
	_, err := NewWebSearchTool(nil).Call(context.Background(), map[string]any{"query": "MemoryFlow docs"})
	if !errors.Is(err, ErrWebSearchProviderNotConfigured) {
		t.Fatalf("err=%v, want ErrWebSearchProviderNotConfigured", err)
	}
}
