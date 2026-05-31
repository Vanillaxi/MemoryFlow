package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	"memoryflow/internal/ai/agent/knowledge_pipeline"
	"memoryflow/internal/bootstrap"
)

const defaultBatchSize = 50

func main() {
	ctx := context.Background()
	batchSizeFlag := flag.Int("batch-size", 0, "number of memories to index per batch")
	flag.Parse()
	batchSize := normalizeBatchSize(*batchSizeFlag, flag.Args())

	app, err := bootstrap.NewApp(ctx)
	if err != nil {
		log.Fatalf("bootstrap app failed: %v", err)
	}
	defer app.Close(ctx)

	output, err := app.KnowledgePipeline.ReindexAll(ctx, knowledge_pipeline.ReindexInput{
		BatchSize: batchSize,
	})
	if err != nil {
		log.Fatalf("memory index reindex failed: %v", err)
	}

	printJSON(output)
}

func normalizeBatchSize(batchSize int, args []string) int {
	if batchSize > 0 {
		return batchSize
	}
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
