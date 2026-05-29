package main

import (
	"context"
	"fmt"
	"log"

	"memoryflow/internal/api"
	"memoryflow/internal/bootstrap"

	"github.com/gin-gonic/gin"
)

func main() {
	ctx := context.Background()

	app, err := bootstrap.NewApp(ctx)
	if err != nil {
		log.Fatalf("bootstrap app failed: %v", err)
	}
	defer app.Close(ctx)

	app.StartWorker(ctx)

	memoryHandler := api.NewMemoryHandler(
		app.MemoryService,
		app.TaskService,
		app.Storage,
		app.MemoryRetriever,
		app.MemoryAgent,
		app.MemoryIndexPipeline,
	)
	taskHandler := api.NewTaskHandler(app.TaskService)

	r := gin.Default()
	api.RegisterRoutes(r, memoryHandler, taskHandler, app.Config.Storage.UploadDir)

	addr := fmt.Sprintf(":%d", app.Config.Server.Port)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server run failed: %v", err)
	}
}
