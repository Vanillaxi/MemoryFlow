package project_pipeline

import (
	"context"
	"errors"
	"testing"

	aitools "memoryflow/internal/ai/tools"
	githubtool "memoryflow/internal/ai/tools/github"
	"memoryflow/internal/domain/model"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type fakeToolCallingModel struct {
	calls int
}

func (f *fakeToolCallingModel) Generate(_ context.Context, messages []*schema.Message, _ ...einomodel.Option) (*schema.Message, error) {
	f.calls++
	if f.calls == 1 {
		return schema.AssistantMessage("", []schema.ToolCall{{
			ID:       "call_1",
			Function: schema.FunctionCall{Name: githubtool.ToolGetRecentCommits, Arguments: `{"repository":"wrong/repo","days":99,"token":"must-not-survive"}`},
		}}), nil
	}
	return schema.AssistantMessage("基于 commits 完成总结。", nil), nil
}

func (f *fakeToolCallingModel) Stream(context.Context, []*schema.Message, ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("not implemented")
}

func (f *fakeToolCallingModel) WithTools([]*schema.ToolInfo) (einomodel.ToolCallingChatModel, error) {
	return f, nil
}

type recordingTool struct {
	args map[string]any
}

func (t *recordingTool) Name() string        { return githubtool.ToolGetRecentCommits }
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
		&fakeToolCallingModel{},
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
