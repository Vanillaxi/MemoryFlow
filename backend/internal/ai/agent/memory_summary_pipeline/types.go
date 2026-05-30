package memory_summary_pipeline

import "time"

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

type SummaryAggregation struct {
	Count      int
	Tags       []string
	Moods      []string
	Highlights []string
	MemoryList string
}
