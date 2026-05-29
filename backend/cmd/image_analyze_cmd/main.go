package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"memoryflow/internal/ai/workflow/image_analyze"
	"memoryflow/internal/bootstrap"
)

const defaultImageURL = "https://example.com/memory.jpg"

func main() {
	ctx := context.Background()
	imageURL := strings.TrimSpace(strings.Join(os.Args[1:], " "))
	if imageURL == "" {
		imageURL = defaultImageURL
	}

	app, err := bootstrap.NewApp(ctx)
	if err != nil {
		log.Fatalf("bootstrap app failed: %v", err)
	}
	defer app.Close(ctx)

	output, err := app.ImageAnalyzeWorkflow.Run(image_analyze.ImageAnalyzeInput{
		ImageURL:  imageURL,
		CreatedAt: time.Now(),
	})
	if err != nil {
		log.Fatalf("image analyze failed: %v", err)
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
