package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"

	"memoryflow/internal/ai/agent/chat_pipeline"
	"memoryflow/internal/bootstrap"
)

const defaultQuestion = "我最近在做什么项目？"

func main() {
	ctx := context.Background()
	debug := flag.Bool("debug", false, "print chat pipeline ReAct trace")
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

	output, err := app.ChatPipeline.Invoke(ctx, chat_pipeline.ChatInput{
		Message: question,
		TopK:    20,
		Debug:   *debug,
	})
	if err != nil {
		log.Fatalf("chat pipeline failed: %v", err)
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
