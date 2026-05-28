package main

import (
	"context"
	"fmt"
	"log"
	"memoryflow/internal/ai/aimodel"
	"memoryflow/internal/ai/embedding"
	"memoryflow/internal/ai/reranker"
	"memoryflow/internal/ai/retriever"
	"memoryflow/internal/ai/vectorstore"
	"memoryflow/internal/ai/workflow/image_analyze"
	"memoryflow/internal/ai/workflow/rag_answer"
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
	//config
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatal("load config failed", err)
	}

	//初始化sqlite
	db, err := repository.InitSQLite(cfg.Database.DSN)
	if err != nil {
		log.Fatalf("init sqlite failed: %v", err)
	}

	//初始化repo/Service
	memoryRepo := repository.NewSQLiteMemoryRepository(db)
	memoryService := service.NewMemoryService(memoryRepo)

	taskRepo := repository.NewSQLiteTaskRepository(db)
	taskService := service.NewTaskService(taskRepo)

	//初始化chatmodel+textAnalyzeWorkflow
	chatModel := aimodel.NewChatModel(
		cfg.Model.BaseURL,
		cfg.Model.APIKey,
		cfg.Model.ModelName,
	)
	textAnalyzeWorkflow := text_analyze.NewWorkflow(chatModel)
	imageAnalyzeWorkflow := image_analyze.NewWorkflow()

	//初始化MilvusStore
	milvusStore, err := vectorstore.NewMilvusStore(
		context.Background(),
		cfg.Milvus.Address,
		cfg.Milvus.Collection,
		cfg.Embedding.Dim,
	)
	if err != nil {
		log.Fatalf("init milvus store failed: %v", err)
	}
	if err := milvusStore.EnsureCollection(context.Background()); err != nil {
		log.Fatalf("ensure collection failed: %v", err)
	}
	defer func() {
		if err := milvusStore.Close(context.Background()); err != nil {
			log.Printf("close milvus store failed: %v", err)
		}
	}()

	//embeddingClient
	embeddingClient := embedding.NewClient(
		cfg.Embedding.BaseURL,
		cfg.Embedding.APIKey,
		cfg.Embedding.ModelName,
		cfg.Embedding.Dim,
	)

	memoryRetriever := retriever.NewMemoryRetriever(
		embeddingClient,
		milvusStore,
		memoryService,
	)

	memoryReranker := reranker.NewMemoryReranker()

	ragAnswerWorkflow := rag_answer.NewRAGAnswerWorkflow(
		memoryRetriever,
		memoryReranker,
		chatModel,
	)

	//初始化worker
	worker := task.NewWorker(
		taskService,
		memoryService,
		textAnalyzeWorkflow,
		imageAnalyzeWorkflow,
		embeddingClient,
		milvusStore,
	)
	go worker.Start(context.Background())

	//初始化storage
	localStorage := storage.NewLocalStorage(cfg.Storage.UploadDir)

	//初始化Handler
	memoryHandler := api.NewMemoryHandler(memoryService, taskService, localStorage, memoryRetriever, ragAnswerWorkflow)
	taskHandler := api.NewTaskHandler(taskService)

	//初始化router
	r := gin.Default()
	api.RegisterRoutes(r, memoryHandler, taskHandler, cfg.Storage.UploadDir)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server run failed: %v", err)
	}

}
