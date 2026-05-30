package memory_summary_pipeline

import (
	"strings"
	"testing"
	"time"

	"memoryflow/internal/domain/model"
)

func TestAggregateMemories(t *testing.T) {
	later := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
	earlier := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	got := AggregateMemories([]*model.MemoryItem{
		{ID: 2, OccurredAt: later, Summary: "完成测试", Tags: `["MemoryFlow","测试"]`, Mood: "开心", ImportanceScore: 9},
		{ID: 1, OccurredAt: earlier, ContentText: "开始开发", Tags: `["MemoryFlow"]`, Mood: "专注", ImportanceScore: 5},
	})

	if got.Count != 2 {
		t.Fatalf("Count = %d, want 2", got.Count)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "MemoryFlow" {
		t.Fatalf("unexpected Tags: %#v", got.Tags)
	}
	if len(got.Highlights) != 2 || got.Highlights[0] != "完成测试" {
		t.Fatalf("unexpected Highlights: %#v", got.Highlights)
	}
	if strings.Index(got.MemoryList, "开始开发") > strings.Index(got.MemoryList, "完成测试") {
		t.Fatalf("memory list is not ordered by occurred_at: %s", got.MemoryList)
	}
}

func TestAggregateMemoriesInvalidTagsDoesNotPanic(t *testing.T) {
	got := AggregateMemories([]*model.MemoryItem{{Tags: "项目, 测试", ContentText: "content"}})
	if len(got.Tags) != 2 {
		t.Fatalf("unexpected Tags: %#v", got.Tags)
	}
}

func TestAggregateMemoriesEmpty(t *testing.T) {
	got := AggregateMemories(nil)
	if got.Count != 0 || got.MemoryList != "" {
		t.Fatalf("unexpected aggregation: %#v", got)
	}
}
