package memory_react_agent

import (
	"time"

	"memoryflow/internal/ai/agent/memory_chat_pipeline"
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
	Mode  string      `json:"mode"`
	Steps []TraceStep `json:"steps,omitempty"`
	Error string      `json:"error,omitempty"`
}

type TraceStep struct {
	Node      string `json:"node"`
	Event     string `json:"event"`
	Input     any    `json:"input,omitempty"`
	Output    any    `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
	StartedAt string `json:"started_at,omitempty"`
	EndedAt   string `json:"ended_at,omitempty"`
}
