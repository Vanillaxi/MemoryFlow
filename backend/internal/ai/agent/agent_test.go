package agent

import (
	"context"
	"strings"
	"testing"

	"memoryflow/internal/ai/agent/project_pipeline"
	memorytools "memoryflow/internal/ai/tools"
	githubtools "memoryflow/internal/ai/tools/github"
	"memoryflow/internal/domain/model"
)

type fakeTool struct {
	name   string
	result string
	err    error
}

func (f fakeTool) Name() string        { return f.name }
func (f fakeTool) Description() string { return f.name }
func (f fakeTool) Call(context.Context, map[string]any) (string, error) {
	return f.result, f.err
}

type fakeSummaryModel struct{}

func (f *fakeSummaryModel) GenerateWithSystem(_ context.Context, _ string, _ string) (string, error) {
	return " 已完成项目进展总结。 ", nil
}

type fakeProjectAgent struct {
	input project_pipeline.ProjectAgentInput
}

func (f *fakeProjectAgent) Invoke(_ context.Context, input project_pipeline.ProjectAgentInput) (*project_pipeline.ProjectAgentOutput, error) {
	f.input = input
	tool := githubtools.ToolGetRecentCommits
	normalized := strings.ToLower(input.Message)
	if strings.Contains(normalized, "issue") || strings.Contains(normalized, "未处理") || strings.Contains(normalized, "待处理") {
		tool = githubtools.ToolGetRecentIssues
	}
	if strings.Contains(normalized, "pr") || strings.Contains(normalized, "pull request") {
		tool = githubtools.ToolGetPullRequests
	}
	return &project_pipeline.ProjectAgentOutput{
		Answer:    "已完成项目进展总结。",
		Project:   model.Project{Name: "MemoryFlow", RepoOwner: "vanillaxi", RepoName: "MemoryFlow"},
		UsedTools: []string{tool},
	}, nil
}

func TestChatProjectProgressUsesProjectAgent(t *testing.T) {
	projectAgent := &fakeProjectAgent{}
	currentAgent := NewAgent(memorytools.NewToolRegistry(), &fakeSummaryModel{}, nil)
	currentAgent.SetProjectAgent(projectAgent)

	output, err := currentAgent.Chat(context.Background(), ChatInput{Message: "MemoryFlow 最近做到哪了？", Days: 3, Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if output.Intent != "project_progress" || output.Pipeline != "project_pipeline" || len(output.UsedTools) != 1 || output.UsedTools[0] != githubtools.ToolGetRecentCommits {
		t.Fatalf("unexpected output: %#v", output)
	}
	if projectAgent.input.Days != 3 || projectAgent.input.Limit != 5 {
		t.Fatalf("unexpected project agent input: %#v", projectAgent.input)
	}
}

func TestChatProjectIssueQuestionUsesProjectAgent(t *testing.T) {
	projectAgent := &fakeProjectAgent{}
	currentAgent := NewAgent(memorytools.NewToolRegistry(), &fakeSummaryModel{}, nil)
	currentAgent.SetProjectAgent(projectAgent)

	output, err := currentAgent.Chat(context.Background(), ChatInput{Message: "MemoryFlow 还有哪些 issue 没处理？"})
	if err != nil {
		t.Fatal(err)
	}
	if output.Intent != "project_issue_status" || output.Pipeline != "project_pipeline" || len(output.UsedTools) != 1 || output.UsedTools[0] != githubtools.ToolGetRecentIssues {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestChatProjectPRQuestionUsesProjectAgent(t *testing.T) {
	projectAgent := &fakeProjectAgent{}
	currentAgent := NewAgent(memorytools.NewToolRegistry(), &fakeSummaryModel{}, nil)
	currentAgent.SetProjectAgent(projectAgent)

	output, err := currentAgent.Chat(context.Background(), ChatInput{Message: "MemoryFlow 最近有哪些 PR？"})
	if err != nil {
		t.Fatal(err)
	}
	if output.Intent != "project_pr_status" || output.Pipeline != "project_pipeline" || len(output.UsedTools) != 1 || output.UsedTools[0] != githubtools.ToolGetPullRequests {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestChatExplicitProjectPipelineOverride(t *testing.T) {
	projectAgent := &fakeProjectAgent{}
	currentAgent := NewAgent(memorytools.NewToolRegistry(), &fakeSummaryModel{}, nil)
	currentAgent.SetProjectAgent(projectAgent)

	output, err := currentAgent.Chat(context.Background(), ChatInput{Message: "你好", Pipeline: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if output.Pipeline != "project_pipeline" || output.Intent != "project_progress" {
		t.Fatalf("unexpected output: %#v", output)
	}
}
