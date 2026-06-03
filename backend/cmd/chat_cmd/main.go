package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"memoryflow/internal/ai/agent"
	"memoryflow/internal/ai/agent/chat_pipeline"
	"memoryflow/internal/ai/agent/dispatcher"
	"memoryflow/internal/ai/agent/project_pipeline"
	"memoryflow/internal/bootstrap"
)

const defaultQuestion = "我的 MemoryFlow 最近做到哪了？"

type chatCLIOutput struct {
	Answer     string         `json:"answer"`
	Pipeline   string         `json:"pipeline"`
	Project    any            `json:"project,omitempty"`
	UsedTools  []string       `json:"used_tools,omitempty"`
	Evidence   any            `json:"evidence,omitempty"`
	ToolErrors []toolError    `json:"tool_errors,omitempty"`
	Raw        map[string]any `json:"raw,omitempty"`
}

type toolError struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

func main() {
	log.SetOutput(os.Stderr)
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "chat_cmd failed: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	question := flag.String("q", "", "user question")
	projectName := flag.String("project", "", "optional project name, e.g. MemoryFlow")
	projectID := flag.Uint("project-id", 0, "optional project id; higher priority than -project")
	days := flag.Int("days", 0, "recent days; default from config.github.default_days or 7")
	limit := flag.Int("limit", 0, "commit limit; default from config.github.default_limit or 10")
	pipeline := flag.String("pipeline", "auto", "pipeline: auto, chat, project")
	debug := flag.Bool("debug", false, "print debug fields such as used_tools and evidence")
	jsonOutput := flag.Bool("json", false, "print full JSON output")
	flag.Parse()

	message := strings.TrimSpace(*question)
	if message == "" {
		message = defaultQuestion
	}
	if name := strings.TrimSpace(*projectName); name != "" && !strings.Contains(strings.ToLower(message), strings.ToLower(name)) {
		message = name + " " + message
	}

	app, err := bootstrap.NewApp(ctx)
	if err != nil {
		return fmt.Errorf("bootstrap app failed: %w", err)
	}
	defer app.Close(ctx)

	if *days <= 0 {
		*days = app.Config.Github.DefaultDays
		if *days <= 0 {
			*days = 7
		}
	}
	if *limit <= 0 {
		*limit = app.Config.Github.DefaultLimit
		if *limit <= 0 {
			*limit = 10
		}
	}

	var id *uint
	if *projectID > 0 {
		v := uint(*projectID)
		id = &v
	}

	output, err := invoke(ctx, app, message, normalizePipeline(*pipeline), id, *days, *limit, *debug)
	if err != nil {
		return err
	}

	if *jsonOutput {
		return printJSON(output)
	}
	fmt.Println(output.Answer)
	if *debug {
		printDebug(output)
	}
	return nil
}

func invoke(ctx context.Context, app *bootstrap.App, message string, pipeline string, projectID *uint, days int, limit int, debug bool) (*chatCLIOutput, error) {
	switch pipeline {
	case dispatcher.PipelineChat:
		output, err := app.ChatPipeline.Invoke(ctx, chat_pipeline.ChatInput{Message: message, TopK: 20, Debug: debug})
		if err != nil {
			return nil, fmt.Errorf("chat pipeline failed: %w", err)
		}
		return &chatCLIOutput{
			Answer:   strings.TrimSpace(output.Answer),
			Pipeline: dispatcher.PipelineChat,
			Raw: map[string]any{
				"intent": output.Intent,
				"trace":  output.Trace,
			},
		}, nil
	case dispatcher.PipelineProject:
		return invokeAgent(ctx, app, message, dispatcher.PipelineProject, projectID, days, limit)
	case "auto":
		return invokeAgent(ctx, app, message, "", projectID, days, limit)
	default:
		return nil, fmt.Errorf("invalid -pipeline %q, expected auto/chat/project", pipeline)
	}
}

func invokeAgent(ctx context.Context, app *bootstrap.App, message string, pipeline string, projectID *uint, days int, limit int) (*chatCLIOutput, error) {
	output, err := app.Agent.Chat(ctx, agent.ChatInput{
		Message: message, ProjectID: projectID, Days: days, Limit: limit, Pipeline: pipeline,
	})
	if err != nil {
		return nil, fmt.Errorf("agent chat failed: %w", err)
	}
	return &chatCLIOutput{
		Answer:     strings.TrimSpace(output.Answer),
		Pipeline:   output.Pipeline,
		Project:    output.Project,
		UsedTools:  output.UsedTools,
		Evidence:   output.Evidence,
		ToolErrors: collectToolErrors(output.RawToolCalls),
	}, nil
}

func normalizePipeline(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto":
		return "auto"
	case "chat", dispatcher.PipelineChat:
		return dispatcher.PipelineChat
	case "project", dispatcher.PipelineProject:
		return dispatcher.PipelineProject
	default:
		return strings.TrimSpace(value)
	}
}

func collectToolErrors(calls []project_pipeline.ToolCallLog) []toolError {
	items := make([]toolError, 0)
	for _, call := range calls {
		if strings.TrimSpace(call.Error) != "" {
			items = append(items, toolError{Name: call.Name, Error: call.Error})
		}
	}
	return items
}

func printDebug(output *chatCLIOutput) {
	if output.Pipeline != "" {
		fmt.Printf("\n[pipeline] %s\n", output.Pipeline)
	}
	if len(output.UsedTools) > 0 {
		fmt.Printf("[used_tools] %s\n", strings.Join(output.UsedTools, ", "))
	}
	if len(output.ToolErrors) > 0 {
		bytes, _ := json.MarshalIndent(output.ToolErrors, "", "  ")
		fmt.Printf("[tool_errors] %s\n", string(bytes))
	}
}

func printJSON(v any) error {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal output failed: %w", err)
	}
	fmt.Println(string(bytes))
	return nil
}
