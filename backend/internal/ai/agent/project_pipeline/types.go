package project_pipeline

import "memoryflow/internal/domain/model"

type ProjectAgentInput struct {
	Message   string `json:"message"`
	ProjectID *uint  `json:"project_id,omitempty"`
	Days      int    `json:"days,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type ProjectAgentOutput struct {
	Answer       string        `json:"answer"`
	Project      model.Project `json:"project"`
	UsedTools    []string      `json:"used_tools"`
	Evidence     []Evidence    `json:"evidence"`
	RawToolCalls []ToolCallLog `json:"raw_tool_calls,omitempty"`
}

type Evidence struct {
	Source string `json:"source"`
	Detail string `json:"detail"`
}

type ToolCallLog struct {
	Name   string         `json:"name"`
	Args   map[string]any `json:"args,omitempty"`
	Result string         `json:"result,omitempty"`
	Error  string         `json:"error,omitempty"`
}
