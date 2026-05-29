package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"memoryflow/internal/ai/pipeline/memory_index"
	"memoryflow/internal/bootstrap"
)

const defaultBatchSize = 50

func main() {
	ctx := context.Background()
	batchSize := parseBatchSize(os.Args[1:])

	app, err := bootstrap.NewApp(ctx)
	if err != nil {
		log.Fatalf("bootstrap app failed: %v", err)
	}
	defer app.Close(ctx)

	output, err := app.MemoryIndexPipeline.ReindexAll(ctx, memory_index.ReindexInput{
		BatchSize: batchSize,
	})
	if err != nil {
		log.Fatalf("memory index reindex failed: %v", err)
	}

	printJSON(output)
}

func parseBatchSize(args []string) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return defaultBatchSize
	}
	value, err := strconv.Atoi(args[0])
	if err != nil || value <= 0 {
		return defaultBatchSize
	}
	return value
}

func printJSON(v any) {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatalf("marshal output failed: %v", err)
	}
	fmt.Println(string(bytes))
}
