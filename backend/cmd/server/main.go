package main

import (
	"context"
	"fmt"
	"log"
	"memoryflow/internal/ai/agent/memory_agent"
	"memoryflow/internal/ai/aimodel"
	"memoryflow/internal/ai/embedding"
	"memoryflow/internal/ai/pipelines/memory_chat_pipeline"
	"memoryflow/internal/ai/reranker"
	"memoryflow/internal/ai/retriever"
	"memoryflow/internal/ai/vectorstore"
	"memoryflow/internal/ai/workflow/image_analyze"
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
	ctx := context.Background()

	// config
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatal("load config failed", err)
	}

	// 初始化 sqlite
	db, err := repository.InitSQLite(cfg.Database.DSN)
	if err != nil {
		log.Fatalf("init sqlite failed: %v", err)
	}

	// 初始化 repo/service
	memoryRepo := repository.NewSQLiteMemoryRepository(db)
	memoryService := service.NewMemoryService(memoryRepo)

	taskRepo := repository.NewSQLiteTaskRepository(db)
	taskService := service.NewTaskService(taskRepo)

	// 初始化 chat model + workflow
	chatModel := aimodel.NewChatModel(
		cfg.Model.BaseURL,
		cfg.Model.APIKey,
		cfg.Model.ModelName,
	)

	textAnalyzeWorkflow := text_analyze.NewWorkflow(chatModel)
	imageAnalyzeWorkflow := image_analyze.NewWorkflow()

	// 初始化 MilvusStore
	milvusStore, err := vectorstore.NewMilvusStore(
		ctx,
		cfg.Milvus.Address,
		cfg.Milvus.Collection,
		cfg.Embedding.Dim,
	)
	if err != nil {
		log.Fatalf("init milvus store failed: %v", err)
	}

	if err := milvusStore.EnsureCollection(ctx); err != nil {
		log.Fatalf("ensure collection failed: %v", err)
	}

	defer func() {
		if err := milvusStore.Close(ctx); err != nil {
			log.Printf("close milvus store failed: %v", err)
		}
	}()

	// 初始化 embedding client
	embeddingClient := embedding.NewClient(
		cfg.Embedding.BaseURL,
		cfg.Embedding.APIKey,
		cfg.Embedding.ModelName,
		cfg.Embedding.Dim,
	)

	// 初始化 retriever/reranker
	memoryRetriever := retriever.NewMemoryRetriever(
		embeddingClient,
		milvusStore,
		memoryService,
	)

	memoryReranker := reranker.NewMemoryReranker()

	//  Eino memory chat pipeline
	memoryChatPipeline, err := memory_chat_pipeline.NewPipeline(
		ctx,
		memoryRetriever,
		memoryReranker,
		chatModel,
	)
	if err != nil {
		log.Fatalf("init memory chat pipeline failed: %v", err)
	}

	//MemoryAgent
	memoryAgent := memory_agent.NewMemoryAgent(
		memoryChatPipeline,
		memoryRetriever,
		memoryService,
		chatModel,
	)

	// 初始化 worker
	worker := task.NewWorker(
		taskService,
		memoryService,
		textAnalyzeWorkflow,
		imageAnalyzeWorkflow,
		embeddingClient,
		milvusStore,
	)
	go worker.Start(ctx)

	// 初始化 storage
	localStorage := storage.NewLocalStorage(cfg.Storage.UploadDir)

	// 初始化 handler
	memoryHandler := api.NewMemoryHandler(
		memoryService,
		taskService,
		localStorage,
		memoryRetriever,
		memoryAgent,
	)
	taskHandler := api.NewTaskHandler(taskService)

	// 初始化 router
	r := gin.Default()
	api.RegisterRoutes(r, memoryHandler, taskHandler, cfg.Storage.UploadDir)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server run failed: %v", err)
	}
}
