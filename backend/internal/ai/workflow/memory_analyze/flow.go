package memory_analyze

import (
	"context"
	"errors"
	"strings"
)

type ChatModel interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type Workflow struct {
	chatModel ChatModel
}

func NewWorkflow(chatModel ChatModel) *Workflow {
	return &Workflow{chatModel: chatModel}
}

func (w *Workflow) Invoke(ctx context.Context, input AnalyzeInput) (*AnalyzeResult, error) {
	normalized, err := input.Normalize()
	if err != nil {
		return nil, err
	}

	switch normalized.Type {
	case TypeText:
		if w.chatModel == nil {
			return nil, errors.New("memory analyze chat model is nil")
		}
		raw, err := w.chatModel.Generate(ctx, BuildPrompt(normalized))
		if err != nil {
			return nil, err
		}
		return ParseAnalyzeResult(raw)
	case TypeImage, TypeMixed:
		return analyzeImage(normalized), nil
	default:
		return nil, errors.New("unsupported memory analyze type")
	}
}

func analyzeImage(input AnalyzeInput) *AnalyzeResult {
	summary := "这是一条图片记忆。"
	tags := []string{"图片", "生活记录"}

	if input.ContentText != "" {
		summary = "这是一条带有文字说明的图片记忆：" + input.ContentText
		tags = append(tags, "图片说明")
	}

	if strings.TrimSpace(input.Location) != "" {
		tags = append(tags, input.Location)
	}

	return &AnalyzeResult{
		Summary:         summary,
		Tags:            tags,
		Mood:            "neutral",
		ImportanceScore: 0.5,
	}
}
