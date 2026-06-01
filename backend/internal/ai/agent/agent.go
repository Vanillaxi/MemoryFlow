package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"memoryflow/internal/ai/agent/dispatcher"
	"memoryflow/internal/ai/agent/project_pipeline"
	"memoryflow/internal/ai/agent/reflection_pipeline"
	agentruntime "memoryflow/internal/ai/agent/runtime"
	"memoryflow/internal/ai/tools"
)

type Agent struct {
	dispatch       func(message string) dispatcher.Decision
	pipelines      map[string]Pipeline
	toolExecutor   *agentruntime.ToolExecutor
	contextBuilder *agentruntime.ContextBuilder
	model          SummaryModel
}

func NewAgent(registry *tools.ToolRegistry, model SummaryModel, chatPipeline Pipeline) *Agent {
	return &Agent{
		dispatch: dispatcher.Dispatch,
		pipelines: map[string]Pipeline{
			dispatcher.PipelineChat:       chatPipeline,
			dispatcher.PipelineProject:    project_pipeline.NewPipeline(),
			dispatcher.PipelineReflection: reflection_pipeline.NewPipeline(),
		},
		toolExecutor:   agentruntime.NewToolExecutor(registry),
		contextBuilder: agentruntime.NewContextBuilder(),
		model:          model,
	}
}

func (a *Agent) Chat(ctx context.Context, input ChatInput) (*ChatOutput, error) {
	message := strings.TrimSpace(input.Message)
	if message == "" {
		return nil, errors.New("message is required")
	}
	if a == nil || a.dispatch == nil || a.toolExecutor == nil || a.contextBuilder == nil {
		return nil, errors.New("agent runtime is not initialized")
	}
	if a.model == nil {
		return nil, errors.New("agent summary model is not initialized")
	}

	decision := a.dispatch(message)
	pipeline, ok := a.pipelines[decision.Pipeline]
	if !ok || pipeline == nil {
		return nil, fmt.Errorf("pipeline %q is not initialized", decision.Pipeline)
	}

	logs, usedTools, err := a.toolExecutor.Execute(ctx, pipeline.BuildToolCalls(decision.Intent, message))
	if err != nil {
		return nil, err
	}
	prompt, err := a.contextBuilder.Build(message, decision.Intent, logs)
	if err != nil {
		return nil, err
	}
	answer, err := a.model.GenerateWithSystem(ctx, agentruntime.SummarySystemPrompt, prompt)
	if err != nil {
		return nil, fmt.Errorf("summarize tool results failed: %w", err)
	}

	return &ChatOutput{
		Answer:    strings.TrimSpace(answer),
		Intent:    decision.Intent,
		UsedTools: usedTools,
	}, nil
}
