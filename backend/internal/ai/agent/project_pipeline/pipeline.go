package project_pipeline

import (
	"memoryflow/internal/ai/agent/dispatcher"
	agentruntime "memoryflow/internal/ai/agent/runtime"
	githubtool "memoryflow/internal/ai/tools/github"
	memorytool "memoryflow/internal/ai/tools/memory"
	systemtool "memoryflow/internal/ai/tools/system"
)

type Pipeline struct{}

func NewPipeline() *Pipeline {
	return &Pipeline{}
}

func (p *Pipeline) BuildToolCalls(intent string, message string) []agentruntime.ToolCall {
	if intent == dispatcher.IntentHandoff {
		return []agentruntime.ToolCall{
			{Name: systemtool.ToolGetCurrentTime, Args: map[string]any{}},
			{Name: memorytool.ToolAggregateMemory, Args: map[string]any{"limit": 20}},
		}
	}
	return []agentruntime.ToolCall{
		{Name: systemtool.ToolGetCurrentTime, Args: map[string]any{}},
		{Name: memorytool.ToolQueryLongTermMemory, Args: map[string]any{"query": message, "mode": memorytool.ModeSemantic, "limit": 10}},
		{Name: githubtool.ToolGetRecentCommits, Args: map[string]any{"limit": 10}},
	}
}
