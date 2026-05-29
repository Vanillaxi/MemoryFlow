package memory_agent

import (
	"sync"
	"time"
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

func (c *TraceCollector) append(step TraceStep) {
	c.mu.Lock()
	c.steps = append(c.steps, step)
	c.mu.Unlock()
}

func traceTime() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
