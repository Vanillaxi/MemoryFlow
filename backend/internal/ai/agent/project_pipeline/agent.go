package project_pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"memoryflow/internal/ai/tools"

	"github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

type Agent struct {
	resolver *ProjectResolver
	react    *react.Agent
}

func NewAgent(ctx context.Context, resolver *ProjectResolver, chatModel model.ToolCallingChatModel, internalTools []tools.Tool) (*Agent, error) {
	if resolver == nil {
		return nil, errors.New("project resolver is required")
	}
	baseTools := make([]einotool.BaseTool, 0, len(internalTools))
	for _, currentTool := range internalTools {
		if currentTool != nil {
			baseTools = append(baseTools, NewAdapter(currentTool))
		}
	}
	reactAgent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig:      compose.ToolsNodeConfig{Tools: baseTools},
		MessageModifier: func(_ context.Context, input []*schema.Message) []*schema.Message {
			return append([]*schema.Message{schema.SystemMessage(SystemPrompt)}, input...)
		},
		MaxStep: 20,
	})
	if err != nil {
		return nil, err
	}
	return &Agent{resolver: resolver, react: reactAgent}, nil
}

func (a *Agent) Invoke(ctx context.Context, input ProjectAgentInput) (*ProjectAgentOutput, error) {
	if a == nil || a.react == nil {
		return nil, errors.New("project agent is not initialized")
	}
	message := strings.TrimSpace(input.Message)
	if message == "" {
		return nil, errors.New("message is required")
	}
	project, err := a.resolver.Resolve(ctx, message, input.ProjectID)
	if err != nil {
		return nil, err
	}

	recorder := &toolCallRecorder{}
	ctx = context.WithValue(ctx, executionScopeKey{}, &executionScope{
		repository: project.Repository(),
		days:       input.Days,
		limit:      input.Limit,
		recorder:   recorder,
	})
	userPrompt := fmt.Sprintf(
		"用户问题：%s\n当前项目：%s\n项目描述：%s\nrepository=%s\ndays=%d\nlimit=%d\n请自主调用工具并基于证据总结。",
		message, project.Name, project.Description, project.Repository(), input.Days, input.Limit,
	)
	result, err := a.react.Generate(ctx, []*schema.Message{schema.UserMessage(userPrompt)})
	if err != nil {
		return nil, fmt.Errorf("project react agent failed: %w", err)
	}
	calls := recorder.snapshot()
	return &ProjectAgentOutput{
		Answer:       strings.TrimSpace(result.Content),
		Project:      *project,
		UsedTools:    usedToolNames(calls),
		Evidence:     toolEvidence(calls),
		RawToolCalls: calls,
	}, nil
}

func usedToolNames(calls []ToolCallLog) []string {
	seen := make(map[string]bool)
	names := make([]string, 0, len(calls))
	for _, call := range calls {
		if !seen[call.Name] {
			seen[call.Name] = true
			names = append(names, call.Name)
		}
	}
	return names
}

func toolEvidence(calls []ToolCallLog) []Evidence {
	evidence := make([]Evidence, 0, len(calls))
	for _, call := range calls {
		detail := call.Result
		if call.Error != "" {
			detail = call.Error
		}
		evidence = append(evidence, Evidence{Source: call.Name, Detail: detail})
	}
	return evidence
}
