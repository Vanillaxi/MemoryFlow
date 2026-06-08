package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"memoryflow/internal/ai/agent/dispatcher"
	"memoryflow/internal/ai/agent/project_pipeline"
	"memoryflow/internal/ai/agent/reflection_pipeline"
	agentruntime "memoryflow/internal/ai/agent/runtime"
	"memoryflow/internal/ai/tools"
	webtool "memoryflow/internal/ai/tools/web"
)

type Agent struct {
	dispatch       func(message string) dispatcher.Decision
	pipelines      map[string]Pipeline
	toolExecutor   *agentruntime.ToolExecutor
	contextBuilder *agentruntime.ContextBuilder
	model          SummaryModel
	projectAgent   ProjectAgent
}

func NewAgent(registry *tools.ToolRegistry, model SummaryModel, chatPipeline Pipeline) *Agent {
	return &Agent{
		dispatch: dispatcher.Dispatch,
		pipelines: map[string]Pipeline{
			dispatcher.PipelineChat:       chatPipeline,
			dispatcher.PipelineReflection: reflection_pipeline.NewPipeline(),
		},
		toolExecutor:   agentruntime.NewToolExecutor(registry),
		contextBuilder: agentruntime.NewContextBuilder(),
		model:          model,
	}
}

func (a *Agent) SetProjectAgent(projectAgent ProjectAgent) {
	a.projectAgent = projectAgent
}

func (a *Agent) SetKnowledgePipeline(knowledgePipeline Pipeline) {
	if a != nil && knowledgePipeline != nil {
		a.pipelines[dispatcher.PipelineKnowledge] = knowledgePipeline
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
	decision := a.dispatch(message)
	if input.ProjectID != nil {
		decision.Pipeline = dispatcher.PipelineProject
		decision.Intent = dispatcher.ProjectIntent(message)
	}
	switch normalizePipelineOverride(input.Pipeline) {
	case dispatcher.PipelineChat:
		decision.Pipeline = dispatcher.PipelineChat
	case dispatcher.PipelineKnowledge:
		decision.Pipeline = dispatcher.PipelineKnowledge
		decision.Intent = dispatcher.IntentExternalKnowledge
	case dispatcher.PipelineProject:
		decision.Pipeline = dispatcher.PipelineProject
		decision.Intent = dispatcher.ProjectIntent(message)
	}
	if decision.Pipeline == dispatcher.PipelineProject {
		if a.projectAgent == nil {
			return nil, errors.New("project agent is not initialized")
		}
		output, err := a.projectAgent.Invoke(ctx, project_pipeline.ProjectAgentInput{
			Message: message, Intent: decision.Intent, ProjectID: input.ProjectID, Days: input.Days, Limit: input.Limit,
		})
		if err != nil {
			return nil, err
		}
		return &ChatOutput{
			Answer:       output.Answer,
			Intent:       decision.Intent,
			Pipeline:     dispatcher.PipelineProject,
			Project:      &output.Project,
			UsedTools:    output.UsedTools,
			Evidence:     output.Evidence,
			RawToolCalls: output.RawToolCalls,
		}, nil
	}
	if a.model == nil {
		return nil, errors.New("agent summary model is not initialized")
	}
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
		Answer:       strings.TrimSpace(answer),
		Intent:       decision.Intent,
		Pipeline:     decision.Pipeline,
		UsedTools:    usedTools,
		Evidence:     runtimeEvidence(logs),
		RawToolCalls: runtimeToolCalls(logs),
	}, nil
}

func normalizePipelineOverride(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "chat", dispatcher.PipelineChat:
		return dispatcher.PipelineChat
	case "knowledge", dispatcher.PipelineKnowledge:
		return dispatcher.PipelineKnowledge
	case "project", dispatcher.PipelineProject:
		return dispatcher.PipelineProject
	default:
		return ""
	}
}

func runtimeEvidence(logs []agentruntime.ToolCallLog) []project_pipeline.Evidence {
	evidence := make([]project_pipeline.Evidence, 0, len(logs))
	for _, log := range logs {
		detail := log.Result
		if log.Error != "" {
			detail = log.Error
		} else if summarized, ok := webEvidenceDetail(log.Name, log.Result); ok {
			detail = summarized
		}
		evidence = append(evidence, project_pipeline.Evidence{Source: log.Name, Detail: detail})
	}
	return evidence
}

func webEvidenceDetail(name string, result string) (string, bool) {
	switch name {
	case webtool.ToolWebFetch:
		var payload struct {
			Title          string `json:"title"`
			URL            string `json:"url"`
			Source         string `json:"source"`
			Domain         string `json:"domain"`
			FetchedAt      string `json:"fetched_at"`
			ContentPreview string `json:"content_preview"`
		}
		if err := json.Unmarshal([]byte(result), &payload); err != nil {
			return "", false
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return "", false
		}
		return string(bytes), true
	case webtool.ToolWebSearch:
		return result, true
	default:
		return "", false
	}
}

func runtimeToolCalls(logs []agentruntime.ToolCallLog) []project_pipeline.ToolCallLog {
	calls := make([]project_pipeline.ToolCallLog, 0, len(logs))
	for _, log := range logs {
		calls = append(calls, project_pipeline.ToolCallLog{Name: log.Name, Result: log.Result, Error: log.Error})
	}
	return calls
}
