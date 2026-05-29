package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"memoryflow/internal/ai/agent/memory_agent"
	"memoryflow/internal/bootstrap"
)

const defaultQuestion = "最近我记录了什么？"

func main() {
	ctx := context.Background()
	question := strings.TrimSpace(strings.Join(os.Args[1:], " "))
	if question == "" {
		question = defaultQuestion
	}

	app, err := bootstrap.NewApp(ctx)
	if err != nil {
		log.Fatalf("bootstrap app failed: %v", err)
	}
	defer app.Close(ctx)

	output, err := app.MemoryAgent.Invoke(ctx, memory_agent.AgentInput{
		Message: question,
		TopK:    20,
		Debug:   true,
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
