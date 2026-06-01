package runtime

import (
	"context"
	"errors"

	"memoryflow/internal/ai/tools"
)

const maxToolResultLength = 6000

type ToolExecutor struct {
	registry *tools.ToolRegistry
}

func NewToolExecutor(registry *tools.ToolRegistry) *ToolExecutor {
	return &ToolExecutor{registry: registry}
}

func (e *ToolExecutor) Execute(ctx context.Context, calls []ToolCall) ([]ToolCallLog, []string, error) {
	if e == nil || e.registry == nil {
		return nil, nil, errors.New("agent runtime tool registry is not initialized")
	}

	logs := make([]ToolCallLog, 0, len(calls))
	usedTools := make([]string, 0, len(calls))
	for _, call := range calls {
		usedTools = append(usedTools, call.Name)
		currentTool, ok := e.registry.Get(call.Name)
		if !ok {
			logs = append(logs, ToolCallLog{Name: call.Name, Error: "tool is not registered"})
			continue
		}

		result, err := currentTool.Call(ctx, call.Args)
		if err != nil {
			logs = append(logs, ToolCallLog{Name: call.Name, Error: err.Error()})
			continue
		}
		logs = append(logs, ToolCallLog{Name: call.Name, Result: truncate(result, maxToolResultLength)})
	}
	return logs, usedTools, nil
}

func truncate(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "...(truncated)"
}
