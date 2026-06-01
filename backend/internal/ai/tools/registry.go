package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type TraceEvent func(ctx context.Context, name string, event string, input any, output any, err error)

type ToolRegistry struct {
	tools map[string]Tool
	order []string
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]Tool)}
}

func (r *ToolRegistry) Register(currentTool Tool) {
	if currentTool == nil || currentTool.Name() == "" {
		return
	}
	if r.tools == nil {
		r.tools = make(map[string]Tool)
	}
	if _, exists := r.tools[currentTool.Name()]; !exists {
		r.order = append(r.order, currentTool.Name())
	}
	r.tools[currentTool.Name()] = currentTool
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	if r == nil {
		return nil, false
	}
	currentTool, ok := r.tools[name]
	return currentTool, ok
}

func (r *ToolRegistry) List() []Tool {
	if r == nil {
		return nil
	}
	items := make([]Tool, 0, len(r.order))
	for _, name := range r.order {
		if currentTool, ok := r.tools[name]; ok {
			items = append(items, currentTool)
		}
	}
	return items
}

type RegisteredTool struct {
	name       string
	desc       string
	parameters map[string]*schema.ParameterInfo
	run        func(ctx context.Context, args map[string]any) (any, error)
	trace      TraceEvent
}

func NewRegisteredTool(
	name string,
	description string,
	parameters map[string]*schema.ParameterInfo,
	run func(ctx context.Context, args map[string]any) (any, error),
	trace TraceEvent,
) *RegisteredTool {
	return &RegisteredTool{
		name:       name,
		desc:       description,
		parameters: parameters,
		run:        run,
		trace:      trace,
	}
}

func (t *RegisteredTool) Name() string {
	return t.name
}

func (t *RegisteredTool) Description() string {
	return t.desc
}

func (t *RegisteredTool) Call(ctx context.Context, args map[string]any) (string, error) {
	t.emit(ctx, "tool_start", args, nil, nil)

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

func (t *RegisteredTool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        t.name,
		Desc:        t.desc,
		ParamsOneOf: schema.NewParamsOneOfByParams(t.parameters),
	}, nil
}

func (t *RegisteredTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		err = fmt.Errorf("invalid tool arguments json: %w", err)
		t.emit(ctx, "tool_error", nil, nil, err)
		return "", err
	}

	return t.Call(ctx, args)
}

func (t *RegisteredTool) emit(ctx context.Context, event string, input any, output any, err error) {
	if t.trace != nil {
		t.trace(ctx, t.name, event, input, output, err)
	}
}

func StringArg(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return value
}

func IntArg(args map[string]any, key string) int {
	switch value := args[key].(type) {
	case int:
		return value
	case float64:
		return int(value)
	default:
		return 0
	}
}
