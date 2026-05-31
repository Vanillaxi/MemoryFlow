package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const (
	ToolGetCurrentTime      = "get_current_time"
	ToolQueryLongTermMemory = "query_long_term_memory"
	ToolGetMemoryDetail     = "get_memory_detail"
)

type TraceEvent func(ctx context.Context, name string, event string, input any, output any, err error)

type RegisteredTool struct {
	name       string
	desc       string
	parameters map[string]*schema.ParameterInfo
	run        func(ctx context.Context, args map[string]any) (any, error)
	trace      TraceEvent
}

func (t *RegisteredTool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        t.name,
		Desc:        t.desc,
		ParamsOneOf: schema.NewParamsOneOfByParams(t.parameters),
	}, nil
}

func (t *RegisteredTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	t.emit(ctx, "tool_start", map[string]any{"arguments": argumentsInJSON}, nil, nil)

	var args map[string]any
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		err = fmt.Errorf("invalid tool arguments json: %w", err)
		t.emit(ctx, "tool_error", nil, nil, err)
		return "", err
	}

	result, err := t.run(ctx, args)
	if err != nil {
		t.emit(ctx, "tool_error", nil, nil, err)
		return "", err
	}

	bytes, err := json.Marshal(result)
	if err != nil {
		t.emit(ctx, "tool_error", nil, nil, err)
		return "", err
	}

	output := string(bytes)
	t.emit(ctx, "tool_end", nil, map[string]any{"summary": output}, nil)
	return output, nil
}

func (t *RegisteredTool) emit(ctx context.Context, event string, input any, output any, err error) {
	if t.trace != nil {
		t.trace(ctx, t.name, event, input, output, err)
	}
}

func NewGetCurrentTimeTool(trace TraceEvent) *RegisteredTool {
	return &RegisteredTool{
		name:       ToolGetCurrentTime,
		desc:       "获取当前时间、日期和时区。解析今天、昨天、最近一周、本月等相对时间前先调用此工具。",
		parameters: map[string]*schema.ParameterInfo{},
		run: func(context.Context, map[string]any) (any, error) {
			return GetCurrentTime(), nil
		},
		trace: trace,
	}
}

func NewQueryLongTermMemoryTool(memoryRetriever MemoryRetriever, memoryService MemoryService, trace TraceEvent) *RegisteredTool {
	return &RegisteredTool{
		name: ToolQueryLongTermMemory,
		desc: "自然语言查询用户长期记忆库。semantic 用于语义检索相关证据；timeline 用于按时间查看记忆列表；aggregate 用于按时间聚合标签、情绪和重点候选。",
		parameters: map[string]*schema.ParameterInfo{
			"query": {Type: schema.String, Desc: "自然语言查询。semantic 模式必填。"},
			"from":  {Type: schema.String, Desc: "开始日期，格式 YYYY-MM-DD，可选。"},
			"to":    {Type: schema.String, Desc: "结束日期，格式 YYYY-MM-DD，可选。"},
			"mode":  {Type: schema.String, Desc: "查询模式：semantic/timeline/aggregate。可选；未填写时根据 query 和日期范围自动选择。"},
			"limit": {Type: schema.Integer, Desc: "返回数量，可选。"},
		},
		run: func(ctx context.Context, args map[string]any) (any, error) {
			return QueryLongTermMemory(ctx, memoryRetriever, memoryService, QueryLongTermMemoryInput{
				Query: stringArg(args, "query"),
				From:  stringArg(args, "from"),
				To:    stringArg(args, "to"),
				Mode:  stringArg(args, "mode"),
				Limit: intArg(args, "limit"),
			})
		},
		trace: trace,
	}
}

func NewGetMemoryDetailTool(memoryService MemoryService, trace TraceEvent) *RegisteredTool {
	return &RegisteredTool{
		name: ToolGetMemoryDetail,
		desc: "根据 memory_id 获取某条长期记忆的完整详情。先通过 query_long_term_memory 找到 memory_id，再按需调用。",
		parameters: map[string]*schema.ParameterInfo{
			"memory_id": {Type: schema.Integer, Desc: "记忆 ID。", Required: true},
		},
		run: func(ctx context.Context, args map[string]any) (any, error) {
			memoryID := intArg(args, "memory_id")
			if memoryID <= 0 {
				return nil, fmt.Errorf("memory_id is required")
			}
			return GetMemoryDetail(ctx, memoryService, GetMemoryDetailInput{MemoryID: uint(memoryID)})
		},
		trace: trace,
	}
}

func stringArg(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return value
}

func intArg(args map[string]any, key string) int {
	switch value := args[key].(type) {
	case int:
		return value
	case float64:
		return int(value)
	default:
		return 0
	}
}
