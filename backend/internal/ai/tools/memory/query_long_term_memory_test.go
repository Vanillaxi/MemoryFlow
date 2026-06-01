package memory

import (
	"context"
	"strings"
	"testing"
	"time"

	"memoryflow/internal/ai/retriever"
	"memoryflow/internal/domain/model"
)

type fakeRetriever struct {
	query string
	opt   retriever.RetrieveOptions
}

func (f *fakeRetriever) Retrieve(_ context.Context, query string, opt retriever.RetrieveOptions) ([]retriever.RetrievedMemory, error) {
	f.query, f.opt = query, opt
	return []retriever.RetrievedMemory{{Memory: model.MemoryItem{ID: 1}, Score: 0.8}}, nil
}

type fakeMemoryService struct {
	rangeLimit int
	start      time.Time
	end        time.Time
	rangeItems []*model.MemoryItem
	detail     *model.MemoryItem
}

func (f *fakeMemoryService) ListByTimeRange(_ context.Context, start, end time.Time, limit int) ([]*model.MemoryItem, error) {
	f.start, f.end, f.rangeLimit = start, end, limit
	return f.rangeItems, nil
}

func (f *fakeMemoryService) GetByID(_ context.Context, _ uint) (*model.MemoryItem, error) {
	return f.detail, nil
}

func TestQueryLongTermMemoryModes(t *testing.T) {
	retrieverClient := &fakeRetriever{}
	service := &fakeMemoryService{rangeItems: []*model.MemoryItem{{ID: 2, Tags: `["Eino"]`, Mood: "专注", Summary: "整理工具"}}}

	semantic, err := QueryLongTermMemory(context.Background(), retrieverClient, service, QueryLongTermMemoryInput{Query: "Eino"})
	if err != nil {
		t.Fatal(err)
	}
	if semantic.Mode != ModeSemantic || semantic.Evidence[0].MemoryID != 1 || retrieverClient.opt.TopK != 20 {
		t.Fatalf("unexpected semantic output: %#v", semantic)
	}

	timeline, err := QueryLongTermMemory(context.Background(), retrieverClient, service, QueryLongTermMemoryInput{From: "2026-05-01", To: "2026-05-31"})
	if err != nil {
		t.Fatal(err)
	}
	if timeline.Mode != ModeTimeline || timeline.Items[0].MemoryID != 2 {
		t.Fatalf("unexpected timeline output: %#v", timeline)
	}

	aggregate, err := QueryLongTermMemory(context.Background(), retrieverClient, service, QueryLongTermMemoryInput{From: "2026-05-01", To: "2026-05-31", Mode: ModeAggregate})
	if err != nil {
		t.Fatal(err)
	}
	if aggregate.Aggregation == nil || aggregate.Aggregation.Count != 1 {
		t.Fatalf("unexpected aggregate output: %#v", aggregate)
	}
}

func TestQueryLongTermMemoryDefaultsCapsAndValidation(t *testing.T) {
	service := &fakeMemoryService{}
	if _, err := QueryLongTermMemory(context.Background(), &fakeRetriever{}, service, QueryLongTermMemoryInput{}); err != nil {
		t.Fatal(err)
	}
	if service.rangeLimit != 20 || service.start.IsZero() || service.end.IsZero() {
		t.Fatalf("unexpected defaults: limit=%d from=%s to=%s", service.rangeLimit, service.start, service.end)
	}

	if _, err := QueryLongTermMemory(context.Background(), &fakeRetriever{}, service, QueryLongTermMemoryInput{From: "2026-05-01", To: "2026-05-31", Limit: 500}); err != nil {
		t.Fatal(err)
	}
	if service.rangeLimit != 100 {
		t.Fatalf("limit = %d, want 100", service.rangeLimit)
	}

	for _, input := range []QueryLongTermMemoryInput{{From: "bad", To: "2026-05-31"}, {From: "2026-06-01", To: "2026-05-31"}} {
		if _, err := QueryLongTermMemory(context.Background(), &fakeRetriever{}, service, input); err == nil {
			t.Fatalf("expected error for %#v", input)
		}
	}
}

func TestQueryLongTermMemoryOmitsContentText(t *testing.T) {
	service := &fakeMemoryService{rangeItems: []*model.MemoryItem{{ID: 1, ContentText: strings.Repeat("content", 100), Summary: strings.Repeat("summary", 100)}}}
	output, err := QueryLongTermMemory(context.Background(), &fakeRetriever{}, service, QueryLongTermMemoryInput{From: "2026-05-01", To: "2026-05-31"})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Items) != 1 || len([]rune(output.Items[0].Summary)) > MaxSummaryLength+3 {
		t.Fatalf("unexpected compact output: %#v", output.Items)
	}
}

func TestQueryLongTermMemoryAggregateBuildsReviewEvidence(t *testing.T) {
	later := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
	earlier := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	got := AggregateMemories([]*model.MemoryItem{
		{ID: 2, OccurredAt: later, Summary: "完成测试", Tags: `["MemoryFlow","测试"]`, Mood: "开心", ImportanceScore: 9},
		{ID: 1, OccurredAt: earlier, ContentText: "开始开发", Tags: `["MemoryFlow"]`, Mood: "专注", ImportanceScore: 5},
	})

	if got.Count != 2 || len(got.Tags) != 2 || got.Tags[0] != "MemoryFlow" || len(got.Highlights) != 2 || got.Highlights[0] != "完成测试" {
		t.Fatalf("unexpected aggregation: %#v", got)
	}
	if strings.Index(got.MemoryList, "开始开发") > strings.Index(got.MemoryList, "完成测试") {
		t.Fatalf("memory list is not ordered by occurred_at: %s", got.MemoryList)
	}
}
