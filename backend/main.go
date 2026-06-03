package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

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
		app.ChatPipeline,
		app.KnowledgePipeline,
	)
	taskHandler := api.NewTaskHandler(app.TaskService)
	projectHandler := api.NewProjectHandler(app.ProjectService)
	agentHandler := api.NewAgentHandler(app.Agent)

	r := gin.Default()
	api.RegisterRoutes(r, memoryHandler, taskHandler, projectHandler, agentHandler, app.Config.Storage.UploadDir)
	log.Printf("MemoryFlow config loaded from %s", bootstrap.DefaultConfigPath)
	log.Println("MemoryFlow routes: POST /agent/chat, POST /projects, GET /projects")

	addr := fmt.Sprintf(":%d", app.Config.Server.Port)
	if host := strings.TrimSpace(app.Config.Server.Host); host != "" {
		if host != "0.0.0.0" {
			addr = net.JoinHostPort(host, strconv.Itoa(app.Config.Server.Port))
		}
	}
	log.Printf("MemoryFlow server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server run failed: %v", err)
	}
}
