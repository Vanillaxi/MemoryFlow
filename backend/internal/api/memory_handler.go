package api

import (
	"memoryflow/internal/ai/retriever"
	"memoryflow/internal/ai/workflow/rag_answer"
	"memoryflow/internal/pkg/response"
	"memoryflow/internal/service"
	"memoryflow/internal/storage"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type MemoryHandler struct {
	memoryService     *service.MemoryService
	taskService       *service.TaskService
	storage           *storage.LocalStorage
	memoryRetriever   *retriever.MemoryRetriever
	ragAnswerWorkflow *rag_answer.RAGAnswerWorkflow
}

func NewMemoryHandler(memoryService *service.MemoryService, taskService *service.TaskService, storage *storage.LocalStorage, memoryRetriever *retriever.MemoryRetriever, ragAnswerWorkflow *rag_answer.RAGAnswerWorkflow) *MemoryHandler {
	return &MemoryHandler{
		memoryService:     memoryService,
		taskService:       taskService,
		storage:           storage,
		memoryRetriever:   memoryRetriever,
		ragAnswerWorkflow: ragAnswerWorkflow,
	}
}

func (h *MemoryHandler) CreateTextMemory(c *gin.Context) {
	var req service.CreateTextMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.ContentText == "" {
		response.Error(c, http.StatusBadRequest, "content text is required")
		return
	}

	item, err := h.memoryService.CreateTextMemory(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	task, err := h.taskService.CreateTask(c.Request.Context(), service.TaskTypeTextAnalyze, item.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.OK(c, gin.H{
		"memory": item,
		"task":   task,
	})
}

func (h *MemoryHandler) CreateImageMemory(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "image file is required")
		return
	}

	imageURL, err := h.storage.SaveImage(file)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	contentText := c.PostForm("content_text")
	location := c.PostForm("location")

	var occurredAt time.Time
	occurredAtStr := c.PostForm("occurred_at")
	if occurredAtStr != "" {
		parsed, err := time.Parse(time.RFC3339, occurredAtStr)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "occurred_at must be RFC3339 format")
			return
		}
		occurredAt = parsed
	}

	item, err := h.memoryService.CreateImageMemory(c.Request.Context(), &service.CreateImageMemoryRequest{
		ContentText: contentText,
		ImageURL:    imageURL,
		OccurredAt:  occurredAt,
		Location:    location,
	})

	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	task, err := h.taskService.CreateTask(c.Request.Context(), service.TaskTypeImageAnalyze, item.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.OK(c, gin.H{
		"memory": item,
		"task":   task,
	})

}

func (h *MemoryHandler) ListRecent(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	items, err := h.memoryService.ListRecent(c.Request.Context(), limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, items)
}

func (h *MemoryHandler) GetTimeline(c *gin.Context) {
	startStr := c.Query("start")
	endStr := c.Query("end")

	if startStr == "" || endStr == "" {
		response.Error(c, http.StatusBadRequest, "start or end is required,format:YYYY-MM-DD")
		return
	}

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid start format,expected YYYY-MM-DD")
		return
	}

	end, err := time.Parse("2006-01-02 ", endStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid end format,expected YYYY-MM-DD")
		return
	}

	groups, err := h.memoryService.GetTimeline(c.Request.Context(), start, end)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.OK(c, groups)

}

func (h *MemoryHandler) SearchMemories(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		response.Error(c, http.StatusBadRequest, "query param is required")
		return
	}

	topK := 5
	topKStr := c.DefaultQuery("top_k", "5")
	if parsed, err := strconv.Atoi(topKStr); err == nil && parsed > 0 {
		topK = parsed
	}
	if topK > 20 {
		topK = 20
	}

	results, err := h.memoryRetriever.Retrieve(
		c.Request.Context(),
		q,
		retriever.RetrieveOptions{
			TopK: topK,
		},
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.OK(c, gin.H{
		"query":  q,
		"top_k":  topK,
		"result": results,
	})

}

func (h *MemoryHandler) Ask(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		response.Error(c, http.StatusBadRequest, "q is required")
		return
	}

	result, err := h.ragAnswerWorkflow.Answer(c.Request.Context(), q)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.OK(c, result)
}
