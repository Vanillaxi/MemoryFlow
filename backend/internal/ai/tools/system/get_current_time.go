package system

import (
	"context"
	"time"

	"memoryflow/internal/ai/tools"
)

const ToolGetCurrentTime = "get_current_time"

type GetCurrentTimeTool struct {
	*tools.RegisteredTool
}

type CurrentTimeOutput struct {
	Now      time.Time `json:"now"`
	Date     string    `json:"date"`
	TimeZone string    `json:"timezone"`
}

func NewGetCurrentTimeTool(trace ...tools.TraceEvent) *GetCurrentTimeTool {
	return &GetCurrentTimeTool{RegisteredTool: tools.NewRegisteredTool(
		ToolGetCurrentTime,
		"获取当前时间、日期和时区。解析今天、昨天、最近一周、本月等相对时间前先调用此工具。",
		nil,
		func(context.Context, map[string]any) (any, error) {
			return GetCurrentTime(), nil
		},
		firstTrace(trace),
	)}
}

func GetCurrentTime() CurrentTimeOutput {
	now := time.Now()
	zone, _ := now.Zone()
	return CurrentTimeOutput{
		Now:      now,
		Date:     now.Format("2006-01-02"),
		TimeZone: zone,
	}
}

func firstTrace(traces []tools.TraceEvent) tools.TraceEvent {
	if len(traces) == 0 {
		return nil
	}
	return traces[0]
}
