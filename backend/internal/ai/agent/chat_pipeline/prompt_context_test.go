package chat_pipeline

import (
	"context"
	"strings"
	"testing"
	"time"

	memorytools "memoryflow/internal/ai/tools"
	"memoryflow/internal/domain/model"
)

type fakeMemoryContextService struct {
	memories []*model.MemoryItem
	err      error
	limit    int
}

func (f *fakeMemoryContextService) ListByTimeRange(_ context.Context, _, _ time.Time, limit int) ([]*model.MemoryItem, error) {
	f.limit = limit
	return f.memories, f.err
}

func (f *fakeMemoryContextService) GetByID(_ context.Context, _ uint) (*model.MemoryItem, error) {
	return nil, nil
}

type fakeChatModel struct {
	answer string
	err    error
	prompt string
}

func (f *fakeChatModel) Generate(_ context.Context, prompt string) (string, error) {
	f.prompt = prompt
	return f.answer, f.err
}

func TestChatPipelineSummaryUsesAggregateEvidence(t *testing.T) {
	service := &fakeMemoryContextService{memories: []*model.MemoryItem{{Summary: "完成测试", Tags: `["项目"]`, Mood: "开心"}}}
	modelClient := &fakeChatModel{answer: " 回顾总结 "}
	pipeline := &Pipeline{memoryService: service, chatModel: modelClient}

	got, err := pipeline.Summarize(context.Background(), SummaryInput{From: summaryTestFrom(), To: summaryTestTo()})
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

func TestChatPipelineSummaryReportsEmptyEvidence(t *testing.T) {
	modelClient := &fakeChatModel{answer: "should not be called"}
	got, err := (&Pipeline{memoryService: &fakeMemoryContextService{}, chatModel: modelClient}).Summarize(context.Background(), SummaryInput{From: summaryTestFrom(), To: summaryTestTo()})
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != 0 || !strings.Contains(got.Summary, "没有记录") || modelClient.prompt != "" {
		t.Fatalf("unexpected output=%#v prompt=%q", got, modelClient.prompt)
	}
}

func TestBuildSummaryPromptKeepsEvidenceConstraints(t *testing.T) {
	got := BuildSummaryPrompt(
		summaryTestFrom(),
		summaryTestTo(),
		memorytools.MemoryAggregation{Count: 2, Highlights: []string{"完成测试"}, MemoryList: "- memory"},
	)
	for _, want := range []string{"不要编造", "依据有限", "完成测试"} {
		if !strings.Contains(got, want) {
			t.Fatalf("prompt missing %q: %s", want, got)
		}
	}
}

func summaryTestFrom() time.Time {
	return time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
}

func summaryTestTo() time.Time {
	return time.Date(2026, 5, 31, 23, 59, 59, 0, time.UTC)
}
