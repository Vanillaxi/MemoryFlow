package chat_pipeline

import (
	"context"
	"errors"
	"strings"
	"testing"

	memorytools "memoryflow/internal/ai/tools"

	"github.com/cloudwego/eino/components/tool"
)

func TestBaseToolsOnlyExposeExternalActions(t *testing.T) {
	tools := (&Pipeline{}).BaseTools()
	want := []string{memorytools.ToolGetCurrentTime, memorytools.ToolQueryLongTermMemory, memorytools.ToolGetMemoryDetail}
	if len(tools) != len(want) {
		t.Fatalf("len(BaseTools()) = %d, want %d", len(tools), len(want))
	}
	for i, currentTool := range tools {
		info, err := currentTool.Info(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if info.Name != want[i] {
			t.Fatalf("tool[%d] = %q, want %q", i, info.Name, want[i])
		}
	}
}

func TestTraceCollectorRecordsStepsAndErrors(t *testing.T) {
	collector := NewTraceCollector("test_mode")
	collector.Start("node", map[string]any{"input": "value"})
	collector.End("node", map[string]any{"output": "value"})
	collector.Error("node", errors.New("boom"))

	trace := collector.Trace()
	if trace.Mode != "test_mode" || len(trace.Steps) != 3 || trace.Error != "boom" {
		t.Fatalf("unexpected trace: %#v", trace)
	}
}

func TestTraceSanitizesSensitiveValues(t *testing.T) {
	got := sanitizeTraceJSON(`{"api_key":"key","authorization":"bearer","nested":{"token":"value","secret":"hidden"},"ok":"visible"}`)
	for _, unwanted := range []string{`"key"`, `"bearer"`, `"value"`, `"hidden"`} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("trace leaked %q: %s", unwanted, got)
		}
	}
	if !strings.Contains(got, `"ok":"visible"`) {
		t.Fatalf("trace lost safe value: %s", got)
	}
}

func TestToolTraceUsesExternalToolName(t *testing.T) {
	collector := NewTraceCollector("test")
	ctx := ContextWithTraceCollector(context.Background(), collector)

	if _, err := (&Pipeline{}).BaseTools()[0].(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	}).InvokableRun(ctx, `{}`); err != nil {
		t.Fatal(err)
	}

	steps := collector.Trace().Steps
	if len(steps) != 2 || steps[0].Node != memorytools.ToolGetCurrentTime || steps[0].Event != "tool_start" || steps[1].Event != "tool_end" {
		t.Fatalf("unexpected trace: %#v", steps)
	}
}
