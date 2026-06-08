package mcp

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
	"memoryflow/internal/ai/tools"
	githubtool "memoryflow/internal/ai/tools/github"
	memorytool "memoryflow/internal/ai/tools/memory"
	webtool "memoryflow/internal/ai/tools/web"
	"memoryflow/internal/domain/model"

	"github.com/gin-gonic/gin"
)

type fakeSummaryModel struct{}

func (fakeSummaryModel) GenerateWithSystem(context.Context, string, string) (string, error) {
	return "summary", nil
}

type fakePipeline struct{}

func (fakePipeline) BuildToolCalls(string, string) []agentruntime.ToolCall {
	return nil
}

type fakeProjectAgent struct{}

func (fakeProjectAgent) Invoke(_ context.Context, input project_pipeline.ProjectAgentInput) (*project_pipeline.ProjectAgentOutput, error) {
	return &project_pipeline.ProjectAgentOutput{
		Answer:  "项目交接摘要",
		Project: model.Project{ID: 7, Name: "MemoryFlow"},
		UsedTools: []string{
			githubtool.ToolGetRecentCommits,
			githubtool.ToolGetRecentIssues,
			githubtool.ToolGetPullRequests,
			memorytool.ToolQueryLongTermMemory,
		},
		Evidence: []project_pipeline.Evidence{{Source: memorytool.ToolQueryLongTermMemory, Detail: `{"evidence":[{"memory_id":1,"summary":"项目进展"}]}`}},
		RawToolCalls: []project_pipeline.ToolCallLog{
			{Name: memorytool.ToolQueryLongTermMemory, Args: map[string]any{"query": input.Message}, Result: `{"evidence":[{"memory_id":1,"summary":"项目进展"}]}`},
		},
	}, nil
}

type fakeTool struct {
	name   string
	result string
	err    error
}

func (t fakeTool) Name() string        { return t.name }
func (t fakeTool) Description() string { return t.name }
func (t fakeTool) Call(context.Context, map[string]any) (string, error) {
	if t.err != nil {
		return "", t.err
	}
	return t.result, nil
}

func TestMCPProjectHandoffSummary(t *testing.T) {
	currentAgent := agent.NewAgent(tools.NewToolRegistry(), fakeSummaryModel{}, fakePipeline{})
	currentAgent.SetProjectAgent(fakeProjectAgent{})
	server := NewServer(currentAgent, tools.NewToolRegistry(), "local-token")

	output := postRPC(t, server, `{"method":"project_handoff_summary","message":"生成 MemoryFlow 交接摘要","token":"local-token"}`)
	if output.Pipeline != "project_pipeline" || output.Project == nil || output.Project.Name != "MemoryFlow" {
		t.Fatalf("unexpected output: %#v", output)
	}
	if !contains(output.UsedTools, memorytool.ToolQueryLongTermMemory) || len(output.Evidence) == 0 || len(output.RawToolCalls) == 0 {
		t.Fatalf("missing trace fields: %#v", output)
	}
}

func TestMCPRejectsEmptyConfiguredToken(t *testing.T) {
	server := NewServer(nil, tools.NewToolRegistry(), "")
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/mcp/rpc", server.Handler())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp/rpc", bytes.NewBufferString(`{"method":"web_search","message":"MemoryFlow","token":"local-token"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), "mcp token is not configured") {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestMCPProjectHandoffIntentAlias(t *testing.T) {
	currentAgent := agent.NewAgent(tools.NewToolRegistry(), fakeSummaryModel{}, fakePipeline{})
	currentAgent.SetProjectAgent(fakeProjectAgent{})
	server := NewServer(currentAgent, tools.NewToolRegistry(), "local-token")

	output := postRPC(t, server, `{"intent":"project_handoff","message":"生成 MemoryFlow 交接摘要","token":"local-token"}`)
	if output.Pipeline != "project_pipeline" || output.Project == nil || output.Project.Name != "MemoryFlow" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestMCPAuthorizationBearerCanEnterAgentChat(t *testing.T) {
	currentAgent := agent.NewAgent(tools.NewToolRegistry(), fakeSummaryModel{}, fakePipeline{})
	currentAgent.SetProjectAgent(fakeProjectAgent{})
	server := NewServer(currentAgent, tools.NewToolRegistry(), "local-token")

	output := postRPCWithHeaders(t, server, `{"intent":"project_handoff","message":"生成 MemoryFlow 交接摘要"}`, map[string]string{
		"Authorization": "Bearer local-token",
	})
	if output.Pipeline != "project_pipeline" || output.Intent != "project_handoff" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestMCPMemoryDetailSanitizesImageMetadata(t *testing.T) {
	registry := tools.NewToolRegistry()
	registry.Register(fakeTool{
		name:   memorytool.ToolGetMemoryDetail,
		result: `{"id":42,"type":"image","content_text":"raw OCR text should not leak","image_url":"/uploads/a.png","summary":"白板架构图","tags":"[\"design\"]"}`,
	})
	server := NewServer(nil, registry, "local-token")

	output := postRPC(t, server, `{"method":"get_memory_detail","message":"图片详情","memory_id":42,"token":"local-token"}`)
	if output.Pipeline != PipelineMCPToolRegistry || !contains(output.UsedTools, memorytool.ToolGetMemoryDetail) {
		t.Fatalf("unexpected output: %#v", output)
	}
	detail := output.Evidence[0].Detail
	if strings.Contains(detail, "raw OCR text") || !strings.Contains(detail, `"image_url"`) || !strings.Contains(detail, `"caption":"白板架构图"`) {
		t.Fatalf("image memory was not sanitized: %s", detail)
	}
}

func TestMCPTokenDoesNotAppearInResponseEvidenceOrRawToolCalls(t *testing.T) {
	registry := tools.NewToolRegistry()
	registry.Register(fakeTool{
		name:   webtool.ToolWebSearch,
		result: `{"query":"MemoryFlow","token":"local-token","api_key":"secret","results":[{"title":"ok","url":"https://example.com","authorization":"Bearer local-token"}]}`,
	})
	server := NewServer(nil, registry, "local-token")

	recorder := postRPCRecorder(t, server, `{"method":"web_search","message":"MemoryFlow","token":"local-token","params":{"query":"MemoryFlow","token":"local-token","api_key":"secret"}}`, nil)
	body := recorder.Body.String()
	if strings.Contains(body, "local-token") || strings.Contains(body, "api_key") || strings.Contains(body, "authorization") {
		t.Fatalf("response leaked secret material: %s", body)
	}
}

func TestMCPQueryLongTermMemoryReturnsEvidence(t *testing.T) {
	registry := tools.NewToolRegistry()
	registry.Register(fakeTool{
		name:   memorytool.ToolQueryLongTermMemory,
		result: `{"mode":"semantic","evidence":[{"memory_id":9,"summary":"MCP 原型完成"}]}`,
	})
	server := NewServer(nil, registry, "local-token")

	output := postRPC(t, server, `{"method":"query_long_term_memory","message":"MCP 原型","limit":5,"token":"local-token"}`)
	if output.Answer == "" || !contains(output.UsedTools, memorytool.ToolQueryLongTermMemory) || !strings.Contains(output.Evidence[0].Detail, "MCP 原型完成") {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestMCPWebFetchRejectsLocalhost(t *testing.T) {
	registry := tools.NewToolRegistry()
	registry.Register(webtool.NewWebFetchTool(nil, nil))
	server := NewServer(nil, registry, "local-token")

	output := postRPC(t, server, `{"method":"web_fetch","message":"http://localhost:8080/health","token":"local-token"}`)
	if output.RawToolCalls[0].Error == "" || !strings.Contains(output.Answer, "localhost") {
		t.Fatalf("expected localhost rejection, got %#v", output)
	}
	if output.Pipeline != PipelineMCPToolRegistry || output.Evidence[0].Source != webtool.ToolWebFetch {
		t.Fatalf("missing evidence fields: %#v", output)
	}
}

func TestMCPWebSearchReturnsEvidence(t *testing.T) {
	registry := tools.NewToolRegistry()
	registry.Register(fakeTool{
		name:   webtool.ToolWebSearch,
		result: `{"query":"MemoryFlow","results":[{"title":"MemoryFlow docs","url":"https://example.com"}]}`,
	})
	server := NewServer(nil, registry, "local-token")

	output := postRPC(t, server, `{"method":"web_search","message":"MemoryFlow docs","token":"local-token"}`)
	if output.Answer == "" || !contains(output.UsedTools, webtool.ToolWebSearch) || !strings.Contains(output.Evidence[0].Detail, "MemoryFlow docs") {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestMCPGitHubToolReturnsEvidenceAndRawCall(t *testing.T) {
	registry := tools.NewToolRegistry()
	registry.Register(fakeTool{
		name:   githubtool.ToolGetRecentCommits,
		result: `{"repository":"vanillaxi/MemoryFlow","commits":[{"sha":"abc","message":"add mcp prototype"}]}`,
	})
	server := NewServer(nil, registry, "local-token")

	output := postRPC(t, server, `{"method":"get_recent_commits","message":"最近提交","params":{"repository":"vanillaxi/MemoryFlow"},"limit":3,"token":"local-token"}`)
	if output.Answer == "" || !contains(output.UsedTools, githubtool.ToolGetRecentCommits) || len(output.Evidence) != 1 || len(output.RawToolCalls) != 1 {
		t.Fatalf("unexpected output: %#v", output)
	}
	if output.RawToolCalls[0].Args["repository"] != "vanillaxi/MemoryFlow" || output.RawToolCalls[0].Args["limit"].(float64) != 3 {
		t.Fatalf("unexpected raw args: %#v", output.RawToolCalls[0].Args)
	}
}

func TestMCPRecentMemoriesAppliesDaysWindow(t *testing.T) {
	registry := tools.NewToolRegistry()
	registry.Register(fakeTool{name: memorytool.ToolQueryLongTermMemory, result: `{"mode":"timeline","items":[]}`})
	server := NewServer(nil, registry, "local-token")

	output := postRPC(t, server, `{"method":"recent_memories","message":"最近记忆","days":3,"limit":5,"token":"local-token"}`)
	args := output.RawToolCalls[0].Args
	if args["mode"] != memorytool.ModeTimeline || args["from"] == "" || args["to"] == "" || args["limit"].(float64) != 5 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestMCPRejectsInvalidToken(t *testing.T) {
	server := NewServer(nil, tools.NewToolRegistry(), "local-token")
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/mcp/rpc", server.Handler())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp/rpc", bytes.NewBufferString(`{"method":"web_search","message":"MemoryFlow","token":"bad"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func postRPC(t *testing.T, server *Server, body string) Response {
	t.Helper()
	return postRPCWithHeaders(t, server, body, nil)
}

func postRPCWithHeaders(t *testing.T, server *Server, body string, headers map[string]string) Response {
	t.Helper()
	recorder := postRPCRecorder(t, server, body, headers)
	var output Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	return output
}

func postRPCRecorder(t *testing.T, server *Server, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/mcp/rpc", server.Handler())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp/rpc", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	return recorder
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
