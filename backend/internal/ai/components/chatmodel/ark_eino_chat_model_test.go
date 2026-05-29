package chatmodel

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestGenerate(t *testing.T) {
	model := NewArkEinoChatModel("http://localhost:8080", "dummy_api_key", "test-model")

	msgs := []*schema.Message{
		{Role: schema.User, Content: "测试生成 tool call 的消息"},
	}

	out, err := model.Generate(context.Background(), msgs)
	if err != nil {
		t.Fatal(err)
	}

	if out.Content == "" {
		t.Fatal("expected non-empty content")
	}
}

func TestBindTools(t *testing.T) {
	model := NewArkEinoChatModel("http://localhost:8080", "dummy_api_key", "test-model")

	err := model.BindTools([]*schema.ToolInfo{
		{Name: "ask_memory", Desc: "测试工具"},
	})
	if err != nil {
		t.Fatal(err)
	}
}
