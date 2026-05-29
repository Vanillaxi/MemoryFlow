package memory_agent

import (
	"time"

	"memoryflow/internal/ai/pipelines/memory_chat_pipeline"
)

type ChatInput struct {
	Message   string
	TopK      int
	Type      string
	StartTime *time.Time
	EndTime   *time.Time
	Debug     bool
}

type ChatOutput struct {
	Answer     string                                 `json:"answer"`
	References []memory_chat_pipeline.MemoryReference `json:"references,omitempty"`
	Intent     string                                 `json:"intent"`
	Trace      *AgentTrace                            `json:"trace,omitempty"`
}

type AgentTrace struct {
	RouterTool      string         `json:"router_tool,omitempty"`
	RouterArguments map[string]any `json:"router_arguments,omitempty"`
	UsedFallback    bool           `json:"used_fallback"`
	Summarized      bool           `json:"summarized"`
	ToolResultCount int            `json:"tool_result_count,omitempty"`
	Error           string         `json:"error,omitempty"`
}

type RouterDecision struct {
	ToolName  ToolName       `json:"tool_name"`
	Arguments map[string]any `json:"arguments"`
}
