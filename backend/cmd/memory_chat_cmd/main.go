package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"memoryflow/internal/ai/pipeline/memory_chat"
	"memoryflow/internal/bootstrap"
)

const defaultQuestion = "我最近在做什么项目？"

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

	output, err := app.MemoryChatPipeline.Run(ctx, memory_chat.ChatInput{
		Question: question,
		TopK:     20,
	})
	if err != nil {
		log.Fatalf("memory chat pipeline failed: %v", err)
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
