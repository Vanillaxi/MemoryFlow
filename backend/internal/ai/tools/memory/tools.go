package memory

import (
	"context"
	"fmt"

	"memoryflow/internal/ai/tools"

	"github.com/cloudwego/eino/schema"
)

type QueryLongTermMemoryTool struct {
	*tools.RegisteredTool
}

type GetMemoryDetailTool struct {
	*tools.RegisteredTool
}

type AggregateMemoryTool struct {
	*tools.RegisteredTool
}

func NewQueryLongTermMemoryTool(memoryRetriever MemoryRetriever, memoryService MemoryService, trace tools.TraceEvent) *QueryLongTermMemoryTool {
	return &QueryLongTermMemoryTool{RegisteredTool: tools.NewRegisteredTool(
		ToolQueryLongTermMemory,
		"自然语言查询用户长期记忆库。semantic 用于语义检索相关证据；timeline 用于按时间查看记忆列表；aggregate 用于按时间聚合标签、情绪和重点候选。",
		map[string]*schema.ParameterInfo{
			"query": {Type: schema.String, Desc: "自然语言查询。semantic 模式必填。"},
			"from":  {Type: schema.String, Desc: "开始日期，格式 YYYY-MM-DD，可选。"},
			"to":    {Type: schema.String, Desc: "结束日期，格式 YYYY-MM-DD，可选。"},
			"mode":  {Type: schema.String, Desc: "查询模式：semantic/timeline/aggregate。可选；未填写时根据 query 和日期范围自动选择。"},
			"limit": {Type: schema.Integer, Desc: "返回数量，可选。"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			return QueryLongTermMemory(ctx, memoryRetriever, memoryService, QueryLongTermMemoryInput{
				Query: tools.StringArg(args, "query"),
				From:  tools.StringArg(args, "from"),
				To:    tools.StringArg(args, "to"),
				Mode:  tools.StringArg(args, "mode"),
				Limit: tools.IntArg(args, "limit"),
			})
		},
		trace,
	)}
}

func NewGetMemoryDetailTool(memoryService MemoryService, trace tools.TraceEvent) *GetMemoryDetailTool {
	return &GetMemoryDetailTool{RegisteredTool: tools.NewRegisteredTool(
		ToolGetMemoryDetail,
		"根据 memory_id 获取某条长期记忆的完整详情。先通过 query_long_term_memory 找到 memory_id，再按需调用。",
		map[string]*schema.ParameterInfo{
			"memory_id": {Type: schema.Integer, Desc: "记忆 ID。", Required: true},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			memoryID := tools.IntArg(args, "memory_id")
			if memoryID <= 0 {
				return nil, fmt.Errorf("memory_id is required")
			}
			return GetMemoryDetail(ctx, memoryService, GetMemoryDetailInput{MemoryID: uint(memoryID)})
		},
		trace,
	)}
}

func NewAggregateMemoryTool(memoryService MemoryService, trace tools.TraceEvent) *AggregateMemoryTool {
	return &AggregateMemoryTool{RegisteredTool: tools.NewRegisteredTool(
		ToolAggregateMemory,
		"聚合一段时间内的长期记忆，返回数量、高频标签、情绪、重点和按时间排序的简要列表。适合总结、回顾和交接。",
		map[string]*schema.ParameterInfo{
			"from":  {Type: schema.String, Desc: "开始日期，格式 YYYY-MM-DD，可选。"},
			"to":    {Type: schema.String, Desc: "结束日期，格式 YYYY-MM-DD，可选。"},
			"limit": {Type: schema.Integer, Desc: "返回数量，可选。"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			return AggregateMemory(ctx, memoryService, AggregateMemoryInput{
				From:  tools.StringArg(args, "from"),
				To:    tools.StringArg(args, "to"),
				Limit: tools.IntArg(args, "limit"),
			})
		},
		trace,
	)}
}
