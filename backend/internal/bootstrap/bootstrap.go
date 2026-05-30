package bootstrap

import (
	"context"
	"log"

	"memoryflow/internal/ai/agent/memory_chat_pipeline"
	"memoryflow/internal/ai/agent/memory_index_pipeline"
	"memoryflow/internal/ai/agent/memory_react_agent"
	"memoryflow/internal/ai/agent/memory_summary_pipeline"
	"memoryflow/internal/ai/embedder"
	"memoryflow/internal/ai/indexer"
	"memoryflow/internal/ai/loader"
	"memoryflow/internal/ai/models"
	"memoryflow/internal/ai/reranker"
	"memoryflow/internal/ai/retriever"
	"memoryflow/internal/ai/vectorstore"
	"memoryflow/internal/ai/workflow/image_analyze"
	"memoryflow/internal/ai/workflow/text_analyze"
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

	AnalysisChatModel *models.ChatModel
	EinoChatModel     *models.ArkEinoChatModel

	TextAnalyzeWorkflow  *text_analyze.Workflow
	ImageAnalyzeWorkflow *image_analyze.Workflow

	MilvusStore     *vectorstore.MilvusStore
	EmbeddingClient *embedder.Client
	MemoryRetriever *retriever.MemoryRetriever
	MemoryReranker  *reranker.MemoryReranker

	MemoryChatPipeline    *memory_chat_pipeline.Pipeline
	MemoryIndexPipeline   *memory_index_pipeline.Pipeline
	MemorySummaryPipeline *memory_summary_pipeline.Pipeline
	MemoryAgent           *memory_react_agent.MemoryAgent

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

	einoChatModel := models.NewArkEinoChatModel(models.Config{
		BaseURL:   cfg.Model.BaseURL,
		APIKey:    cfg.Model.APIKey,
		ModelName: cfg.Model.ModelName,
	})

	textAnalyzeWorkflow := text_analyze.NewWorkflow(analysisChatModel)
	imageAnalyzeWorkflow := image_analyze.NewWorkflow()

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

	memoryRetriever := retriever.NewMemoryRetriever(
		embeddingClient,
		milvusStore,
		memoryService,
	)

	memoryReranker := reranker.NewMemoryReranker()

	memoryChatPipeline, err := memory_chat_pipeline.NewPipeline(
		ctx,
		memoryRetriever,
		memoryReranker,
		analysisChatModel,
	)
	if err != nil {
		_ = milvusStore.Close(ctx)
		return nil, err
	}

	memoryIndexPipeline := memory_index_pipeline.NewPipeline(
		loader.NewMemoryLoader(memoryService),
		indexer.NewMemoryIndexer(
			embeddingClient,
			milvusStore,
		),
	)

	memorySummaryPipeline := memory_summary_pipeline.NewPipeline(memoryService, analysisChatModel)

	memoryAgent, err := memory_react_agent.NewMemoryAgent(
		ctx,
		memoryChatPipeline,
		memoryRetriever,
		memoryService,
		memorySummaryPipeline,
		einoChatModel,
	)
	if err != nil {
		_ = milvusStore.Close(ctx)
		return nil, err
	}

	localStorage := storage.NewLocalStorage(cfg.Storage.UploadDir)

	worker := task.NewWorker(
		taskService,
		memoryService,
		textAnalyzeWorkflow,
		imageAnalyzeWorkflow,
		embeddingClient,
		milvusStore,
	)

	return &App{
		Config: cfg,

		MemoryService: memoryService,
		TaskService:   taskService,

		AnalysisChatModel: analysisChatModel,
		EinoChatModel:     einoChatModel,

		TextAnalyzeWorkflow:  textAnalyzeWorkflow,
		ImageAnalyzeWorkflow: imageAnalyzeWorkflow,

		MilvusStore:     milvusStore,
		EmbeddingClient: embeddingClient,
		MemoryRetriever: memoryRetriever,
		MemoryReranker:  memoryReranker,

		MemoryChatPipeline:    memoryChatPipeline,
		MemoryIndexPipeline:   memoryIndexPipeline,
		MemorySummaryPipeline: memorySummaryPipeline,
		MemoryAgent:           memoryAgent,

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
