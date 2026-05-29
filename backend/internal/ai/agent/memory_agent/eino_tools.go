package memory_agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type EinoMemoryTool struct {
	name       ToolName
	desc       string
	parameters map[string]*schema.ParameterInfo
	run        func(ctx context.Context, args map[string]any) (any, error)
}

func (t *EinoMemoryTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        string(t.name),
		Desc:        t.desc,
		ParamsOneOf: schema.NewParamsOneOfByParams(t.parameters),
	}, nil
}

func (t *EinoMemoryTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("invalid tool arguments json: %w", err)
	}

	if t.run == nil {
		return "", fmt.Errorf("tool %s has no runner", t.name)
	}

	result, err := t.run(ctx, args)
	if err != nil {
		return "", err
	}

	bytes, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (a *MemoryAgent) BaseTools() []tool.BaseTool {
	return []tool.BaseTool{
		a.newAskMemoryEinoTool(),
		a.newSearchMemoryEinoTool(),
		a.newListRecentEinoTool(),
		a.newGetTimelineEinoTool(),
	}
}

func (a *MemoryAgent) newAskMemoryEinoTool() *EinoMemoryTool {
	return &EinoMemoryTool{
		name: ToolAskMemory,
		desc: "基于长期记忆进行 RAG 问答，适合回答用户关于过去经历、项目进展、生活片段的问题。",
		parameters: map[string]*schema.ParameterInfo{
			"question": {
				Type:     schema.String,
				Desc:     "用户问题",
				Required: true,
			},
			"top_k": {
				Type:     schema.Integer,
				Desc:     "召回数量，默认 20",
				Required: false,
			},
			"type": {
				Type:     schema.String,
				Desc:     "记忆类型，可选：text/image/mixed",
				Required: false,
			},
			"start": {
				Type:     schema.String,
				Desc:     "开始日期，格式 YYYY-MM-DD，可选",
				Required: false,
			},
			"end": {
				Type:     schema.String,
				Desc:     "结束日期，格式 YYYY-MM-DD，可选",
				Required: false,
			},
		},
		run: func(ctx context.Context, args map[string]any) (any, error) {
			return a.callAskMemory(ctx, args)
		},
	}
}

func (a *MemoryAgent) newSearchMemoryEinoTool() *EinoMemoryTool {
	return &EinoMemoryTool{
		name: ToolSearchMemory,
		desc: "搜索相关记忆，返回原始召回结果，适合查找、定位、调试记忆。",
		parameters: map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "搜索关键词或问题",
				Required: true,
			},
			"top_k": {
				Type:     schema.Integer,
				Desc:     "召回数量，默认 5",
				Required: false,
			},
			"type": {
				Type:     schema.String,
				Desc:     "记忆类型，可选：text/image/mixed",
				Required: false,
			},
			"start": {
				Type:     schema.String,
				Desc:     "开始日期，格式 YYYY-MM-DD，可选",
				Required: false,
			},
			"end": {
				Type:     schema.String,
				Desc:     "结束日期，格式 YYYY-MM-DD，可选",
				Required: false,
			},
		},
		run: func(ctx context.Context, args map[string]any) (any, error) {
			return a.callSearchMemory(ctx, args)
		},
	}
}

func (a *MemoryAgent) newListRecentEinoTool() *EinoMemoryTool {
	return &EinoMemoryTool{
		name: ToolListRecent,
		desc: "查询最近记忆，适合回答“最近我记录了什么”“最新记忆有哪些”。",
		parameters: map[string]*schema.ParameterInfo{
			"limit": {
				Type:     schema.Integer,
				Desc:     "返回数量，默认 10",
				Required: false,
			},
		},
		run: func(ctx context.Context, args map[string]any) (any, error) {
			return a.callListRecent(ctx, args)
		},
	}
}

func (a *MemoryAgent) newGetTimelineEinoTool() *EinoMemoryTool {
	return &EinoMemoryTool{
		name: ToolGetTimeline,
		desc: "按时间范围查询记忆时间线，适合回答“某段时间我做了什么”。",
		parameters: map[string]*schema.ParameterInfo{
			"start": {
				Type:     schema.String,
				Desc:     "开始日期，格式 YYYY-MM-DD",
				Required: true,
			},
			"end": {
				Type:     schema.String,
				Desc:     "结束日期，格式 YYYY-MM-DD",
				Required: true,
			},
		},
		run: func(ctx context.Context, args map[string]any) (any, error) {
			return a.callGetTimeline(ctx, args)
		},
	}
}

func (a *MemoryAgent) DebugListEinoTools(ctx context.Context) ([]*schema.ToolInfo, error) {
	tools := a.BaseTools()

	infos := make([]*schema.ToolInfo, 0, len(tools))
	for _, t := range tools {
		info, err := t.Info(ctx)
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}

	return infos, nil
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
