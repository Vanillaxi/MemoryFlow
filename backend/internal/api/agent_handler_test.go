package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"memoryflow/internal/ai/agent"
	agentruntime "memoryflow/internal/ai/agent/runtime"
	memorytools "memoryflow/internal/ai/tools"
	systemtool "memoryflow/internal/ai/tools/system"

	"github.com/gin-gonic/gin"
)

type fakeAgentTool struct{}

func (fakeAgentTool) Name() string        { return systemtool.ToolGetCurrentTime }
func (fakeAgentTool) Description() string { return "time" }
func (fakeAgentTool) Call(context.Context, map[string]any) (string, error) {
	return `{"date":"2026-06-01"}`, nil
}

type fakeAgentSummaryModel struct{}

func (fakeAgentSummaryModel) GenerateWithSystem(context.Context, string, string) (string, error) {
	return "你好", nil
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
