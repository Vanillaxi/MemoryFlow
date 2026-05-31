package chat_pipeline

import (
	"context"

	"memoryflow/internal/ai/models"
)

type ChatModel interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

func NewModel(cfg models.Config) *models.ArkToolCallingChatModel {
	return models.NewArkToolCallingChatModel(cfg)
}
