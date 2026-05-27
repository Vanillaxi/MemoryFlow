package text_analyze

import "time"

// 输入
type TextAnalyzeInput struct {
	MemoryID    uint
	ContentText string
	Location    string
	CreatedAt   time.Time
}

// 输出
type AIAnalyzeResult struct {
	Summary         string   `json:"summary"`
	Tags            []string `json:"tags"`
	Mood            string   `json:"mood"`
	ImportanceScore float64  `json:"importanceScore"`
}
