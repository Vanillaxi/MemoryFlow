package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"memoryflow/internal/ai/workflow/text_analyze"
	"memoryflow/internal/bootstrap"
)

const defaultText = "今天整理了 MemoryFlow 的 Agent 架构，并补充了调试入口。"

func main() {
	ctx := context.Background()
	content := strings.TrimSpace(strings.Join(os.Args[1:], " "))
	if content == "" {
		content = defaultText
	}

	app, err := bootstrap.NewApp(ctx)
	if err != nil {
		log.Fatalf("bootstrap app failed: %v", err)
	}
	defer app.Close(ctx)

	output, err := app.TextAnalyzeWorkflow.Run(ctx, text_analyze.TextAnalyzeInput{
		ContentText: content,
		CreatedAt:   time.Now(),
	})
	if err != nil {
		log.Fatalf("text analyze failed: %v", err)
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
