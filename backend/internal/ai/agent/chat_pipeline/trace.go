package chat_pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
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
	if c != nil {
		c.append(TraceStep{Node: node, Event: "start", Input: input, StartedAt: traceTime()})
	}
}

func (c *TraceCollector) End(node string, output any) {
	if c != nil {
		c.append(TraceStep{Node: node, Event: "end", Output: output, EndedAt: traceTime()})
	}
}

func (c *TraceCollector) Error(node string, err error) {
	if c == nil || err == nil {
		return
	}
	errText := err.Error()
	c.mu.Lock()
	c.err = errText
	c.steps = append(c.steps, TraceStep{Node: node, Event: "error", Error: errText, EndedAt: traceTime()})
	c.mu.Unlock()
}

func (c *TraceCollector) Event(node string, event string, input any, output any, err error) {
	if c == nil {
		return
	}
	step := TraceStep{Node: node, Event: event, Input: input, Output: output}
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
	return &AgentTrace{Mode: c.mode, Steps: steps, Error: c.err}
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

func summarizeTraceText(text string, limit int) string {
	runes := []rune(text)
	if limit <= 0 || len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
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
	var value any
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		return text
	}
	bytes, err := json.Marshal(sanitizeTraceValue(value))
	if err != nil {
		return text
	}
	return string(bytes)
}

func sanitizeTraceValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if isSensitiveTraceKey(key) {
				out[key] = "[redacted]"
				continue
			}
			out[key] = sanitizeTraceValue(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, sanitizeTraceValue(item))
		}
		return out
	default:
		return typed
	}
}

func isSensitiveTraceKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	return strings.Contains(normalized, "api_key") ||
		strings.Contains(normalized, "authorization") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret")
}
