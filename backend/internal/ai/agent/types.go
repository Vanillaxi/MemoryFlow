package agent

import (
	"context"

	agentruntime "memoryflow/internal/ai/agent/runtime"
)

type SummaryModel interface {
	GenerateWithSystem(ctx context.Context, systemPrompt string, userPrompt string) (string, error)
}

type Pipeline interface {
	BuildToolCalls(intent string, message string) []agentruntime.ToolCall
}

type ChatInput struct {
	Message string `json:"message"`
}

type ChatOutput struct {
	Answer    string   `json:"answer"`
	Intent    string   `json:"intent"`
	UsedTools []string `json:"used_tools"`
}
