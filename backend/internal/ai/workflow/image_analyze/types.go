package image_analyze

import "time"

type ImageAnalyzeInput struct {
	MemoryID    uint
	ImageURL    string
	ContentText string
	Location    string
	CreatedAt   time.Time
}

type ImageAnalyzeResult struct {
	Summary         string
	Tags            []string
	Mood            string
	ImportanceScore float64
}
