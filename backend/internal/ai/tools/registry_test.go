package tools

import (
	"context"
	"testing"
)

type fakeTool struct{}

func (fakeTool) Name() string        { return "fake" }
func (fakeTool) Description() string { return "fake tool" }
func (fakeTool) Call(context.Context, map[string]any) (string, error) {
	return "ok", nil
}

func TestToolRegistryRegistersGetsAndListsTools(t *testing.T) {
	registry := NewToolRegistry()
	registry.Register(fakeTool{})

	got, ok := registry.Get("fake")
	if !ok || got.Name() != "fake" {
		t.Fatalf("unexpected tool: %#v ok=%v", got, ok)
	}
	if tools := registry.List(); len(tools) != 1 || tools[0].Description() == "" {
		t.Fatalf("unexpected tools: %#v", tools)
	}
}
