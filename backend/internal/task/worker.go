package task

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"memoryflow/internal/ai/workflow/text_analyze"
	"memoryflow/internal/model"
	"memoryflow/internal/service"
	"strings"
	"time"
)

type Worker struct {
	taskService         *service.TaskService
	memoryService       *service.MemoryService
	textAnalyzeWorkflow *text_analyze.Workflow
	interval            time.Duration
}

func NewWorker(taskService *service.TaskService, memoryService *service.MemoryService, textAnalyzeWorkflow *text_analyze.Workflow) *Worker {
	return &Worker{
		taskService:         taskService,
		memoryService:       memoryService,
		textAnalyzeWorkflow: textAnalyzeWorkflow,
		interval:            2 * time.Second,
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
		err = w.handleTextAnalyze(ctx, t)
	case service.TaskTypeImageAnalyze:
		err = w.handleImageAnalyze(ctx, t)
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

func (w *Worker) handleTextAnalyze(ctx context.Context, t model.Task) error {
	item, err := w.memoryService.GetByID(ctx, t.TargetID)
	if err != nil {
		return err
	}

	result, err := w.textAnalyzeWorkflow.Run(ctx, text_analyze.TextAnalyzeInput{
		MemoryID:    item.ID,
		ContentText: item.ContentText,
		Location:    item.Location,
		CreatedAt:   item.CreatedAt,
	})
	if err != nil {
		//task failed
		return err
	}

	//数据库 tags 字段如果是 string，就需要 json.Marshal 成字符串
	tagsBytes, err := json.Marshal(result.Tags)
	if err != nil {
		return err
	}

	return w.memoryService.UpdateAnalysis(
		ctx,
		item.ID,
		result.Summary,
		string(tagsBytes),
		result.Mood,
		result.ImportanceScore,
	)
}

func (w *Worker) handleImageAnalyze(ctx context.Context, t model.Task) error {
	item, err := w.memoryService.GetByID(ctx, t.TargetID)
	if err != nil {
		return err
	}

	summary := "这是一条图片记忆，后续会由多模态模型生成图片描述。"
	if strings.TrimSpace(item.ContentText) != "" {
		summary = "这是一条带有文字说明的图片记忆：" + item.ContentText
	}

	tagsBytes, _ := json.Marshal([]string{"图片", "生活记录"})
	tags := string(tagsBytes)
	mood := "neutral"
	importanceScore := 0.5

	return w.memoryService.UpdateAnalysis(ctx, item.ID, summary, tags, mood, importanceScore)
}

func buildFakeSummary(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return "这是一条生活记录。"
	}
	if len([]rune(content)) > 30 {
		return "这是一条生活记录：" + string([]rune(content)[:30]) + "..."
	}
	return "这是一条生活记录：" + content
}
