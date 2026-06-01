package memory

import (
	"context"
	"time"

	"memoryflow/internal/ai/retriever"
	"memoryflow/internal/domain/model"
)

const (
	ToolQueryLongTermMemory = "query_long_term_memory"
	ToolGetMemoryDetail     = "get_memory_detail"
	ToolAggregateMemory     = "aggregate_memory"

	ModeSemantic  = "semantic"
	ModeTimeline  = "timeline"
	ModeAggregate = "aggregate"

	DefaultLimit     = 20
	MaxLimit         = 100
	MaxSummaryLength = 240
)

type QueryLongTermMemoryInput struct {
	Query string `json:"query,omitempty"`
	From  string `json:"from,omitempty"`
	To    string `json:"to,omitempty"`
	Mode  string `json:"mode,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type GetMemoryDetailInput struct {
	MemoryID uint `json:"memory_id"`
}

type AggregateMemoryInput struct {
	From  string `json:"from,omitempty"`
	To    string `json:"to,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type LongTermMemoryItem struct {
	MemoryID        uint      `json:"memory_id"`
	Type            string    `json:"type,omitempty"`
	OccurredAt      time.Time `json:"occurred_at"`
	Location        string    `json:"location,omitempty"`
	Summary         string    `json:"summary,omitempty"`
	Mood            string    `json:"mood,omitempty"`
	Tags            string    `json:"tags,omitempty"`
	ImportanceScore float64   `json:"importance_score,omitempty"`
}

type LongTermMemoryEvidence struct {
	LongTermMemoryItem
	Score float32 `json:"score"`
}

type LongTermMemoryAggregation struct {
	Count      int      `json:"count"`
	Tags       []string `json:"tags"`
	Moods      []string `json:"moods"`
	Highlights []string `json:"highlights"`
}

type QueryLongTermMemoryOutput struct {
	Mode        string                     `json:"mode"`
	Evidence    []LongTermMemoryEvidence   `json:"evidence,omitempty"`
	Items       []LongTermMemoryItem       `json:"items,omitempty"`
	Aggregation *LongTermMemoryAggregation `json:"aggregation,omitempty"`
}

type MemoryRetriever interface {
	Retrieve(ctx context.Context, query string, opt retriever.RetrieveOptions) ([]retriever.RetrievedMemory, error)
}

type MemoryService interface {
	ListByTimeRange(ctx context.Context, from, to time.Time, limit int) ([]*model.MemoryItem, error)
	GetByID(ctx context.Context, id uint) (*model.MemoryItem, error)
}
