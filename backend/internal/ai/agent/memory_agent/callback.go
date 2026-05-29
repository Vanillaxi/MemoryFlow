package memory_agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	einoagent "github.com/cloudwego/eino/flow/agent"
)

const reactTraceMode = "eino_react"

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

func (c *TraceCollector) EinoAgentOption() einoagent.AgentOption {
	return einoagent.WithComposeOptions(compose.WithCallbacks(c.EinoHandler()))
}

func (c *TraceCollector) EinoHandler() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			c.Start(traceNode(info), input)
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			c.End(traceNode(info), output)
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			c.Error(traceNode(info), err)
			return ctx
		}).
		Build()
}

func (c *TraceCollector) append(step TraceStep) {
	c.mu.Lock()
	c.steps = append(c.steps, step)
	c.mu.Unlock()
}

func traceNode(info *callbacks.RunInfo) string {
	if info == nil {
		return "unknown"
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
	return fmt.Sprintf("%#v", info)
}

func traceTime() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
