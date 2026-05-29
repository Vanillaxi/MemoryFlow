package memory_agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const reactTraceMode = "eino_react"

type traceCollectorContextKey struct{}

type TraceCollector struct {
	mu    sync.Mutex
	mode  string
	steps []TraceStep
	err   string
}

func NewTraceCollector(mode string) *TraceCollector {
	return &TraceCollector{mode: mode}
}

func (c *TraceCollector) Start(node string, input any) {
	if c == nil {
		return
	}

	c.append(TraceStep{
		Node:      node,
		Event:     "start",
		Input:     input,
		StartedAt: traceTime(),
	})
}

func (c *TraceCollector) End(node string, output any) {
	if c == nil {
		return
	}

	c.append(TraceStep{
		Node:    node,
		Event:   "end",
		Output:  output,
		EndedAt: traceTime(),
	})
}

func (c *TraceCollector) Error(node string, err error) {
	if c == nil || err == nil {
		return
	}

	errText := err.Error()
	c.mu.Lock()
	c.err = errText
	c.steps = append(c.steps, TraceStep{
		Node:    node,
		Event:   "error",
		Error:   errText,
		EndedAt: traceTime(),
	})
	c.mu.Unlock()
}

func (c *TraceCollector) Event(node string, event string, input any, output any, err error) {
	if c == nil {
		return
	}

	step := TraceStep{
		Node:  node,
		Event: event,
	}
	if input != nil {
		step.Input = input
	}
	if output != nil {
		step.Output = output
	}
	if err != nil {
		step.Error = err.Error()
		step.EndedAt = traceTime()
	} else if strings.HasSuffix(event, "_start") {
		step.StartedAt = traceTime()
	} else {
		step.EndedAt = traceTime()
	}

	c.mu.Lock()
	if err != nil {
		c.err = err.Error()
	}
	c.steps = append(c.steps, step)
	c.mu.Unlock()
}

func (c *TraceCollector) Trace() *AgentTrace {
	if c == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	steps := make([]TraceStep, len(c.steps))
	copy(steps, c.steps)
	return &AgentTrace{
		Mode:  c.mode,
		Steps: steps,
		Error: c.err,
	}
}

func (c *TraceCollector) append(step TraceStep) {
	c.mu.Lock()
	c.steps = append(c.steps, step)
	c.mu.Unlock()
}

func traceTime() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func ContextWithTraceCollector(ctx context.Context, collector *TraceCollector) context.Context {
	if collector == nil {
		return ctx
	}
	return context.WithValue(ctx, traceCollectorContextKey{}, collector)
}

func TraceCollectorFromContext(ctx context.Context) *TraceCollector {
	collector, _ := ctx.Value(traceCollectorContextKey{}).(*TraceCollector)
	return collector
}

type EinoTraceHandler struct {
	collector *TraceCollector
}

func NewEinoTraceHandler(collector *TraceCollector) *EinoTraceHandler {
	return &EinoTraceHandler{collector: collector}
}

func (h *EinoTraceHandler) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	h.record(info, "start", input, nil, nil)
	return ctx
}

func (h *EinoTraceHandler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	h.record(info, "end", nil, output, nil)
	return ctx
}

func (h *EinoTraceHandler) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	h.record(info, "error", nil, nil, err)
	return ctx
}

func (h *EinoTraceHandler) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	if input != nil {
		input.Close()
	}
	h.record(info, "start", map[string]any{"stream": true}, nil, nil)
	return ctx
}

func (h *EinoTraceHandler) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	if output != nil {
		output.Close()
	}
	h.record(info, "end", nil, map[string]any{"stream": true}, nil)
	return ctx
}

func (h *EinoTraceHandler) Needed(ctx context.Context, info *callbacks.RunInfo, timing callbacks.CallbackTiming) bool {
	return h != nil && h.collector != nil
}

func (h *EinoTraceHandler) record(info *callbacks.RunInfo, phase string, input callbacks.CallbackInput, output callbacks.CallbackOutput, err error) {
	if h == nil || h.collector == nil {
		return
	}

	kind := traceKind(info)
	event := kind + "_" + phase
	node := traceNodeName(info, kind)
	h.collector.Event(node, event, summarizeCallbackInput(info, input), summarizeCallbackOutput(info, output), err)
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

	return map[string]any{
		"message_count": len(input.Messages),
		"messages":      summarizeMessages(input.Messages, 500),
		"tools":         tools,
	}
}

func summarizeModelOutput(output *einomodel.CallbackOutput) any {
	if output == nil || output.Message == nil {
		return nil
	}

	return map[string]any{
		"summary":    summarizeTraceText(output.Message.Content, 500),
		"tool_calls": summarizeToolCalls(output.Message.ToolCalls),
	}
}

func summarizeToolInput(input *einotool.CallbackInput) any {
	if input == nil {
		return nil
	}
	return map[string]any{
		"arguments": summarizeTraceText(sanitizeTraceJSON(input.ArgumentsInJSON), 500),
	}
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
		if msg == nil {
			continue
		}
		summaries = append(summaries, map[string]any{
			"role":       msg.Role,
			"summary":    summarizeTraceText(msg.Content, limit),
			"tool_calls": summarizeToolCalls(msg.ToolCalls),
		})
	}
	return summaries
}

func summarizeToolCalls(toolCalls []schema.ToolCall) []map[string]any {
	summaries := make([]map[string]any, 0, len(toolCalls))
	for _, call := range toolCalls {
		summaries = append(summaries, map[string]any{
			"name":      call.Function.Name,
			"arguments": summarizeTraceText(sanitizeTraceJSON(call.Function.Arguments), 500),
		})
	}
	return summaries
}

func summarizeTraceValue(v any, limit int) any {
	if v == nil {
		return nil
	}

	bytes, err := json.Marshal(v)
	if err != nil {
		return map[string]any{"summary": summarizeTraceText(fmt.Sprintf("%v", v), limit)}
	}
	var decoded any
	if err := json.Unmarshal(bytes, &decoded); err == nil {
		bytes, err = json.Marshal(sanitizeTraceValue(decoded))
		if err != nil {
			return map[string]any{"summary": summarizeTraceText(fmt.Sprintf("%v", v), limit)}
		}
	}
	return map[string]any{"summary": summarizeTraceText(string(bytes), limit)}
}

func sanitizeTraceJSON(text string) string {
	var v any
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		return text
	}
	bytes, err := json.Marshal(sanitizeTraceValue(v))
	if err != nil {
		return text
	}
	return string(bytes)
}

func sanitizeTraceValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, value := range t {
			if isSensitiveTraceKey(k) {
				out[k] = "[redacted]"
				continue
			}
			out[k] = sanitizeTraceValue(value)
		}
		return out
	case []any:
		out := make([]any, 0, len(t))
		for _, value := range t {
			out = append(out, sanitizeTraceValue(value))
		}
		return out
	default:
		return t
	}
}

func isSensitiveTraceKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	return strings.Contains(normalized, "api_key") ||
		strings.Contains(normalized, "authorization") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret")
}
