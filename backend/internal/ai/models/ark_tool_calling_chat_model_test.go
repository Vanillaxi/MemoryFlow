package models

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestGenerate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl-test",
			"model": "test-model",
			"choices": [{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": null,
					"tool_calls": [{
						"id": "call_123",
						"type": "function",
						"function": {
							"name": "query_long_term_memory",
							"arguments": "{\"query\":\"embedding\"}"
						}
					}]
				},
				"finish_reason": "tool_calls"
			}]
		}`))
	}))
	defer server.Close()

	model := NewArkToolCallingChatModel(Config{
		BaseURL:   server.URL,
		APIKey:    "dummy_api_key",
		ModelName: "test-model",
	})

	msgs := []*schema.Message{
		{Role: schema.User, Content: "测试生成 tool call 的消息"},
	}

	out, err := model.Generate(context.Background(), msgs)
	if err != nil {
		t.Fatal(err)
	}

	if len(out.ToolCalls) != 1 {
		t.Fatalf("expected one tool call, got %d", len(out.ToolCalls))
	}
	if out.ToolCalls[0].Function.Name != "query_long_term_memory" {
		t.Fatalf("unexpected tool name: %s", out.ToolCalls[0].Function.Name)
	}
	if out.ResponseMeta == nil || out.ResponseMeta.FinishReason != "tool_calls" {
		t.Fatalf("unexpected response meta: %+v", out.ResponseMeta)
	}
}

func TestBindTools(t *testing.T) {
	model := NewArkToolCallingChatModel(Config{
		BaseURL:   "http://localhost:8080",
		APIKey:    "dummy_api_key",
		ModelName: "test-model",
	})

	err := model.BindTools([]*schema.ToolInfo{
		{Name: "query_long_term_memory", Desc: "测试工具"},
	})
	if err != nil {
		t.Fatal(err)
	}
}
