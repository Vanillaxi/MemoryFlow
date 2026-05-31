package tools

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

func TestGetCurrentTimeReturnsNowDateAndTimezone(t *testing.T) {
	output := GetCurrentTime()
	if output.Now.IsZero() || output.Date == "" || output.TimeZone == "" {
		t.Fatalf("unexpected current time: %#v", output)
	}
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
