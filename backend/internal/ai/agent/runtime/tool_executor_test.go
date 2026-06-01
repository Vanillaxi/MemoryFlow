package runtime

import (
	"context"
	"strings"
	"testing"

	"memoryflow/internal/ai/tools"
)

type fakeTool struct {
	name   string
	result string
}

func (f fakeTool) Name() string        { return f.name }
func (f fakeTool) Description() string { return f.name }
func (f fakeTool) Call(context.Context, map[string]any) (string, error) {
	return f.result, nil
}

func TestToolExecutorRunsAndTruncatesTools(t *testing.T) {
	registry := tools.NewToolRegistry()
	registry.Register(fakeTool{name: "long", result: strings.Repeat("x", maxToolResultLength+1)})

	logs, usedTools, err := NewToolExecutor(registry).Execute(context.Background(), []ToolCall{{Name: "long"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || len(usedTools) != 1 || !strings.HasSuffix(logs[0].Result, "...(truncated)") {
		t.Fatalf("logs=%#v used_tools=%#v", logs, usedTools)
	}
}
