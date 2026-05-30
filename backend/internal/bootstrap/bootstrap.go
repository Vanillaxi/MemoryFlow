package bootstrap

import (
	"context"
	"log"

	"memoryflow/internal/ai/agent/memory_agent"
	"memoryflow/internal/ai/component/chatmodel"
	"memoryflow/internal/ai/component/embedding"
	"memoryflow/internal/ai/component/reranker"
	"memoryflow/internal/ai/component/retriever"
	"memoryflow/internal/ai/component/vectorstore"
	"memoryflow/internal/ai/pipeline/memory_chat"
	"memoryflow/internal/ai/pipeline/memory_index"
	"memoryflow/internal/ai/pipeline/memory_summary"
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

	AnalysisChatModel *chatmodel.ChatModel
	EinoChatModel     *chatmodel.ArkEinoChatModel

	TextAnalyzeWorkflow  *text_analyze.Workflow
	ImageAnalyzeWorkflow *image_analyze.Workflow

	MilvusStore     *vectorstore.MilvusStore
	EmbeddingClient *embedding.Client
	MemoryRetriever *retriever.MemoryRetriever
	MemoryReranker  *reranker.MemoryReranker

	MemoryChatPipeline    *memory_chat.Pipeline
	MemoryIndexPipeline   *memory_index.Pipeline
	MemorySummaryPipeline *memory_summary.Pipeline
	MemoryAgent           *memory_agent.MemoryAgent

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

	analysisChatModel := chatmodel.NewChatModel(
		cfg.Model.BaseURL,
		cfg.Model.APIKey,
		cfg.Model.ModelName,
	)

	einoChatModel := chatmodel.NewArkEinoChatModel(chatmodel.Config{
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

	memoryChatPipeline, err := memory_chat.NewPipeline(
		ctx,
		memoryRetriever,
		memoryReranker,
		analysisChatModel,
	)
	if err != nil {
		_ = milvusStore.Close(ctx)
		return nil, err
	}

	memoryIndexPipeline := memory_index.NewPipeline(
		memoryService,
		memory_index.NewIndexer(
			embeddingClient,
			milvusStore,
		),
	)

	memorySummaryPipeline := memory_summary.NewPipeline(memoryService, analysisChatModel)

	memoryAgent, err := memory_agent.NewMemoryAgent(
		ctx,
		memoryChatPipeline,
		memoryRetriever,
		memoryService,
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
