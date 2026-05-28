package image_analyze

import (
	"strings"
)

type Workflow struct{}

func NewWorkflow() *Workflow {
	return &Workflow{}
}

func (w *Workflow) Run(input ImageAnalyzeInput) (*ImageAnalyzeResult, error) {
	contentText := strings.TrimSpace(input.ContentText)

	summary := "这是一条图片记忆。"
	tags := []string{"图片", "生活记录"}
	mood := "neutral"
	importanceScore := 0.5

	if contentText != "" {
		summary = "这是一条带有文字说明的图片记忆：" + contentText
		tags = append(tags, "图片说明")
	}

	if strings.TrimSpace(input.Location) != "" {
		tags = append(tags, input.Location)
	}

	return &ImageAnalyzeResult{
		Summary:         summary,
		Tags:            tags,
		Mood:            mood,
		ImportanceScore: importanceScore,
	}, nil
}
