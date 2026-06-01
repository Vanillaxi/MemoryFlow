package reflection_pipeline

import (
	agentruntime "memoryflow/internal/ai/agent/runtime"
	memorytool "memoryflow/internal/ai/tools/memory"
	systemtool "memoryflow/internal/ai/tools/system"
)

// Pipeline is the extension point for daily insights, weekly reviews,
// mood trends, and action suggestions.
type Pipeline struct{}

func NewPipeline() *Pipeline {
	return &Pipeline{}
}

func (p *Pipeline) BuildToolCalls(_ string, _ string) []agentruntime.ToolCall {
	return []agentruntime.ToolCall{
		{Name: systemtool.ToolGetCurrentTime, Args: map[string]any{}},
		{Name: memorytool.ToolAggregateMemory, Args: map[string]any{"limit": 20}},
	}
}
