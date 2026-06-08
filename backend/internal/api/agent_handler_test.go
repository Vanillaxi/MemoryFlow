package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"memoryflow/internal/ai/agent"
	"memoryflow/internal/ai/agent/project_pipeline"
	agentruntime "memoryflow/internal/ai/agent/runtime"
	memorytools "memoryflow/internal/ai/tools"
	githubtools "memoryflow/internal/ai/tools/github"
	memorytool "memoryflow/internal/ai/tools/memory"
	systemtool "memoryflow/internal/ai/tools/system"
	"memoryflow/internal/domain/model"

	"github.com/gin-gonic/gin"
)

type fakeAgentTool struct{}

func (fakeAgentTool) Name() string        { return systemtool.ToolGetCurrentTime }
func (fakeAgentTool) Description() string { return "time" }
func (fakeAgentTool) Call(context.Context, map[string]any) (string, error) {
	return `{"date":"2026-06-01"}`, nil
}

type dynamicAPIProjectAgent struct{}

func (dynamicAPIProjectAgent) Invoke(_ context.Context, input project_pipeline.ProjectAgentInput) (*project_pipeline.ProjectAgentOutput, error) {
	if input.Intent == "project_handoff" {
		return &project_pipeline.ProjectAgentOutput{
			Answer:  "项目交接摘要",
			Project: model.Project{Name: "MemoryFlow", RepoOwner: "vanillaxi", RepoName: "MemoryFlow"},
			UsedTools: []string{
				systemtool.ToolGetCurrentTime,
				githubtools.ToolGetRecentCommits,
				githubtools.ToolGetRecentIssues,
				githubtools.ToolGetPullRequests,
				memorytool.ToolQueryLongTermMemory,
			},
		}, nil
	}
	tool := githubtools.ToolGetRecentCommits
	normalized := strings.ToLower(input.Message)
	if strings.Contains(normalized, "issue") || strings.Contains(normalized, "未处理") || strings.Contains(normalized, "待处理") {
		tool = githubtools.ToolGetRecentIssues
	}
	if strings.Contains(normalized, "pr") || strings.Contains(normalized, "pull request") {
		tool = githubtools.ToolGetPullRequests
	}
	return &project_pipeline.ProjectAgentOutput{
		Answer:    "项目进展",
		Project:   model.Project{Name: "MemoryFlow", RepoOwner: "vanillaxi", RepoName: "MemoryFlow"},
		UsedTools: []string{tool},
	}, nil
}

type fakeAgentSummaryModel struct{}

func (fakeAgentSummaryModel) GenerateWithSystem(context.Context, string, string) (string, error) {
	return "你好", nil
}

func TestAgentChatRoutesProjectQuestionToProjectPipeline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	currentAgent := agent.NewAgent(memorytools.NewToolRegistry(), fakeAgentSummaryModel{}, fakeAgentPipeline{})
	currentAgent.SetProjectAgent(dynamicAPIProjectAgent{})
	router := gin.New()
	router.POST("/agent/chat", NewAgentHandler(currentAgent).Chat)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/agent/chat", bytes.NewBufferString(`{"message":"我的 MemoryFlow 最近做到哪了？"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	var output agent.ChatOutput
	if err := json.Unmarshal(recorder.Body.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK || output.Pipeline != "project_pipeline" || output.Project == nil || output.Project.Name != "MemoryFlow" {
		t.Fatalf("status=%d output=%#v", recorder.Code, output)
	}
}

func TestAgentChatRoutesIssueQuestionToProjectPipeline(t *testing.T) {
	output := postAgentChat(t, `{"message":"MemoryFlow 还有哪些 issue 没处理？"}`)
	if output.Pipeline != "project_pipeline" || output.Intent != "project_issue_status" || !containsTool(output.UsedTools, githubtools.ToolGetRecentIssues) {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestAgentChatRoutesPRQuestionToProjectPipeline(t *testing.T) {
	output := postAgentChat(t, `{"message":"MemoryFlow 最近有哪些 PR？"}`)
	if output.Pipeline != "project_pipeline" || output.Intent != "project_pr_status" || !containsTool(output.UsedTools, githubtools.ToolGetPullRequests) {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestAgentChatRoutesProgressQuestionToProjectPipeline(t *testing.T) {
	output := postAgentChat(t, `{"message":"我的 MemoryFlow 最近做到哪了？"}`)
	if output.Pipeline != "project_pipeline" || output.Intent != "project_progress" || !containsTool(output.UsedTools, githubtools.ToolGetRecentCommits) {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestAgentChatRoutesHandoffQuestionToProjectPipeline(t *testing.T) {
	output := postAgentChat(t, `{"message":"帮我总结 MemoryFlow 当前进度，方便开启新聊天"}`)
	if output.Pipeline != "project_pipeline" || output.Intent != "project_handoff" {
		t.Fatalf("unexpected output: %#v", output)
	}
	wantTools := []string{
		systemtool.ToolGetCurrentTime,
		githubtools.ToolGetRecentCommits,
		githubtools.ToolGetRecentIssues,
		githubtools.ToolGetPullRequests,
		memorytool.ToolQueryLongTermMemory,
	}
	for _, want := range wantTools {
		if !containsTool(output.UsedTools, want) {
			t.Fatalf("missing tool %q in %#v", want, output.UsedTools)
		}
	}
}

func TestAgentChatExplicitProjectPipelineOverride(t *testing.T) {
	output := postAgentChat(t, `{"message":"你好","pipeline":"project"}`)
	if output.Pipeline != "project_pipeline" || output.Intent != "project_progress" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func postAgentChat(t *testing.T, body string) agent.ChatOutput {
	t.Helper()
	gin.SetMode(gin.TestMode)
	currentAgent := agent.NewAgent(memorytools.NewToolRegistry(), fakeAgentSummaryModel{}, fakeAgentPipeline{})
	currentAgent.SetProjectAgent(dynamicAPIProjectAgent{})
	router := gin.New()
	router.POST("/agent/chat", NewAgentHandler(currentAgent).Chat)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/agent/chat", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	var output agent.ChatOutput
	if err := json.Unmarshal(recorder.Body.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d output=%#v body=%s", recorder.Code, output, recorder.Body.String())
	}
	return output
}

func containsTool(tools []string, want string) bool {
	for _, tool := range tools {
		if tool == want {
			return true
		}
	}
	return false
}

type fakeAgentPipeline struct{}

func (fakeAgentPipeline) BuildToolCalls(string, string) []agentruntime.ToolCall {
	return []agentruntime.ToolCall{{Name: systemtool.ToolGetCurrentTime}}
}

func TestAgentChatReturnsDirectOutput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := memorytools.NewToolRegistry()
	registry.Register(fakeAgentTool{})
	router := gin.New()
	router.POST("/agent/chat", NewAgentHandler(agent.NewAgent(registry, fakeAgentSummaryModel{}, fakeAgentPipeline{})).Chat)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/agent/chat", bytes.NewBufferString(`{"message":"你好"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	var output agent.ChatOutput
	if err := json.Unmarshal(recorder.Body.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK || output.Answer != "你好" || output.Intent != "general" {
		t.Fatalf("status=%d output=%#v", recorder.Code, output)
	}
}

func TestAgentChatRejectsEmptyMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/agent/chat", NewAgentHandler(nil).Chat)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/agent/chat", bytes.NewBufferString(`{"message":" "}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
