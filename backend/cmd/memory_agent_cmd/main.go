package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"

	"memoryflow/internal/ai/agent/memory_react_agent"
	"memoryflow/internal/bootstrap"
)

const defaultQuestion = "最近一周我记录了什么？"

func main() {
	ctx := context.Background()
	debug := flag.Bool("debug", false, "print MemoryAgent debug trace")
	flag.Parse()

	question := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if question == "" {
		question = defaultQuestion
	}

	app, err := bootstrap.NewApp(ctx)
	if err != nil {
		log.Fatalf("bootstrap app failed: %v", err)
	}
	defer app.Close(ctx)

	output, err := app.MemoryAgent.Invoke(ctx, memory_react_agent.AgentInput{
		Message: question,
		TopK:    20,
		Debug:   *debug,
	})
	if err != nil {
		log.Fatalf("memory agent invoke failed: %v", err)
	}

	printJSON(output)
}

func printJSON(v any) {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatalf("marshal output failed: %v", err)
	}
	fmt.Println(string(bytes))
}
