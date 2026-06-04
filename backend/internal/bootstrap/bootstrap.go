package bootstrap

import (
	"context"
	"log"
	"time"

	"memoryflow/internal/ai/agent"
	"memoryflow/internal/ai/agent/chat_pipeline"
	"memoryflow/internal/ai/agent/knowledge_pipeline"
	"memoryflow/internal/ai/agent/project_pipeline"
	"memoryflow/internal/ai/embedder"
	"memoryflow/internal/ai/models"
	"memoryflow/internal/ai/reranker"
	"memoryflow/internal/ai/retriever"
	"memoryflow/internal/ai/tools"
	githubtool "memoryflow/internal/ai/tools/github"
	memorytool "memoryflow/internal/ai/tools/memory"
	systemtool "memoryflow/internal/ai/tools/system"
	"memoryflow/internal/ai/vectorstore"
	"memoryflow/internal/ai/workflow/memory_analyze"
	"memoryflow/internal/config"
	"memoryflow/internal/domain/repository"
	"memoryflow/internal/domain/service"
	"memoryflow/internal/storage"
	"memoryflow/internal/task"
)

const DefaultConfigPath = "configs/config.yaml"

const milvusStartupTimeout = 3 * time.Second

type App struct {
	Config *config.Config

	MemoryService  *service.MemoryService
	TaskService    *service.TaskService
	ProjectService *service.ProjectService

	AnalysisChatModel    *models.ChatModel
	ToolCallingChatModel *models.ArkToolCallingChatModel

	MemoryAnalyzeWorkflow *memory_analyze.Workflow

	MilvusStore     *vectorstore.MilvusStore
	EmbeddingClient *embedder.Client
	MemoryRetriever *retriever.MemoryRetriever
	MemoryReranker  *reranker.MemoryReranker

	ChatPipeline      *chat_pipeline.Pipeline
	KnowledgePipeline *knowledge_pipeline.Pipeline
	Agent             *agent.Agent

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
	projectRepo := repository.NewSQLiteProjectRepository(db)
	projectService := service.NewProjectService(projectRepo)

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

	var milvusStore *vectorstore.MilvusStore
	vectorStoreForRuntime := any(vectorstore.NewDisabledStore())
	milvusCtx, cancelMilvus := context.WithTimeout(ctx, milvusStartupTimeout)
	if store, err := vectorstore.NewMilvusStore(
		milvusCtx,
		cfg.Milvus.Address,
		cfg.Milvus.Collection,
		cfg.Embedding.Dim,
	); err != nil {
		log.Printf("[bootstrap] milvus unavailable, vector features disabled: %v", err)
	} else if err := store.EnsureCollection(milvusCtx); err != nil {
		_ = store.Close(context.Background())
		log.Printf("[bootstrap] milvus unavailable, vector features disabled: %v", err)
	} else {
		milvusStore = store
		vectorStoreForRuntime = store
	}
	cancelMilvus()

	embeddingClient := embedder.NewClient(
		cfg.Embedding.BaseURL,
		cfg.Embedding.APIKey,
		cfg.Embedding.ModelName,
		cfg.Embedding.Dim,
	)

	memoryRetriever := chat_pipeline.NewRetriever(
		embeddingClient,
		vectorStoreForRuntime.(retriever.VectorStore),
		memoryService,
	)

	memoryReranker := reranker.NewMemoryReranker()

	knowledgePipeline := knowledge_pipeline.NewPipeline(
		knowledge_pipeline.NewLoader(memoryService),
		knowledge_pipeline.NewIndexer(
			embeddingClient,
			vectorStoreForRuntime.(interface {
				DeleteMemoryVector(context.Context, int64) error
				InsertMemoryVector(context.Context, vectorstore.MemoryVector) error
			}),
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

	toolRegistry := tools.NewToolRegistry()
	currentTimeTool := systemtool.NewGetCurrentTimeTool()
	queryMemoryTool := memorytool.NewQueryLongTermMemoryTool(memoryRetriever, memoryService, nil)
	recentCommitsTool := githubtool.NewGetRecentCommitsTool(
		cfg.Github.Token,
		cfg.Github.DefaultLimit,
		cfg.Github.DefaultDays,
		"",
		nil,
	)
	recentIssuesTool := githubtool.NewGetRecentIssuesTool(
		cfg.Github.Token,
		cfg.Github.DefaultLimit,
		cfg.Github.DefaultDays,
		"",
		nil,
	)
	pullRequestsTool := githubtool.NewGetPullRequestsTool(
		cfg.Github.Token,
		cfg.Github.DefaultLimit,
		cfg.Github.DefaultDays,
		"",
		nil,
	)
	toolRegistry.Register(currentTimeTool)
	toolRegistry.Register(queryMemoryTool)
	toolRegistry.Register(memorytool.NewGetMemoryDetailTool(memoryService, nil))
	toolRegistry.Register(memorytool.NewAggregateMemoryTool(memoryService, nil))
	toolRegistry.Register(recentCommitsTool)
	toolRegistry.Register(recentIssuesTool)
	toolRegistry.Register(pullRequestsTool)
	pipelineAgent := agent.NewAgent(toolRegistry, analysisChatModel, chatPipeline)
	projectAgent, err := project_pipeline.NewAgent(
		ctx,
		project_pipeline.NewProjectResolver(projectService),
		toolCallingChatModel,
		[]tools.Tool{currentTimeTool, queryMemoryTool, recentCommitsTool, recentIssuesTool, pullRequestsTool},
	)
	if err != nil {
		_ = milvusStore.Close(ctx)
		return nil, err
	}
	pipelineAgent.SetProjectAgent(projectAgent)

	localStorage := storage.NewLocalStorage(cfg.Storage.UploadDir)

	var worker *task.Worker
	if milvusStore != nil {
		worker = task.NewWorker(
			taskService,
			memoryService,
			memoryAnalyzeWorkflow,
			embeddingClient,
			milvusStore,
		)
	} else {
		log.Printf("[bootstrap] task worker disabled because milvus is unavailable")
	}

	return &App{
		Config: cfg,

		MemoryService:  memoryService,
		TaskService:    taskService,
		ProjectService: projectService,

		AnalysisChatModel:    analysisChatModel,
		ToolCallingChatModel: toolCallingChatModel,

		MemoryAnalyzeWorkflow: memoryAnalyzeWorkflow,

		MilvusStore:     milvusStore,
		EmbeddingClient: embeddingClient,
		MemoryRetriever: memoryRetriever,
		MemoryReranker:  memoryReranker,

		ChatPipeline:      chatPipeline,
		KnowledgePipeline: knowledgePipeline,
		Agent:             pipelineAgent,

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
