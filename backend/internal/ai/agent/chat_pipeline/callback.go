package chat_pipeline

import (
	"context"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type TraceHandler struct {
	collector *TraceCollector
}

func NewTraceHandler(collector *TraceCollector) *TraceHandler {
	return &TraceHandler{collector: collector}
}

func (h *TraceHandler) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	h.record(info, "start", input, nil, nil)
	return ctx
}

func (h *TraceHandler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	h.record(info, "end", nil, output, nil)
	return ctx
}

func (h *TraceHandler) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	h.record(info, "error", nil, nil, err)
	return ctx
}

func (h *TraceHandler) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	if input != nil {
		input.Close()
	}
	h.record(info, "start", map[string]any{"stream": true}, nil, nil)
	return ctx
}

func (h *TraceHandler) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	if output != nil {
		output.Close()
	}
	h.record(info, "end", nil, map[string]any{"stream": true}, nil)
	return ctx
}

func (h *TraceHandler) Needed(context.Context, *callbacks.RunInfo, callbacks.CallbackTiming) bool {
	return h != nil && h.collector != nil
}

func (h *TraceHandler) record(info *callbacks.RunInfo, phase string, input callbacks.CallbackInput, output callbacks.CallbackOutput, err error) {
	if h == nil || h.collector == nil {
		return
	}
	kind := traceKind(info)
	h.collector.Event(traceNodeName(info, kind), kind+"_"+phase, summarizeCallbackInput(info, input), summarizeCallbackOutput(info, output), err)
}

func traceKind(info *callbacks.RunInfo) string {
	if info == nil {
		return "component"
	}
	switch info.Component {
	case compose.ComponentOfGraph, compose.ComponentOfChain, compose.ComponentOfWorkflow:
		return "graph"
	case components.ComponentOfChatModel:
		return "model"
	case components.ComponentOfTool:
		return "tool"
	default:
		return "component"
	}
}

func traceNodeName(info *callbacks.RunInfo, kind string) string {
	if info == nil {
		return kind
	}
	if info.Name != "" {
		return info.Name
	}
	if info.Component != "" {
		return string(info.Component)
	}
	if info.Type != "" {
		return info.Type
	}
	return kind
}

func summarizeCallbackInput(info *callbacks.RunInfo, input callbacks.CallbackInput) any {
	if input == nil {
		return nil
	}
	if info != nil {
		switch info.Component {
		case components.ComponentOfChatModel:
			return summarizeModelInput(einomodel.ConvCallbackInput(input))
		case components.ComponentOfTool:
			return summarizeToolInput(einotool.ConvCallbackInput(input))
		}
	}
	return summarizeTraceValue(input, 500)
}

func summarizeCallbackOutput(info *callbacks.RunInfo, output callbacks.CallbackOutput) any {
	if output == nil {
		return nil
	}
	if info != nil {
		switch info.Component {
		case components.ComponentOfChatModel:
			return summarizeModelOutput(einomodel.ConvCallbackOutput(output))
		case components.ComponentOfTool:
			return summarizeToolOutput(einotool.ConvCallbackOutput(output))
		}
	}
	return summarizeTraceValue(output, 500)
}

func summarizeModelInput(input *einomodel.CallbackInput) any {
	if input == nil {
		return nil
	}
	tools := make([]string, 0, len(input.Tools))
	for _, toolInfo := range input.Tools {
		if toolInfo != nil {
			tools = append(tools, toolInfo.Name)
		}
	}
	return map[string]any{"message_count": len(input.Messages), "messages": summarizeMessages(input.Messages, 500), "tools": tools}
}

func summarizeModelOutput(output *einomodel.CallbackOutput) any {
	if output == nil || output.Message == nil {
		return nil
	}
	return map[string]any{"summary": summarizeTraceText(output.Message.Content, 500), "tool_calls": summarizeToolCalls(output.Message.ToolCalls)}
}

func summarizeToolInput(input *einotool.CallbackInput) any {
	if input == nil {
		return nil
	}
	return map[string]any{"arguments": summarizeTraceText(sanitizeTraceJSON(input.ArgumentsInJSON), 500)}
}

func summarizeToolOutput(output *einotool.CallbackOutput) any {
	if output == nil {
		return nil
	}
	if output.Response != "" {
		return map[string]any{"summary": summarizeTraceText(sanitizeTraceJSON(output.Response), 500)}
	}
	return summarizeTraceValue(output.ToolOutput, 500)
}

func summarizeMessages(messages []*schema.Message, limit int) []map[string]any {
	summaries := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		if msg != nil {
			summaries = append(summaries, map[string]any{"role": msg.Role, "summary": summarizeTraceText(msg.Content, limit), "tool_calls": summarizeToolCalls(msg.ToolCalls)})
		}
	}
	return summaries
}

func summarizeToolCalls(toolCalls []schema.ToolCall) []map[string]any {
	summaries := make([]map[string]any, 0, len(toolCalls))
	for _, call := range toolCalls {
		summaries = append(summaries, map[string]any{"name": call.Function.Name, "arguments": summarizeTraceText(sanitizeTraceJSON(call.Function.Arguments), 500)})
	}
	return summaries
}
