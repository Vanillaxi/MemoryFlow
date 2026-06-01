package memory

import (
	"context"
	"errors"
	"testing"

	"memoryflow/internal/domain/model"
)

type missingMemoryService struct {
	*fakeMemoryService
}

func (missingMemoryService) GetByID(context.Context, uint) (*model.MemoryItem, error) {
	return nil, errors.New("memory not found")
}

func TestGetMemoryDetailValidatesMemoryID(t *testing.T) {
	service := &fakeMemoryService{detail: &model.MemoryItem{ID: 7, ContentText: "完整内容"}}
	output, err := GetMemoryDetail(context.Background(), service, GetMemoryDetailInput{MemoryID: 7})
	if err != nil || output.ID != 7 {
		t.Fatalf("unexpected detail=%#v err=%v", output, err)
	}
	if _, err := GetMemoryDetail(context.Background(), service, GetMemoryDetailInput{}); err == nil {
		t.Fatal("expected zero memory_id error")
	}
	if _, err := GetMemoryDetail(context.Background(), missingMemoryService{service}, GetMemoryDetailInput{MemoryID: 99}); err == nil {
		t.Fatal("expected missing memory error")
	}
}

func TestQueryLongTermMemoryToolEmitsTrace(t *testing.T) {
	var events []string
	trace := func(_ context.Context, name string, event string, _ any, _ any, _ error) {
		events = append(events, name+":"+event)
	}
	currentTool := NewQueryLongTermMemoryTool(&fakeRetriever{}, &fakeMemoryService{}, trace)
	if _, err := currentTool.InvokableRun(context.Background(), `{"query":"Eino","mode":"semantic"}`); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0] != ToolQueryLongTermMemory+":tool_start" || events[1] != ToolQueryLongTermMemory+":tool_end" {
		t.Fatalf("unexpected events: %#v", events)
	}
}

func TestAggregateMemoryToolReturnsJSON(t *testing.T) {
	service := &fakeMemoryService{rangeItems: []*model.MemoryItem{{ID: 1, Summary: "完成 Tool MVP"}}}
	currentTool := NewAggregateMemoryTool(service, nil)
	output, err := currentTool.Call(context.Background(), map[string]any{
		"from": "2026-05-01",
		"to":   "2026-05-31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if output == "" {
		t.Fatal("expected aggregate output")
	}
}
