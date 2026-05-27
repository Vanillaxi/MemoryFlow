package main

import (
	"context"
	"fmt"
	"log"
	"memoryflow/internal/ai/aimodel"
	"memoryflow/internal/ai/workflow/text_analyze"
	"memoryflow/internal/api"
	"memoryflow/internal/config"
	"memoryflow/internal/repository"
	"memoryflow/internal/service"
	"memoryflow/internal/storage"
	"memoryflow/internal/task"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatal("load config failed", err)
	}

	db, err := repository.InitSQLite(cfg.Database.DSN)
	if err != nil {
		log.Fatalf("init sqlite failed: %v", err)
	}

	//初始化Service
	memoryRepo := repository.NewSQLiteMemoryRepository(db)
	memoryService := service.NewMemoryService(memoryRepo)

	taskRepo := repository.NewSQLiteTaskRepository(db)
	taskService := service.NewTaskService(taskRepo)

	//初始化worker
	chatModel := aimodel.NewChatModel(
		cfg.Model.BaseURL,
		cfg.Model.APIKey,
		cfg.Model.ModelName,
	)

	textAnalyzeWorkflow := text_analyze.NewWorkflow(chatModel)

	worker := task.NewWorker(
		taskService,
		memoryService,
		textAnalyzeWorkflow,
	)
	go worker.Start(context.Background())

	//初始化storage
	localStorage := storage.NewLocalStorage(cfg.Storage.UploadDir)

	//初始化Handler
	memoryHandler := api.NewMemoryHandler(memoryService, taskService, localStorage)
	taskHandler := api.NewTaskHandler(taskService)

	r := gin.Default()
	api.RegisterRoutes(r, memoryHandler, taskHandler, cfg.Storage.UploadDir)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server run failed: %v", err)
	}

}
