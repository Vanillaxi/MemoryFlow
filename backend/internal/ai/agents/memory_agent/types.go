package memory_agent

import (
	"time"

	"memoryflow/internal/ai/pipelines/memory_chat_pipeline"
)

type AgentInput struct {
	Message   string     `json:"message"`
	TopK      int        `json:"top_k,omitempty"`
	Type      string     `json:"type,omitempty"`
	StartTime *time.Time `json:"-"`
	EndTime   *time.Time `json:"-"`
	Debug     bool       `json:"debug,omitempty"`
}

type AgentOutput struct {
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
