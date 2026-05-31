package task

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"memoryflow/internal/ai/workflow/memory_analyze"
	"memoryflow/internal/domain/model"
	"memoryflow/internal/domain/repository"
	"memoryflow/internal/domain/service"
)

type recordingAnalyzeWorkflow struct {
	inputs []memory_analyze.AnalyzeInput
}

func (w *recordingAnalyzeWorkflow) Invoke(_ context.Context, input memory_analyze.AnalyzeInput) (*memory_analyze.AnalyzeResult, error) {
	w.inputs = append(w.inputs, input)
	return &memory_analyze.AnalyzeResult{
		Summary:         "分析结果",
		Tags:            []string{"测试", "统一分析"},
		Mood:            "positive",
		ImportanceScore: 0.8,
	}, nil
}

func TestWorkerHandleAnalyzeUsesUnifiedWorkflowAndWritesBack(t *testing.T) {
	tests := []struct {
		name string
		item model.MemoryItem
	}{
		{
			name: "text memory",
			item: model.MemoryItem{Type: "text", ContentText: "整理入口", OccurredAt: time.Now()},
		},
		{
			name: "image memory",
			item: model.MemoryItem{Type: "image", ImageURL: "/uploads/photo.jpg", OccurredAt: time.Now()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			memoryService, taskService, memoryRepo := newWorkerTestServices(t)
			if err := memoryRepo.Create(ctx, &tt.item); err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			workflow := &recordingAnalyzeWorkflow{}
			worker := NewWorker(taskService, memoryService, workflow, nil, nil)

			if err := worker.handleAnalyze(ctx, model.Task{TargetID: tt.item.ID}); err != nil {
				t.Fatalf("handleAnalyze() error = %v", err)
			}

			if len(workflow.inputs) != 1 || workflow.inputs[0].Type != tt.item.Type {
				t.Fatalf("workflow inputs = %+v", workflow.inputs)
			}

			got, err := memoryService.GetByID(ctx, tt.item.ID)
			if err != nil {
				t.Fatalf("GetByID() error = %v", err)
			}
			if got.Summary != "分析结果" || got.Tags != `["测试","统一分析"]` || got.Mood != "positive" || got.ImportanceScore != 0.8 {
				t.Fatalf("updated memory = %+v", got)
			}

			tasks, err := taskService.FindPending(ctx, 10)
			if err != nil {
				t.Fatalf("FindPending() error = %v", err)
			}
			if len(tasks) != 1 || tasks[0].TaskType != service.TaskTypeEmbedding || tasks[0].TargetID != tt.item.ID {
				t.Fatalf("embedding tasks = %+v", tasks)
			}
		})
	}
}

func TestBuildEmbeddingTextKeepsIndexedFields(t *testing.T) {
	text := buildEmbeddingText(&model.MemoryItem{
		Type:        "mixed",
		ContentText: "项目白板",
		ImageURL:    "/uploads/photo.jpg",
		Summary:     "讨论入口整理",
		Tags:        `["开发"]`,
		Location:    "上海",
		Mood:        "positive",
		OccurredAt:  time.Date(2026, 5, 31, 10, 30, 0, 0, time.Local),
	})

	for _, want := range []string{"类型：mixed", "内容：项目白板", "图片地址：/uploads/photo.jpg", "讨论入口整理", `["开发"]`, "地点：上海", "情绪：positive", "时间：2026-05-31 10:30:00"} {
		if !strings.Contains(text, want) {
			t.Fatalf("buildEmbeddingText() = %q, want %q", text, want)
		}
	}
}

func newWorkerTestServices(t *testing.T) (*service.MemoryService, *service.TaskService, repository.MemoryRepository) {
	t.Helper()

	db, err := repository.InitSQLite(filepath.Join(t.TempDir(), "memoryflow.db"))
	if err != nil {
		t.Fatalf("InitSQLite() error = %v", err)
	}
	memoryRepo := repository.NewSQLiteMemoryRepository(db)
	taskRepo := repository.NewSQLiteTaskRepository(db)
	return service.NewMemoryService(memoryRepo), service.NewTaskService(taskRepo), memoryRepo
}
