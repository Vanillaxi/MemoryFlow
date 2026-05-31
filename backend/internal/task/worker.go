package task

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"memoryflow/internal/ai/embedder"
	"memoryflow/internal/ai/vectorstore"
	"memoryflow/internal/ai/workflow/memory_analyze"
	"memoryflow/internal/domain/model"
	"memoryflow/internal/domain/service"
	"strings"
	"time"
)

type Worker struct {
	taskService           *service.TaskService
	memoryService         *service.MemoryService
	memoryAnalyzeWorkflow memoryAnalyzeWorkflow
	embeddingClient       *embedder.Client
	milvusStore           *vectorstore.MilvusStore
	interval              time.Duration
}

type memoryAnalyzeWorkflow interface {
	Invoke(ctx context.Context, input memory_analyze.AnalyzeInput) (*memory_analyze.AnalyzeResult, error)
}

func NewWorker(
	taskService *service.TaskService,
	memoryService *service.MemoryService,
	memoryAnalyzeWorkflow memoryAnalyzeWorkflow,
	embeddingClient *embedder.Client,
	milvusStore *vectorstore.MilvusStore,
) *Worker {
	return &Worker{
		taskService:           taskService,
		memoryService:         memoryService,
		memoryAnalyzeWorkflow: memoryAnalyzeWorkflow,
		embeddingClient:       embeddingClient,
		milvusStore:           milvusStore,
		interval:              2 * time.Second,
	}
}

func (w *Worker) Start(ctx context.Context) {
	log.Println("[task-worker] started")

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[task-worker] stopped")
			return

		case <-ticker.C:
			w.handlePendingTasks(ctx)
		}
	}
}

func (w *Worker) handlePendingTasks(ctx context.Context) {
	tasks, err := w.taskService.FindPending(ctx, 10)
	if err != nil {
		log.Printf("[task-worker] find pending tasks failed:%v\n", err)
		return
	}

	for _, t := range tasks {
		w.handleTask(ctx, t)
	}

}

func (w *Worker) handleTask(ctx context.Context, t model.Task) {
	if err := w.taskService.UpdateStatus(ctx, t.ID, service.TaskStatusRunning, ""); err != nil {
		log.Printf("[task-worker] update task status failed,task_id=%d,err=%v\n", t.ID, err)
		return
	}

	var err error

	switch t.TaskType {
	case service.TaskTypeTextAnalyze:
		err = w.handleAnalyze(ctx, t)
	case service.TaskTypeImageAnalyze:
		err = w.handleAnalyze(ctx, t)
	case service.TaskTypeEmbedding:
		err = w.handleEmbedding(ctx, t)
	default:
		err = errors.New("unknown task type")
		return
	}

	if err != nil {
		_ = w.taskService.UpdateStatus(ctx, t.ID, service.TaskStatusFailed, err.Error())
		log.Printf("[task-worker] task failed,task_id=%d,err=%v\n", t.ID, err)
		return
	}

	if err = w.taskService.UpdateStatus(ctx, t.ID, service.TaskStatusSuccess, ""); err != nil {
		log.Printf("[task-worker] update task success failed,task_id=%d,err=%v\n", t.ID, err)
	}

}

func (w *Worker) handleAnalyze(ctx context.Context, t model.Task) error {
	item, err := w.memoryService.GetByID(ctx, t.TargetID)
	if err != nil {
		return err
	}

	result, err := w.memoryAnalyzeWorkflow.Invoke(ctx, analyzeInputFromMemory(item))
	if err != nil {
		return err
	}

	tagsBytes, err := json.Marshal(result.Tags)
	if err != nil {
		return err
	}

	if err := w.memoryService.UpdateAnalysis(
		ctx,
		item.ID,
		result.Summary,
		string(tagsBytes),
		result.Mood,
		result.ImportanceScore,
	); err != nil {
		return err
	}

	_, err = w.taskService.CreateTask(ctx, service.TaskTypeEmbedding, item.ID)
	if err != nil {
		return err
	}

	log.Printf("[task-worker] created embedding task, memory_id=%d\n", item.ID)
	return nil
}

func analyzeInputFromMemory(item *model.MemoryItem) memory_analyze.AnalyzeInput {
	return memory_analyze.AnalyzeInput{
		MemoryID:    item.ID,
		Type:        item.Type,
		ImageURL:    item.ImageURL,
		ContentText: item.ContentText,
		Location:    item.Location,
		OccurredAt:  item.OccurredAt,
	}
}

func (w *Worker) handleEmbedding(ctx context.Context, t model.Task) error {
	log.Printf("[task-worker] start embedding, task_id=%d, memory_id=%d\n", t.ID, t.TargetID)

	item, err := w.memoryService.GetByID(ctx, t.TargetID)
	if err != nil {
		return err
	}

	embeddingText := buildEmbeddingText(item)

	vec, err := w.embeddingClient.Embed(ctx, embeddingText)
	if err != nil {
		return err
	}

	memoryID := int64(item.ID)

	if err := w.milvusStore.DeleteMemoryVector(ctx, memoryID); err != nil {
		log.Printf("[task-worker] delete old memory vector skipped, memory_id=%d, err=%v\n", item.ID, err)
	}

	if err := w.milvusStore.InsertMemoryVector(ctx, vectorstore.MemoryVector{
		MemoryID:   int64(item.ID),
		Content:    truncateForMilvus(embeddingText, 4000),
		MemoryType: item.Type,
		OccurredAt: item.OccurredAt.Unix(),
		Vector:     vec,
	}); err != nil {
		return err
	}

	log.Printf("[task-worker] embedding inserted to milvus, memory_id=%d\n", item.ID)
	return nil
}

func buildEmbeddingText(item *model.MemoryItem) string {
	var b strings.Builder

	b.WriteString("类型：")
	b.WriteString(item.Type)
	b.WriteString("\n")

	if strings.TrimSpace(item.ContentText) != "" {
		b.WriteString("内容：")
		b.WriteString(item.ContentText)
		b.WriteString("\n")
	}

	if strings.TrimSpace(item.ImageURL) != "" {
		b.WriteString("图片地址：")
		b.WriteString(item.ImageURL)
		b.WriteString("\n")
	}

	if strings.TrimSpace(item.Summary) != "" {
		b.WriteString("摘要")
		b.WriteString(item.Summary)
		b.WriteString("\n")
	}

	if strings.TrimSpace(item.Tags) != "" {
		b.WriteString("标签：")
		b.WriteString(item.Tags)
		b.WriteString("\n")
	}

	if strings.TrimSpace(item.Location) != "" {
		b.WriteString("地点：")
		b.WriteString(item.Location)
		b.WriteString("\n")
	}

	if strings.TrimSpace(item.ContentText) != "" {
		b.WriteString("情绪：")
		b.WriteString(item.Mood)
		b.WriteString("\n")
	}

	b.WriteString("时间：")
	b.WriteString(item.OccurredAt.Format("2006-01-02 15:04:05"))

	return b.String()
}

func truncateForMilvus(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes])
}
