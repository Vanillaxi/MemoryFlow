package chat_pipeline

import (
	"context"

	memorytool "memoryflow/internal/ai/tools/memory"
	systemtool "memoryflow/internal/ai/tools/system"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

func (p *Pipeline) BaseTools() []tool.BaseTool {
	return []tool.BaseTool{
		systemtool.NewGetCurrentTimeTool(traceExternalTool),
		memorytool.NewQueryLongTermMemoryTool(p.memoryRetriever, p.memoryService, traceExternalTool),
		memorytool.NewGetMemoryDetailTool(p.memoryService, traceExternalTool),
		memorytool.NewAggregateMemoryTool(p.memoryService, traceExternalTool),
	}
}

func (p *Pipeline) DebugListTools(ctx context.Context) ([]*schema.ToolInfo, error) {
	tools := p.BaseTools()
	infos := make([]*schema.ToolInfo, 0, len(tools))
	for _, currentTool := range tools {
		info, err := currentTool.Info(ctx)
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func traceExternalTool(ctx context.Context, name string, event string, input any, output any, err error) {
	if collector := TraceCollectorFromContext(ctx); collector != nil {
		collector.Event(name, event, sanitizeExternalToolTrace(input), sanitizeExternalToolTrace(output), err)
	}
}

func sanitizeExternalToolTrace(payload any) any {
	values, ok := payload.(map[string]any)
	if !ok {
		return payload
	}

	sanitized := make(map[string]any, len(values))
	for key, value := range values {
		if text, ok := value.(string); ok {
			sanitized[key] = summarizeTraceText(sanitizeTraceJSON(text), 500)
			continue
		}
		sanitized[key] = value
	}
	return sanitized
}
