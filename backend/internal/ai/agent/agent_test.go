package agent

import (
	"context"
	"strings"
	"testing"

	memorytools "memoryflow/internal/ai/tools"
	githubtools "memoryflow/internal/ai/tools/github"
	memorytool "memoryflow/internal/ai/tools/memory"
	systemtool "memoryflow/internal/ai/tools/system"
)

type fakeTool struct {
	name   string
	result string
	err    error
}

func (f fakeTool) Name() string        { return f.name }
func (f fakeTool) Description() string { return f.name }
func (f fakeTool) Call(context.Context, map[string]any) (string, error) {
	return f.result, f.err
}

type fakeSummaryModel struct {
	prompt string
}

func (f *fakeSummaryModel) GenerateWithSystem(_ context.Context, _ string, prompt string) (string, error) {
	f.prompt = prompt
	return " 已完成项目进展总结。 ", nil
}

func TestChatProjectProgressUsesExpectedTools(t *testing.T) {
	registry := memorytools.NewToolRegistry()
	registry.Register(fakeTool{name: systemtool.ToolGetCurrentTime, result: `{"date":"2026-06-01"}`})
	registry.Register(fakeTool{name: memorytool.ToolQueryLongTermMemory, result: `{"evidence":[]}`})
	registry.Register(fakeTool{name: githubtools.ToolGetRecentCommits, result: `{"commits":[]}`})
	model := &fakeSummaryModel{}

	output, err := NewAgent(registry, model, nil).Chat(context.Background(), ChatInput{Message: "MemoryFlow 最近做到哪了？"})
	if err != nil {
		t.Fatal(err)
	}
	if output.Intent != "project_progress" || len(output.UsedTools) != 3 || output.UsedTools[2] != githubtools.ToolGetRecentCommits {
		t.Fatalf("unexpected output: %#v", output)
	}
	if strings.Contains(output.Answer, " ") || !strings.Contains(model.prompt, `commits`) {
		t.Fatalf("unexpected answer=%q prompt=%q", output.Answer, model.prompt)
	}
}
