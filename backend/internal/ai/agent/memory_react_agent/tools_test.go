package memory_react_agent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"memoryflow/internal/ai/agent/memory_chat_pipeline"
	"memoryflow/internal/ai/agent/memory_summary_pipeline"
	"memoryflow/internal/ai/retriever"
	"memoryflow/internal/domain/model"
	"memoryflow/internal/domain/service"
)

type fakeChatPipeline struct {
	input memory_chat_pipeline.ChatInput
}

func (f *fakeChatPipeline) Run(_ context.Context, input memory_chat_pipeline.ChatInput) (*memory_chat_pipeline.ChatOutput, error) {
	f.input = input
	return &memory_chat_pipeline.ChatOutput{Answer: "answer"}, nil
}

type fakeAgentRetriever struct {
	query string
	opt   retriever.RetrieveOptions
}

func (f *fakeAgentRetriever) Retrieve(_ context.Context, query string, opt retriever.RetrieveOptions) ([]retriever.RetrievedMemory, error) {
	f.query, f.opt = query, opt
	return []retriever.RetrievedMemory{{Memory: model.MemoryItem{ID: 1}, Score: 0.8}}, nil
}

type fakeAgentMemoryService struct {
	recentLimit int
	rangeLimit  int
	start       time.Time
	end         time.Time
	rangeItems  []*model.MemoryItem
	detail      *model.MemoryItem
}

func (f *fakeAgentMemoryService) ListRecent(_ context.Context, limit int) ([]model.MemoryItem, error) {
	f.recentLimit = limit
	return []model.MemoryItem{{ID: 1}}, nil
}

func (f *fakeAgentMemoryService) GetTimeline(_ context.Context, start, end time.Time) ([]service.TimelineGroup, error) {
	f.start, f.end = start, end
	return []service.TimelineGroup{{Date: "2026-05-30"}}, nil
}

func (f *fakeAgentMemoryService) ListByTimeRange(_ context.Context, start, end time.Time, limit int) ([]*model.MemoryItem, error) {
	f.start, f.end, f.rangeLimit = start, end, limit
	return f.rangeItems, nil
}

func (f *fakeAgentMemoryService) GetByID(_ context.Context, id uint) (*model.MemoryItem, error) {
	if f.detail != nil {
		return f.detail, nil
	}
	return &model.MemoryItem{ID: id}, nil
}

type fakeSummaryPipeline struct {
	input memory_summary_pipeline.SummaryInput
	err   error
}

func (f *fakeSummaryPipeline) Invoke(_ context.Context, input memory_summary_pipeline.SummaryInput) (*memory_summary_pipeline.SummaryOutput, error) {
	f.input = input
	if f.err != nil {
		return nil, f.err
	}
	return &memory_summary_pipeline.SummaryOutput{Summary: "summary", Count: 1}, nil
}

func TestToolArgumentParsing(t *testing.T) {
	chat := &fakeChatPipeline{}
	memoryRetriever := &fakeAgentRetriever{}
	memoryService := &fakeAgentMemoryService{}
	summaryPipeline := &fakeSummaryPipeline{}
	agent := &MemoryAgent{chatPipeline: chat, memoryRetriever: memoryRetriever, memoryService: memoryService, summaryPipeline: summaryPipeline}

	if _, err := agent.callAskMemory(context.Background(), map[string]any{"question": " hello ", "top_k": float64(7)}); err != nil {
		t.Fatal(err)
	}
	if chat.input.Question != "hello" || chat.input.TopK != 7 {
		t.Fatalf("unexpected ask input: %#v", chat.input)
	}

	if _, err := agent.callSearchMemory(context.Background(), map[string]any{"query": " project "}); err != nil {
		t.Fatal(err)
	}
	if memoryRetriever.query != "project" || memoryRetriever.opt.TopK != 5 {
		t.Fatalf("unexpected search input: query=%q opt=%#v", memoryRetriever.query, memoryRetriever.opt)
	}

	if _, err := agent.callListRecent(context.Background(), map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if memoryService.recentLimit != 10 {
		t.Fatalf("recent limit = %d, want 10", memoryService.recentLimit)
	}

	if _, err := agent.callGetTimeline(context.Background(), map[string]any{"start": "2026-05-01", "end": "2026-05-30"}); err != nil {
		t.Fatal(err)
	}
	if got := memoryService.start.Format("2006-01-02"); got != "2026-05-01" {
		t.Fatalf("timeline start = %s", got)
	}
	if got := memoryService.end.Format("2006-01-02 15:04:05"); got != "2026-05-30 23:59:59" {
		t.Fatalf("timeline end = %s", got)
	}

	if _, err := agent.callSummarizeMemory(context.Background(), map[string]any{"start": "2026-05-01", "end": "2026-05-31"}); err != nil {
		t.Fatal(err)
	}
	if summaryPipeline.input.Limit != 100 {
		t.Fatalf("summary limit = %d, want 100", summaryPipeline.input.Limit)
	}
	if got := summaryPipeline.input.To.Format("2006-01-02 15:04:05"); got != "2026-05-31 23:59:59" {
		t.Fatalf("summary end = %s", got)
	}
}

func TestBaseToolsExposeExternalCapabilities(t *testing.T) {
	tools := (&MemoryAgent{}).BaseTools()
	if len(tools) != 3 {
		t.Fatalf("len(BaseTools()) = %d, want 3", len(tools))
	}
	want := []string{string(ToolGetCurrentTime), string(ToolQueryLongTermMemory), string(ToolGetMemoryDetail)}
	for i, tool := range tools {
		info, err := tool.Info(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if info.Name != want[i] {
			t.Fatalf("tool[%d] name = %q, want %q", i, info.Name, want[i])
		}
	}
}

func TestDebugListEinoToolsExposeExternalCapabilities(t *testing.T) {
	infos, err := (&MemoryAgent{}).DebugListEinoTools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{string(ToolGetCurrentTime), string(ToolQueryLongTermMemory), string(ToolGetMemoryDetail)}
	if len(infos) != len(want) {
		t.Fatalf("len(DebugListEinoTools()) = %d, want %d", len(infos), len(want))
	}
	for i := range infos {
		if infos[i].Name != want[i] {
			t.Fatalf("tool[%d] name = %q, want %q", i, infos[i].Name, want[i])
		}
	}
}

func TestCallSummarizeMemoryError(t *testing.T) {
	agent := &MemoryAgent{summaryPipeline: &fakeSummaryPipeline{err: errors.New("summary failed")}}
	if _, err := agent.callSummarizeMemory(context.Background(), map[string]any{"start": "2026-05-01", "end": "2026-05-31"}); err == nil {
		t.Fatal("expected summary error")
	}
}

func TestQueryLongTermMemoryToolDebugTrace(t *testing.T) {
	collector := NewTraceCollector("test")
	ctx := ContextWithTraceCollector(context.Background(), collector)
	agent := &MemoryAgent{memoryRetriever: &fakeAgentRetriever{}}

	if _, err := agent.newQueryLongTermMemoryEinoTool().InvokableRun(ctx, `{"query":"Eino","mode":"semantic"}`); err != nil {
		t.Fatal(err)
	}
	steps := collector.Trace().Steps
	if len(steps) != 2 || steps[0].Node != string(ToolQueryLongTermMemory) || steps[0].Event != "tool_start" || steps[1].Event != "tool_end" {
		t.Fatalf("unexpected trace: %#v", steps)
	}
}

func TestGetCurrentTime(t *testing.T) {
	output := (&MemoryAgent{}).GetCurrentTime()
	if output.Now.IsZero() || output.Date == "" || output.TimeZone == "" {
		t.Fatalf("unexpected current time: %#v", output)
	}
}

func TestQueryLongTermMemorySemantic(t *testing.T) {
	memoryRetriever := &fakeAgentRetriever{}
	output, err := (&MemoryAgent{memoryRetriever: memoryRetriever}).QueryLongTermMemory(context.Background(), QueryLongTermMemoryInput{
		Query: "Eino",
		Mode:  longTermMemoryModeSemantic,
	})
	if err != nil {
		t.Fatal(err)
	}
	if output.Mode != longTermMemoryModeSemantic || memoryRetriever.query != "Eino" || memoryRetriever.opt.TopK != 20 {
		t.Fatalf("unexpected output=%#v query=%q opt=%#v", output, memoryRetriever.query, memoryRetriever.opt)
	}
	if len(output.Evidence) != 1 || output.Evidence[0].MemoryID != 1 {
		t.Fatalf("unexpected evidence: %#v", output.Evidence)
	}
}

func TestQueryLongTermMemoryTimeline(t *testing.T) {
	memoryService := &fakeAgentMemoryService{rangeItems: []*model.MemoryItem{{ID: 1}}}
	output, err := (&MemoryAgent{memoryService: memoryService}).QueryLongTermMemory(context.Background(), QueryLongTermMemoryInput{
		From: "2026-05-01",
		To:   "2026-05-31",
		Mode: longTermMemoryModeTimeline,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Items) != 1 || output.Items[0].MemoryID != 1 || memoryService.rangeLimit != 20 {
		t.Fatalf("unexpected output=%#v limit=%d", output, memoryService.rangeLimit)
	}
}

func TestQueryLongTermMemoryTimelineWithoutDatesUsesLastSevenDays(t *testing.T) {
	memoryService := &fakeAgentMemoryService{rangeItems: []*model.MemoryItem{{ID: 1}}}
	before := time.Now()
	output, err := (&MemoryAgent{memoryService: memoryService}).QueryLongTermMemory(context.Background(), QueryLongTermMemoryInput{
		Mode: longTermMemoryModeTimeline,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Items) != 1 || memoryService.rangeLimit != 20 {
		t.Fatalf("unexpected output=%#v limit=%d", output, memoryService.rangeLimit)
	}
	if memoryService.start.Before(before.AddDate(0, 0, -7).Add(-time.Second)) || memoryService.end.Before(before) {
		t.Fatalf("unexpected default range: %s - %s", memoryService.start, memoryService.end)
	}
}

func TestQueryLongTermMemoryAggregate(t *testing.T) {
	memoryService := &fakeAgentMemoryService{rangeItems: []*model.MemoryItem{{ID: 1, Tags: `["Eino"]`, Mood: "专注", Summary: "整理工具"}}}
	output, err := (&MemoryAgent{memoryService: memoryService}).QueryLongTermMemory(context.Background(), QueryLongTermMemoryInput{
		From: "2026-05-01",
		To:   "2026-05-31",
		Mode: longTermMemoryModeAggregate,
	})
	if err != nil {
		t.Fatal(err)
	}
	if output.Aggregation == nil || output.Aggregation.Count != 1 || memoryService.rangeLimit != 20 {
		t.Fatalf("unexpected output=%#v limit=%d", output, memoryService.rangeLimit)
	}
}

func TestQueryLongTermMemoryDefaultsAndCapsLimit(t *testing.T) {
	memoryService := &fakeAgentMemoryService{}
	agent := &MemoryAgent{memoryService: memoryService}

	if _, err := agent.QueryLongTermMemory(context.Background(), QueryLongTermMemoryInput{
		From: "2026-05-01",
		To:   "2026-05-31",
	}); err != nil {
		t.Fatal(err)
	}
	if memoryService.rangeLimit != 20 {
		t.Fatalf("default limit = %d, want 20", memoryService.rangeLimit)
	}

	if _, err := agent.QueryLongTermMemory(context.Background(), QueryLongTermMemoryInput{
		From:  "2026-05-01",
		To:    "2026-05-31",
		Limit: 500,
	}); err != nil {
		t.Fatal(err)
	}
	if memoryService.rangeLimit != 100 {
		t.Fatalf("maximum limit = %d, want 100", memoryService.rangeLimit)
	}
}

func TestQueryLongTermMemoryDoesNotReturnContentText(t *testing.T) {
	memoryService := &fakeAgentMemoryService{rangeItems: []*model.MemoryItem{{
		ID:          1,
		ContentText: strings.Repeat("content", 100),
		Summary:     strings.Repeat("summary", 100),
	}}}
	output, err := (&MemoryAgent{memoryService: memoryService}).QueryLongTermMemory(context.Background(), QueryLongTermMemoryInput{
		From: "2026-05-01",
		To:   "2026-05-31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Items) != 1 || len([]rune(output.Items[0].Summary)) > maxMemorySummaryLength+3 {
		t.Fatalf("unexpected compact item: %#v", output.Items)
	}
}

func TestQueryLongTermMemoryRejectsInvalidDates(t *testing.T) {
	agent := &MemoryAgent{memoryService: &fakeAgentMemoryService{}}
	for _, input := range []QueryLongTermMemoryInput{
		{From: "bad", To: "2026-05-31", Mode: longTermMemoryModeTimeline},
		{From: "2026-06-01", To: "2026-05-31", Mode: longTermMemoryModeTimeline},
	} {
		if _, err := agent.QueryLongTermMemory(context.Background(), input); err == nil {
			t.Fatalf("expected error for %#v", input)
		}
	}
}

func TestGetMemoryDetail(t *testing.T) {
	memoryService := &fakeAgentMemoryService{detail: &model.MemoryItem{ID: 7, Summary: "detail"}}
	output, err := (&MemoryAgent{memoryService: memoryService}).GetMemoryDetail(context.Background(), GetMemoryDetailInput{MemoryID: 7})
	if err != nil {
		t.Fatal(err)
	}
	if output.ID != 7 || output.Summary != "detail" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func TestEinoMemoryToolInvokableRun(t *testing.T) {
	tool := &EinoMemoryTool{
		name: ToolListRecent,
		run: func(_ context.Context, args map[string]any) (any, error) {
			return map[string]any{"limit": args["limit"]}, nil
		},
	}

	got, err := tool.InvokableRun(context.Background(), `{"limit":5}`)
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"limit":5}` {
		t.Fatalf("InvokableRun() = %s", got)
	}

	if _, err := tool.InvokableRun(context.Background(), `{`); err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestEinoMemoryToolDebugTrace(t *testing.T) {
	collector := NewTraceCollector("test")
	ctx := ContextWithTraceCollector(context.Background(), collector)
	tool := &EinoMemoryTool{
		name: ToolListRecent,
		run: func(_ context.Context, args map[string]any) (any, error) {
			return map[string]any{"ok": true}, nil
		},
	}

	if _, err := tool.InvokableRun(ctx, `{"limit":5}`); err != nil {
		t.Fatal(err)
	}
	steps := collector.Trace().Steps
	if len(steps) != 2 || steps[0].Event != "tool_start" || steps[1].Event != "tool_end" {
		t.Fatalf("unexpected trace: %#v", steps)
	}
}

func TestEinoMemoryToolDebugTraceError(t *testing.T) {
	collector := NewTraceCollector("test")
	ctx := ContextWithTraceCollector(context.Background(), collector)
	tool := &EinoMemoryTool{
		name: ToolListRecent,
		run: func(_ context.Context, _ map[string]any) (any, error) {
			return nil, errors.New("tool failed")
		},
	}

	if _, err := tool.InvokableRun(ctx, `{}`); err == nil {
		t.Fatal("expected tool error")
	}
	steps := collector.Trace().Steps
	if len(steps) != 2 || steps[1].Event != "tool_error" || !strings.Contains(steps[1].Error, "tool failed") {
		t.Fatalf("unexpected trace: %#v", steps)
	}
}
