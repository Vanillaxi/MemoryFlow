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
}

type ChatOutput struct {
	Answer     string                                 `json:"answer"`
	References []memory_chat_pipeline.MemoryReference `json:"references,omitempty"`
	Intent     string                                 `json:"intent"`
}
