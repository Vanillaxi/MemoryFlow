package agent

import (
	"context"

	"memoryflow/internal/ai/agent/project_pipeline"
	agentruntime "memoryflow/internal/ai/agent/runtime"
	"memoryflow/internal/domain/model"
)

type SummaryModel interface {
	GenerateWithSystem(ctx context.Context, systemPrompt string, userPrompt string) (string, error)
}

type Pipeline interface {
	BuildToolCalls(intent string, message string) []agentruntime.ToolCall
}

type ChatInput struct {
	Message   string `json:"message"`
	ProjectID *uint  `json:"project_id,omitempty"`
	Days      int    `json:"days,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type ChatOutput struct {
	Answer       string                         `json:"answer"`
	Intent       string                         `json:"intent"`
	Pipeline     string                         `json:"pipeline"`
	Project      *model.Project                 `json:"project,omitempty"`
	UsedTools    []string                       `json:"used_tools"`
	Evidence     []project_pipeline.Evidence    `json:"evidence,omitempty"`
	RawToolCalls []project_pipeline.ToolCallLog `json:"raw_tool_calls,omitempty"`
}

type ProjectAgent interface {
	Invoke(ctx context.Context, input project_pipeline.ProjectAgentInput) (*project_pipeline.ProjectAgentOutput, error)
}
