package project_pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"memoryflow/internal/ai/agent/dispatcher"
	"memoryflow/internal/ai/tools"
	domainmodel "memoryflow/internal/domain/model"

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
	userPrompt := buildProjectUserPrompt(message, input.Intent, *project, input.Days, input.Limit)
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

func buildProjectUserPrompt(message string, intent string, project domainmodel.Project, days int, limit int) string {
	if intent == "" {
		intent = dispatcher.ProjectIntent(message)
	}
	base := fmt.Sprintf(
		"用户问题：%s\n识别出的 intent：%s\n当前项目：%s\n项目描述：%s\nrepo_owner=%s\nrepo_name=%s\nrepo_url=%s\nrepository=%s\ntech_stack=%s\nstatus=%s\ndays=%d\nlimit=%d\n",
		message,
		intent,
		project.Name,
		project.Description,
		project.RepoOwner,
		project.RepoName,
		project.RepoURL,
		project.Repository(),
		project.TechStack,
		project.Status,
		days,
		limit,
	)
	if intent != dispatcher.IntentProjectHandoff {
		return base + "请自主调用工具并基于证据总结。"
	}
	return base + `这是 Project Handoff Summary 模式。handoff 不是工具，而是 project_pipeline 的输出目标。
请先收集只读证据，再生成结构化项目交接摘要：
- 必须调用 get_current_time。
- 必须调用 get_recent_commits，总结最近 GitHub 进展。
- 必须调用 get_recent_issues，并优先查询 state=open。
- 必须调用 get_pull_requests 查询最近 PR。
- 必须调用 query_long_term_memory，围绕项目名、最近进度、架构决策、已完成、未完成等查询。
- web_fetch / web_search 不是必调；只有用户提供外部 URL 或明确要求参考官方文档/外部资料时才调用。

最终 answer 必须使用结构化 Markdown，并包含这些章节：
# Project Handoff Summary: <ProjectName>
## 1. 项目定位
## 2. 当前核心能力
## 3. 最近 GitHub 进展
## 4. Issues / PR 状态
## 5. 关键架构决策
## 6. 当前风险 / 待确认点
## 7. 下一步建议
## 8. 给 Codex / ChatGPT 的上下文包

不要新增 HandoffTool，不要创建 TodoTool，不要写 GitHub，不要写文件，不要自动保存到 memory。
不要编造 commits/issues/PR/memory；如果某类证据没有查到，要明确说明没有查询到。`
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
