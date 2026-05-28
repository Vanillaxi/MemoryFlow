package rag_answer_eino

import (
	"time"

	"memoryflow/internal/ai/retriever"
)

type RAGAnswerInput struct {
	Question  string
	TopK      int
	Type      string
	StartTime *time.Time
	EndTime   *time.Time
}

type RAGAnswerOutput struct {
	Answer     string            `json:"answer"`
	References []MemoryReference `json:"references"`
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

type RAGState struct {
	Input         RAGAnswerInput
	Retrieved     []retriever.RetrievedMemory
	Reranked      []retriever.RetrievedMemory
	MemoryContext string
	Prompt        string
	Answer        string
}
