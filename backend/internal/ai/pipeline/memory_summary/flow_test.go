package memory_summary

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"memoryflow/internal/domain/model"
)

type fakeSummaryMemoryService struct {
	memories []*model.MemoryItem
	err      error
	limit    int
}

func (f *fakeSummaryMemoryService) ListByTimeRange(_ context.Context, _, _ time.Time, limit int) ([]*model.MemoryItem, error) {
	f.limit = limit
	return f.memories, f.err
}

type fakeSummaryChatModel struct {
	answer string
	err    error
	prompt string
}

func (f *fakeSummaryChatModel) Generate(_ context.Context, prompt string) (string, error) {
	f.prompt = prompt
	return f.answer, f.err
}

func TestPipelineInvoke(t *testing.T) {
	service := &fakeSummaryMemoryService{memories: []*model.MemoryItem{{Summary: "完成测试", Tags: `["项目"]`, Mood: "开心"}}}
	modelClient := &fakeSummaryChatModel{answer: " 回顾总结 "}
	pipeline := NewPipeline(service, modelClient)

	got, err := pipeline.Invoke(context.Background(), SummaryInput{From: summaryTestFrom(), To: summaryTestTo()})
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary != "回顾总结" || got.Count != 1 || service.limit != 100 {
		t.Fatalf("unexpected output: %#v limit=%d", got, service.limit)
	}
	if !strings.Contains(modelClient.prompt, "完成测试") {
		t.Fatalf("unexpected prompt: %s", modelClient.prompt)
	}
}

func TestPipelineInvokeEmptyMemories(t *testing.T) {
	modelClient := &fakeSummaryChatModel{answer: "should not be called"}
	got, err := NewPipeline(&fakeSummaryMemoryService{}, modelClient).Invoke(context.Background(), SummaryInput{From: summaryTestFrom(), To: summaryTestTo()})
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != 0 || !strings.Contains(got.Summary, "没有记录") || modelClient.prompt != "" {
		t.Fatalf("unexpected output=%#v prompt=%q", got, modelClient.prompt)
	}
}

func TestPipelineInvokeChatModelError(t *testing.T) {
	pipeline := NewPipeline(
		&fakeSummaryMemoryService{memories: []*model.MemoryItem{{Summary: "memory"}}},
		&fakeSummaryChatModel{err: errors.New("model failed")},
	)
	if _, err := pipeline.Invoke(context.Background(), SummaryInput{From: summaryTestFrom(), To: summaryTestTo()}); err == nil {
		t.Fatal("expected model error")
	}
}

func TestPipelineInvokeMemoryServiceError(t *testing.T) {
	pipeline := NewPipeline(&fakeSummaryMemoryService{err: errors.New("query failed")}, &fakeSummaryChatModel{})
	if _, err := pipeline.Invoke(context.Background(), SummaryInput{From: summaryTestFrom(), To: summaryTestTo()}); err == nil {
		t.Fatal("expected query error")
	}
}

func summaryTestFrom() time.Time {
	return time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
}

func summaryTestTo() time.Time {
	return time.Date(2026, 5, 31, 23, 59, 59, 0, time.UTC)
}
