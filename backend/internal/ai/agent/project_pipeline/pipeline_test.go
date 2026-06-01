package project_pipeline

import (
	"testing"

	"memoryflow/internal/ai/agent/dispatcher"
	githubtool "memoryflow/internal/ai/tools/github"
	memorytool "memoryflow/internal/ai/tools/memory"
)

func TestBuildToolCallsForProjectProgress(t *testing.T) {
	calls := NewPipeline().BuildToolCalls(dispatcher.IntentProjectProgress, "MemoryFlow 最近做到哪了？")
	if len(calls) != 3 || calls[2].Name != githubtool.ToolGetRecentCommits {
		t.Fatalf("unexpected calls: %#v", calls)
	}
}

func TestBuildToolCallsForHandoff(t *testing.T) {
	calls := NewPipeline().BuildToolCalls(dispatcher.IntentHandoff, "生成交接")
	if len(calls) != 2 || calls[1].Name != memorytool.ToolAggregateMemory {
		t.Fatalf("unexpected calls: %#v", calls)
	}
}
