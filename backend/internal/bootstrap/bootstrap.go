package bootstrap

import (
	"context"
	"log"

	"memoryflow/internal/ai/agent/chat_pipeline"
	"memoryflow/internal/ai/agent/knowledge_pipeline"
	"memoryflow/internal/ai/embedder"
	"memoryflow/internal/ai/models"
	"memoryflow/internal/ai/reranker"
	"memoryflow/internal/ai/retriever"
	"memoryflow/internal/ai/vectorstore"
	"memoryflow/internal/ai/workflow/memory_analyze"
	"memoryflow/internal/config"
	"memoryflow/internal/domain/repository"
	"memoryflow/internal/domain/service"
	"memoryflow/internal/storage"
	"memoryflow/internal/task"
)

const DefaultConfigPath = "configs/config.yaml"

type App struct {
	Config *config.Config

	MemoryService *service.MemoryService
	TaskService   *service.TaskService

	AnalysisChatModel    *models.ChatModel
	ToolCallingChatModel *models.ArkToolCallingChatModel

	MemoryAnalyzeWorkflow *memory_analyze.Workflow

	MilvusStore     *vectorstore.MilvusStore
	EmbeddingClient *embedder.Client
	MemoryRetriever *retriever.MemoryRetriever
	MemoryReranker  *reranker.MemoryReranker

	ChatPipeline      *chat_pipeline.Pipeline
	KnowledgePipeline *knowledge_pipeline.Pipeline

	Storage *storage.LocalStorage
	Worker  *task.Worker
}

func NewApp(ctx context.Context) (*App, error) {
	cfg, err := config.LoadConfig(DefaultConfigPath)
	if err != nil {
		return nil, err
	}

	db, err := repository.InitSQLite(cfg.Database.DSN)
	if err != nil {
		return nil, err
	}

	memoryRepo := repository.NewSQLiteMemoryRepository(db)
	memoryService := service.NewMemoryService(memoryRepo)

	taskRepo := repository.NewSQLiteTaskRepository(db)
	taskService := service.NewTaskService(taskRepo)

	analysisChatModel := models.NewChatModel(
		cfg.Model.BaseURL,
		cfg.Model.APIKey,
		cfg.Model.ModelName,
	)

	toolCallingChatModel := chat_pipeline.NewModel(models.Config{
		BaseURL:   cfg.Model.BaseURL,
		APIKey:    cfg.Model.APIKey,
		ModelName: cfg.Model.ModelName,
	})

	memoryAnalyzeWorkflow := memory_analyze.NewWorkflow(analysisChatModel)

	milvusStore, err := vectorstore.NewMilvusStore(
		ctx,
		cfg.Milvus.Address,
		cfg.Milvus.Collection,
		cfg.Embedding.Dim,
	)
	if err != nil {
		return nil, err
	}

	if err := milvusStore.EnsureCollection(ctx); err != nil {
		_ = milvusStore.Close(ctx)
		return nil, err
	}

	embeddingClient := embedder.NewClient(
		cfg.Embedding.BaseURL,
		cfg.Embedding.APIKey,
		cfg.Embedding.ModelName,
		cfg.Embedding.Dim,
	)

	memoryRetriever := chat_pipeline.NewRetriever(
		embeddingClient,
		milvusStore,
		memoryService,
	)

	memoryReranker := reranker.NewMemoryReranker()

	knowledgePipeline := knowledge_pipeline.NewPipeline(
		knowledge_pipeline.NewLoader(memoryService),
		knowledge_pipeline.NewIndexer(
			embeddingClient,
			milvusStore,
		),
	)

	chatPipeline, err := chat_pipeline.NewPipeline(
		ctx,
		memoryRetriever,
		memoryService,
		analysisChatModel,
		toolCallingChatModel,
	)
	if err != nil {
		_ = milvusStore.Close(ctx)
		return nil, err
	}

	localStorage := storage.NewLocalStorage(cfg.Storage.UploadDir)

	worker := task.NewWorker(
		taskService,
		memoryService,
		memoryAnalyzeWorkflow,
		embeddingClient,
		milvusStore,
	)

	return &App{
		Config: cfg,

		MemoryService: memoryService,
		TaskService:   taskService,

		AnalysisChatModel:    analysisChatModel,
		ToolCallingChatModel: toolCallingChatModel,

		MemoryAnalyzeWorkflow: memoryAnalyzeWorkflow,

		MilvusStore:     milvusStore,
		EmbeddingClient: embeddingClient,
		MemoryRetriever: memoryRetriever,
		MemoryReranker:  memoryReranker,

		ChatPipeline:      chatPipeline,
		KnowledgePipeline: knowledgePipeline,

		Storage: localStorage,
		Worker:  worker,
	}, nil
}

func (a *App) StartWorker(ctx context.Context) {
	if a == nil || a.Worker == nil {
		return
	}
	go a.Worker.Start(ctx)
}

func (a *App) Close(ctx context.Context) {
	if a == nil || a.MilvusStore == nil {
		return
	}
	if err := a.MilvusStore.Close(ctx); err != nil {
		log.Printf("close milvus store failed: %v", err)
	}
}
