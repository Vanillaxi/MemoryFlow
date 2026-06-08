package project_pipeline

import (
	"context"
	"errors"
	"testing"

	aitools "memoryflow/internal/ai/tools"
	githubtool "memoryflow/internal/ai/tools/github"
	memorytool "memoryflow/internal/ai/tools/memory"
	systemtool "memoryflow/internal/ai/tools/system"
	"memoryflow/internal/domain/model"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type fakeToolCallingModel struct {
	toolName  string
	arguments string
	answer    string
	calls     int
}

type sequenceToolCallingModel struct {
	toolCalls []schema.ToolCall
	answer    string
	calls     int
}

func (f *sequenceToolCallingModel) Generate(_ context.Context, messages []*schema.Message, _ ...einomodel.Option) (*schema.Message, error) {
	f.calls++
	if f.calls <= len(f.toolCalls) {
		return schema.AssistantMessage("", []schema.ToolCall{f.toolCalls[f.calls-1]}), nil
	}
	answer := f.answer
	if answer == "" {
		answer = "# Project Handoff Summary: MemoryFlow\n\n## 1. 项目定位\n基于证据总结。"
	}
	return schema.AssistantMessage(answer, nil), nil
}

func (f *sequenceToolCallingModel) Stream(context.Context, []*schema.Message, ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("not implemented")
}

func (f *sequenceToolCallingModel) WithTools([]*schema.ToolInfo) (einomodel.ToolCallingChatModel, error) {
	return f, nil
}

func (f *fakeToolCallingModel) Generate(_ context.Context, messages []*schema.Message, _ ...einomodel.Option) (*schema.Message, error) {
	f.calls++
	if f.calls == 1 {
		toolName := f.toolName
		if toolName == "" {
			toolName = githubtool.ToolGetRecentCommits
		}
		arguments := f.arguments
		if arguments == "" {
			arguments = `{"repository":"wrong/repo","days":99,"token":"must-not-survive"}`
		}
		return schema.AssistantMessage("", []schema.ToolCall{{
			ID:       "call_1",
			Function: schema.FunctionCall{Name: toolName, Arguments: arguments},
		}}), nil
	}
	answer := f.answer
	if answer == "" {
		answer = "基于工具结果完成总结。"
	}
	return schema.AssistantMessage(answer, nil), nil
}

func (f *fakeToolCallingModel) Stream(context.Context, []*schema.Message, ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("not implemented")
}

func (f *fakeToolCallingModel) WithTools([]*schema.ToolInfo) (einomodel.ToolCallingChatModel, error) {
	return f, nil
}

type recordingTool struct {
	name string
	args map[string]any
}

func (t *recordingTool) Name() string {
	if t.name == "" {
		return githubtool.ToolGetRecentCommits
	}
	return t.name
}
func (t *recordingTool) Description() string { return "commits" }
func (t *recordingTool) Call(_ context.Context, args map[string]any) (string, error) {
	t.args = args
	return `{"repository":"vanillaxi/MemoryFlow","commits":[]}`, nil
}

func TestProjectAgentInjectsResolvedRepositoryAndReportsUsedTool(t *testing.T) {
	github := &recordingTool{}
	currentAgent, err := NewAgent(
		context.Background(),
		NewProjectResolver(fakeProjectLookup{fromMessage: &model.Project{Name: "MemoryFlow", RepoOwner: "vanillaxi", RepoName: "MemoryFlow"}}),
		&fakeToolCallingModel{toolName: githubtool.ToolGetRecentCommits},
		[]aitools.Tool{github},
	)
	if err != nil {
		t.Fatal(err)
	}
	output, err := currentAgent.Invoke(context.Background(), ProjectAgentInput{Message: "MemoryFlow 最近做到哪了", Days: 7, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if github.args["repository"] != "vanillaxi/MemoryFlow" || github.args["days"] != 7 || github.args["limit"] != 10 {
		t.Fatalf("unexpected github args: %#v", github.args)
	}
	if _, exists := github.args["token"]; exists {
		t.Fatalf("sensitive tool arg was not removed: %#v", github.args)
	}
	if len(output.UsedTools) != 1 || output.UsedTools[0] != githubtool.ToolGetRecentCommits {
		t.Fatalf("unexpected tools: %#v", output.UsedTools)
	}
}

func TestProjectAgentInjectsRepositoryForIssuesTool(t *testing.T) {
	github := &recordingTool{name: githubtool.ToolGetRecentIssues}
	currentAgent, err := NewAgent(
		context.Background(),
		NewProjectResolver(fakeProjectLookup{fromMessage: &model.Project{Name: "MemoryFlow", RepoOwner: "vanillaxi", RepoName: "MemoryFlow"}}),
		&fakeToolCallingModel{toolName: githubtool.ToolGetRecentIssues, arguments: `{"repository":"wrong/repo","state":"open"}`},
		[]aitools.Tool{github},
	)
	if err != nil {
		t.Fatal(err)
	}
	output, err := currentAgent.Invoke(context.Background(), ProjectAgentInput{Message: "MemoryFlow 还有哪些 issue", Days: 30, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if github.args["repository"] != "vanillaxi/MemoryFlow" || github.args["days"] != 30 || github.args["limit"] != 10 {
		t.Fatalf("unexpected github args: %#v", github.args)
	}
	if len(output.UsedTools) != 1 || output.UsedTools[0] != githubtool.ToolGetRecentIssues {
		t.Fatalf("unexpected tools: %#v", output.UsedTools)
	}
}

func TestProjectAgentInjectsRepositoryForPullRequestsTool(t *testing.T) {
	github := &recordingTool{name: githubtool.ToolGetPullRequests}
	currentAgent, err := NewAgent(
		context.Background(),
		NewProjectResolver(fakeProjectLookup{fromMessage: &model.Project{Name: "MemoryFlow", RepoOwner: "vanillaxi", RepoName: "MemoryFlow"}}),
		&fakeToolCallingModel{toolName: githubtool.ToolGetPullRequests, arguments: `{"repository":"wrong/repo","state":"open"}`},
		[]aitools.Tool{github},
	)
	if err != nil {
		t.Fatal(err)
	}
	output, err := currentAgent.Invoke(context.Background(), ProjectAgentInput{Message: "MemoryFlow 最近有哪些 PR", Days: 30, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if github.args["repository"] != "vanillaxi/MemoryFlow" || github.args["limit"] != 10 {
		t.Fatalf("unexpected github args: %#v", github.args)
	}
	if _, exists := github.args["days"]; exists {
		t.Fatalf("pull request tool should not receive days: %#v", github.args)
	}
	if len(output.UsedTools) != 1 || output.UsedTools[0] != githubtool.ToolGetPullRequests {
		t.Fatalf("unexpected tools: %#v", output.UsedTools)
	}
}

func TestProjectAgentHandoffCollectsRequiredReadOnlyEvidence(t *testing.T) {
	timeTool := &recordingTool{name: systemtool.ToolGetCurrentTime}
	commitsTool := &recordingTool{name: githubtool.ToolGetRecentCommits}
	issuesTool := &recordingTool{name: githubtool.ToolGetRecentIssues}
	prTool := &recordingTool{name: githubtool.ToolGetPullRequests}
	memoryTool := &recordingTool{name: memorytool.ToolQueryLongTermMemory}
	currentAgent, err := NewAgent(
		context.Background(),
		NewProjectResolver(fakeProjectLookup{fromMessage: &model.Project{
			Name:        "MemoryFlow",
			Description: "本地优先的个人长期记忆 Agent",
			RepoOwner:   "vanillaxi",
			RepoName:    "MemoryFlow",
			RepoURL:     "https://github.com/vanillaxi/MemoryFlow",
			TechStack:   "Go, Gin, SQLite, Milvus, Eino",
			Status:      "active",
		}}),
		&sequenceToolCallingModel{toolCalls: []schema.ToolCall{
			{ID: "time", Function: schema.FunctionCall{Name: systemtool.ToolGetCurrentTime, Arguments: `{}`}},
			{ID: "commits", Function: schema.FunctionCall{Name: githubtool.ToolGetRecentCommits, Arguments: `{}`}},
			{ID: "issues", Function: schema.FunctionCall{Name: githubtool.ToolGetRecentIssues, Arguments: `{"state":"open"}`}},
			{ID: "prs", Function: schema.FunctionCall{Name: githubtool.ToolGetPullRequests, Arguments: `{"state":"all"}`}},
			{ID: "memory", Function: schema.FunctionCall{Name: memorytool.ToolQueryLongTermMemory, Arguments: `{"query":"MemoryFlow 最近进度 架构决策 已完成 未完成","mode":"semantic","limit":5}`}},
		}},
		[]aitools.Tool{timeTool, commitsTool, issuesTool, prTool, memoryTool},
	)
	if err != nil {
		t.Fatal(err)
	}
	output, err := currentAgent.Invoke(context.Background(), ProjectAgentInput{
		Message: "帮我总结 MemoryFlow 当前进度，方便开启新聊天",
		Intent:  "project_handoff",
		Days:    7,
		Limit:   5,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		systemtool.ToolGetCurrentTime,
		githubtool.ToolGetRecentCommits,
		githubtool.ToolGetRecentIssues,
		githubtool.ToolGetPullRequests,
		memorytool.ToolQueryLongTermMemory,
	} {
		if !containsToolName(output.UsedTools, want) {
			t.Fatalf("missing tool %q in %#v", want, output.UsedTools)
		}
	}
	if commitsTool.args["repository"] != "vanillaxi/MemoryFlow" || commitsTool.args["days"] != 7 || commitsTool.args["limit"] != 5 {
		t.Fatalf("unexpected commits args: %#v", commitsTool.args)
	}
	if issuesTool.args["repository"] != "vanillaxi/MemoryFlow" || issuesTool.args["state"] != "open" {
		t.Fatalf("unexpected issues args: %#v", issuesTool.args)
	}
	if prTool.args["repository"] != "vanillaxi/MemoryFlow" || prTool.args["limit"] != 5 {
		t.Fatalf("unexpected pull request args: %#v", prTool.args)
	}
	if memoryTool.args["query"] == "" {
		t.Fatalf("expected memory query args: %#v", memoryTool.args)
	}
}

func containsToolName(tools []string, want string) bool {
	for _, tool := range tools {
		if tool == want {
			return true
		}
	}
	return false
}
