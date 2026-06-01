package chat_pipeline

import (
	"memoryflow/internal/ai/agent/dispatcher"
	agentruntime "memoryflow/internal/ai/agent/runtime"
	memorytool "memoryflow/internal/ai/tools/memory"
	systemtool "memoryflow/internal/ai/tools/system"
)

func (p *Pipeline) BuildToolCalls(intent string, message string) []agentruntime.ToolCall {
	if intent == dispatcher.IntentMemoryQuery {
		return []agentruntime.ToolCall{
			{Name: systemtool.ToolGetCurrentTime, Args: map[string]any{}},
			{Name: memorytool.ToolQueryLongTermMemory, Args: map[string]any{"query": message, "mode": memorytool.ModeSemantic, "limit": 10}},
		}
	}
	return []agentruntime.ToolCall{
		{Name: systemtool.ToolGetCurrentTime, Args: map[string]any{}},
	}
}
