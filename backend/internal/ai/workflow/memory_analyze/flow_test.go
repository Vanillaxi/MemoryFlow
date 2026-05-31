package memory_analyze

import (
	"context"
	"strings"
	"testing"
)

type fakeChatModel struct {
	prompt string
	raw    string
}

func (m *fakeChatModel) Generate(_ context.Context, prompt string) (string, error) {
	m.prompt = prompt
	return m.raw, nil
}

func TestWorkflowInvokeText(t *testing.T) {
	model := &fakeChatModel{
		raw: `{"summary":"完成入口整理","tags":["开发"],"mood":"成就感","importance_score":8}`,
	}
	workflow := NewWorkflow(model)

	result, err := workflow.Invoke(context.Background(), AnalyzeInput{
		Type:        TypeText,
		ContentText: "完成 MemoryFlow 入口整理",
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if result.Summary != "完成入口整理" || result.Mood != "positive" || result.ImportanceScore != 0.8 {
		t.Fatalf("Invoke() result = %+v", result)
	}
	if !strings.Contains(model.prompt, "完成 MemoryFlow 入口整理") {
		t.Fatalf("prompt = %q", model.prompt)
	}
}

func TestWorkflowInvokeImage(t *testing.T) {
	workflow := NewWorkflow(nil)

	result, err := workflow.Invoke(context.Background(), AnalyzeInput{
		Type:     TypeImage,
		ImageURL: "/uploads/photo.jpg",
		Location: "上海",
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if result.Summary != "这是一条图片记忆。" || result.Mood != "neutral" {
		t.Fatalf("Invoke() result = %+v", result)
	}
	if !contains(result.Tags, "上海") {
		t.Fatalf("Invoke() tags = %v", result.Tags)
	}
}

func TestWorkflowInvokeMixed(t *testing.T) {
	workflow := NewWorkflow(nil)

	result, err := workflow.Invoke(context.Background(), AnalyzeInput{
		Type:        TypeMixed,
		ImageURL:    "/uploads/photo.jpg",
		ContentText: "项目白板",
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if result.Summary != "这是一条带有文字说明的图片记忆：项目白板" {
		t.Fatalf("Invoke() summary = %q", result.Summary)
	}
	if !contains(result.Tags, "图片说明") {
		t.Fatalf("Invoke() tags = %v", result.Tags)
	}
}

func TestWorkflowInvokeInfersMixedType(t *testing.T) {
	workflow := NewWorkflow(nil)

	result, err := workflow.Invoke(context.Background(), AnalyzeInput{
		ImageURL:    "/uploads/photo.jpg",
		ContentText: "项目白板",
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if result.Summary != "这是一条带有文字说明的图片记忆：项目白板" {
		t.Fatalf("Invoke() summary = %q", result.Summary)
	}
}

func TestParseAnalyzeResultToleratesWrappedJSON(t *testing.T) {
	result, err := ParseAnalyzeResult("结果如下：\n```json\n{\"summary\":\"记录\",\"tags\":[],\"mood\":\"未知\",\"importance_score\":12}\n```")
	if err != nil {
		t.Fatalf("ParseAnalyzeResult() error = %v", err)
	}
	if len(result.Tags) != 1 || result.Tags[0] != "生活记录" || result.Mood != "neutral" || result.ImportanceScore != 1 {
		t.Fatalf("ParseAnalyzeResult() result = %+v", result)
	}
}

func TestWorkflowInvokeRejectsEmptyInput(t *testing.T) {
	workflow := NewWorkflow(nil)

	if _, err := workflow.Invoke(context.Background(), AnalyzeInput{}); err == nil {
		t.Fatal("Invoke() error = nil, want non-nil")
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
