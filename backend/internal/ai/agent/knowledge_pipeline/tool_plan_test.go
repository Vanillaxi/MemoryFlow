package knowledge_pipeline

import (
	"testing"

	"memoryflow/internal/ai/agent/dispatcher"
	webtool "memoryflow/internal/ai/tools/web"
)

func TestBuildToolCallsUsesWebFetchForURL(t *testing.T) {
	calls := (&Pipeline{}).BuildToolCalls(dispatcher.IntentExternalKnowledge, "读取 https://example.com/docs 这页资料")
	if len(calls) != 1 || calls[0].Name != webtool.ToolWebFetch || calls[0].Args["url"] != "https://example.com/docs" {
		t.Fatalf("unexpected calls: %#v", calls)
	}
}

func TestBuildToolCallsUsesWebSearchForExternalKnowledge(t *testing.T) {
	calls := (&Pipeline{}).BuildToolCalls(dispatcher.IntentExternalKnowledge, "搜索 Gin 官方文档")
	if len(calls) != 1 || calls[0].Name != webtool.ToolWebSearch || calls[0].Args["query"] != "搜索 Gin 官方文档" {
		t.Fatalf("unexpected calls: %#v", calls)
	}
}
