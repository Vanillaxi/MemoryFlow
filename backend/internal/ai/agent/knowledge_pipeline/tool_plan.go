package knowledge_pipeline

import (
	"regexp"
	"strings"

	"memoryflow/internal/ai/agent/dispatcher"
	agentruntime "memoryflow/internal/ai/agent/runtime"
	webtool "memoryflow/internal/ai/tools/web"
)

var urlPattern = regexp.MustCompile(`https?://[^\s"'<>，。]+`)

func (p *Pipeline) BuildToolCalls(intent string, message string) []agentruntime.ToolCall {
	if intent != dispatcher.IntentExternalKnowledge {
		return nil
	}
	if url := firstURL(message); url != "" {
		return []agentruntime.ToolCall{{Name: webtool.ToolWebFetch, Args: map[string]any{"url": url}}}
	}
	return []agentruntime.ToolCall{{Name: webtool.ToolWebSearch, Args: map[string]any{"query": strings.TrimSpace(message), "limit": 5}}}
}

func firstURL(message string) string {
	return urlPattern.FindString(strings.TrimSpace(message))
}
