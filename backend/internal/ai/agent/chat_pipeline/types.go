package chat_pipeline

import (
	"time"

	memorytools "memoryflow/internal/ai/tools"
)

type MemoryRetriever = memorytools.MemoryRetriever

type MemoryService = memorytools.MemoryService

type ChatInput struct {
	Message   string     `json:"message"`
	TopK      int        `json:"top_k,omitempty"`
	Type      string     `json:"type,omitempty"`
	StartTime *time.Time `json:"-"`
	EndTime   *time.Time `json:"-"`
	Debug     bool       `json:"debug,omitempty"`
}

type ChatOutput struct {
	Answer     string            `json:"answer"`
	References []MemoryReference `json:"references,omitempty"`
	Intent     string            `json:"intent"`
	Trace      *AgentTrace       `json:"trace,omitempty"`
}

type MemoryReference struct {
	ID         uint    `json:"id"`
	Summary    string  `json:"summary"`
	Content    string  `json:"content,omitempty"`
	ImageURL   string  `json:"image_url,omitempty"`
	OccurredAt string  `json:"occurred_at,omitempty"`
	Location   string  `json:"location,omitempty"`
	Mood       string  `json:"mood,omitempty"`
	Score      float32 `json:"score"`
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

type SummaryInput struct {
	From  time.Time `json:"from"`
	To    time.Time `json:"to"`
	Limit int       `json:"limit"`
}

type SummaryOutput struct {
	From       time.Time `json:"from"`
	To         time.Time `json:"to"`
	Summary    string    `json:"summary"`
	Highlights []string  `json:"highlights"`
	Tags       []string  `json:"tags"`
	Moods      []string  `json:"moods"`
	Count      int       `json:"count"`
}
