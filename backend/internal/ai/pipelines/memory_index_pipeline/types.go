package memory_index_pipeline

import "memoryflow/internal/model"

type ReindexInput struct {
	BatchSize int
}

type ReindexOutput struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}

type IndexDocument struct {
	MemoryID   int64
	Content    string
	MemoryType string
	OccurredAt int64
	Memory     model.MemoryItem
}
