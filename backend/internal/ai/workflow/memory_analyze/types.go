package memory_analyze

import (
	"errors"
	"strings"
	"time"
)

const (
	TypeText  = "text"
	TypeImage = "image"
	TypeMixed = "mixed"
)

type AnalyzeInput struct {
	MemoryID    uint
	Type        string
	ContentText string
	ImageURL    string
	Location    string
	OccurredAt  time.Time
}

type AnalyzeResult struct {
	Summary         string   `json:"summary"`
	Tags            []string `json:"tags"`
	Mood            string   `json:"mood"`
	ImportanceScore float64  `json:"importance_score"`
}

func (input AnalyzeInput) Normalize() (AnalyzeInput, error) {
	input.Type = strings.TrimSpace(input.Type)
	input.ContentText = strings.TrimSpace(input.ContentText)
	input.ImageURL = strings.TrimSpace(input.ImageURL)

	if input.Type == "" {
		switch {
		case input.ContentText != "" && input.ImageURL != "":
			input.Type = TypeMixed
		case input.ImageURL != "":
			input.Type = TypeImage
		case input.ContentText != "":
			input.Type = TypeText
		}
	}

	switch input.Type {
	case TypeText:
		if input.ContentText == "" {
			return AnalyzeInput{}, errors.New("text memory content is empty")
		}
	case TypeImage:
		if input.ImageURL == "" {
			return AnalyzeInput{}, errors.New("image memory URL is empty")
		}
	case TypeMixed:
		if input.ContentText == "" && input.ImageURL == "" {
			return AnalyzeInput{}, errors.New("mixed memory content is empty")
		}
	default:
		return AnalyzeInput{}, errors.New("memory analyze input is empty or has unsupported type")
	}

	return input, nil
}
