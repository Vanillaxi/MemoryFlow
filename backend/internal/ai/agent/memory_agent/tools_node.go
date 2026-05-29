package memory_agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ToolName string

const (
	ToolAskMemory    ToolName = "ask_memory"
	ToolSearchMemory ToolName = "search_memory"
	ToolListRecent   ToolName = "list_recent"
	ToolGetTimeline  ToolName = "get_timeline"
)

type ToolDefinition struct {
	Name        ToolName `json:"name"`
	Description string   `json:"description"`
	InputSchema string   `json:"input_schema"`
}

type ToolCall struct {
	Name      ToolName       `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type ToolResult struct {
	Name   ToolName `json:"name"`
	Result any      `json:"result"`
}

func (a *MemoryAgent) ListTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        ToolAskMemory,
			Description: "基于长期记忆进行 RAG 问答，适合回答用户关于过去经历、项目进展、生活片段的问题。",
			InputSchema: `{
  "question": "用户问题",
  "top_k": 20,
  "type": "text/image/mixed，可选",
  "start": "YYYY-MM-DD，可选",
  "end": "YYYY-MM-DD，可选"
}`,
		},
		{
			Name:        ToolSearchMemory,
			Description: "只搜索相关记忆，不生成最终回答，适合调试召回结果或查看原始记忆。",
			InputSchema: `{
  "query": "搜索关键词",
  "top_k": 5,
  "type": "text/image/mixed，可选",
  "start": "YYYY-MM-DD，可选",
  "end": "YYYY-MM-DD，可选"
}`,
		},
		{
			Name:        ToolListRecent,
			Description: "查询最近的记忆，适合回答“最近我记录了什么”“最新记忆有哪些”。",
			InputSchema: `{
  "limit": 10
}`,
		},
		{
			Name:        ToolGetTimeline,
			Description: "按时间范围查询记忆时间线，适合回答“某段时间我做了什么”。",
			InputSchema: `{
  "start": "YYYY-MM-DD",
  "end": "YYYY-MM-DD"
}`,
		},
	}
}

func (a *MemoryAgent) CallTool(ctx context.Context, call ToolCall) (*ToolResult, error) {
	switch call.Name {
	case ToolAskMemory:
		result, err := a.callAskMemory(ctx, call.Arguments)
		if err != nil {
			return nil, err
		}
		return &ToolResult{Name: ToolAskMemory, Result: result}, nil

	case ToolSearchMemory:
		result, err := a.callSearchMemory(ctx, call.Arguments)
		if err != nil {
			return nil, err
		}
		return &ToolResult{Name: ToolSearchMemory, Result: result}, nil

	case ToolListRecent:
		result, err := a.callListRecent(ctx, call.Arguments)
		if err != nil {
			return nil, err
		}
		return &ToolResult{Name: ToolListRecent, Result: result}, nil

	case ToolGetTimeline:
		result, err := a.callGetTimeline(ctx, call.Arguments)
		if err != nil {
			return nil, err
		}
		return &ToolResult{Name: ToolGetTimeline, Result: result}, nil

	default:
		return nil, fmt.Errorf("unknown tool: %s", call.Name)
	}
}

func (a *MemoryAgent) callAskMemory(ctx context.Context, args map[string]any) (any, error) {
	question := getStringArg(args, "question")
	if strings.TrimSpace(question) == "" {
		return nil, fmt.Errorf("question is required")
	}

	startTime, err := parseOptionalDateArg(args, "start", false)
	if err != nil {
		return nil, err
	}

	endTime, err := parseOptionalDateArg(args, "end", true)
	if err != nil {
		return nil, err
	}

	return a.AskMemory(ctx, AskMemoryInput{
		Question:  question,
		TopK:      getIntArg(args, "top_k", 20),
		Type:      getStringArg(args, "type"),
		StartTime: startTime,
		EndTime:   endTime,
	})
}

func (a *MemoryAgent) callSearchMemory(ctx context.Context, args map[string]any) (any, error) {
	query := getStringArg(args, "query")
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query is required")
	}

	startTime, err := parseOptionalDateArg(args, "start", false)
	if err != nil {
		return nil, err
	}

	endTime, err := parseOptionalDateArg(args, "end", true)
	if err != nil {
		return nil, err
	}

	return a.SearchMemory(ctx, SearchMemoryInput{
		Query:     query,
		TopK:      getIntArg(args, "top_k", 5),
		Type:      getStringArg(args, "type"),
		StartTime: startTime,
		EndTime:   endTime,
	})
}

func (a *MemoryAgent) callListRecent(ctx context.Context, args map[string]any) (any, error) {
	return a.ListRecent(ctx, RecentMemoryInput{
		Limit: getIntArg(args, "limit", 10),
	})
}

func (a *MemoryAgent) callGetTimeline(ctx context.Context, args map[string]any) (any, error) {
	startStr := getStringArg(args, "start")
	endStr := getStringArg(args, "end")

	if strings.TrimSpace(startStr) == "" || strings.TrimSpace(endStr) == "" {
		return nil, fmt.Errorf("start and end are required")
	}

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return nil, fmt.Errorf("invalid start format, expected YYYY-MM-DD")
	}

	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return nil, fmt.Errorf("invalid end format, expected YYYY-MM-DD")
	}
	end = end.Add(24*time.Hour - time.Second)

	return a.GetTimeline(ctx, TimelineInput{
		Start: start,
		End:   end,
	})
}

func getStringArg(args map[string]any, key string) string {
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val)
	default:
		return strings.TrimSpace(fmt.Sprint(val))
	}
}

func getIntArg(args map[string]any, key string, defaultValue int) int {
	v, ok := args[key]
	if !ok || v == nil {
		return defaultValue
	}

	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case json.Number:
		i, err := val.Int64()
		if err == nil {
			return int(i)
		}
		return defaultValue
	default:
		return defaultValue
	}
}

func parseOptionalDateArg(args map[string]any, key string, includeWholeDay bool) (*time.Time, error) {
	dateStr := getStringArg(args, key)
	if dateStr == "" {
		return nil, nil
	}

	parsed, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid %s format, expected YYYY-MM-DD", key)
	}

	if includeWholeDay {
		parsed = parsed.Add(24*time.Hour - time.Second)
	}

	return &parsed, nil
}
