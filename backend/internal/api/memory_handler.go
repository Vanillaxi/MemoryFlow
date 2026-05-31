package api

import (
	"memoryflow/internal/ai/agent/chat_pipeline"
	"memoryflow/internal/ai/agent/knowledge_pipeline"
	"memoryflow/internal/ai/retriever"
	"memoryflow/internal/domain/service"
	"memoryflow/internal/pkg/response"
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
	chatPipeline      *chat_pipeline.Pipeline
	knowledgePipeline *knowledge_pipeline.Pipeline
}

func NewMemoryHandler(
	memoryService *service.MemoryService,
	taskService *service.TaskService,
	storage *storage.LocalStorage,
	memoryRetriever *retriever.MemoryRetriever,
	chatPipeline *chat_pipeline.Pipeline,
	knowledgePipeline *knowledge_pipeline.Pipeline,
) *MemoryHandler {
	return &MemoryHandler{
		memoryService:     memoryService,
		taskService:       taskService,
		storage:           storage,
		memoryRetriever:   memoryRetriever,
		chatPipeline:      chatPipeline,
		knowledgePipeline: knowledgePipeline,
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

func (h *MemoryHandler) SummarizeMemories(c *gin.Context) {
	fromStr := strings.TrimSpace(c.Query("from"))
	toStr := strings.TrimSpace(c.Query("to"))
	if fromStr == "" || toStr == "" {
		response.Error(c, http.StatusBadRequest, "from and to are required, expected YYYY-MM-DD")
		return
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid from format, expected YYYY-MM-DD")
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid to format, expected YYYY-MM-DD")
		return
	}
	to = to.Add(24*time.Hour - time.Second)

	limit := 100
	if limitStr := strings.TrimSpace(c.Query("limit")); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	output, err := h.chatPipeline.Summarize(c.Request.Context(), chat_pipeline.SummaryInput{
		From:  from,
		To:    to,
		Limit: limit,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, output)
}

func (h *MemoryHandler) SearchMemories(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		response.Error(c, http.StatusBadRequest, "q is required")
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

	memoryType := strings.TrimSpace(c.Query("type"))
	if memoryType != "" && memoryType != "text" && memoryType != "image" && memoryType != "mixed" {
		response.Error(c, http.StatusBadRequest, "invalid type, expected text/image/mixed")
		return
	}

	var startTime *time.Time
	if startStr := strings.TrimSpace(c.Query("start")); startStr != "" {
		parsed, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid start format, expected YYYY-MM-DD")
			return
		}
		startTime = &parsed
	}

	var endTime *time.Time
	if endStr := strings.TrimSpace(c.Query("end")); endStr != "" {
		parsed, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid end format, expected YYYY-MM-DD")
			return
		}

		// end=2026-05-28 表示包含 2026-05-28 当天
		parsed = parsed.Add(24*time.Hour - time.Second)
		endTime = &parsed
	}

	results, err := h.memoryRetriever.Retrieve(
		c.Request.Context(),
		q,
		retriever.RetrieveOptions{
			TopK:      topK,
			Type:      memoryType,
			StartTime: startTime,
			EndTime:   endTime,
		},
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.OK(c, gin.H{
		"query":   q,
		"top_k":   topK,
		"type":    memoryType,
		"results": results,
	})
}

func (h *MemoryHandler) Ask(c *gin.Context) {
	debug := c.Query("debug") == "true"
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		response.Error(c, http.StatusBadRequest, "q is required")
		return
	}

	topK := 20
	topKStr := c.DefaultQuery("top_k", "20")
	if parsed, err := strconv.Atoi(topKStr); err == nil && parsed > 0 {
		topK = parsed
	}
	if topK > 20 {
		topK = 20
	}

	memoryType := strings.TrimSpace(c.Query("type"))
	if memoryType != "" && memoryType != "text" && memoryType != "image" && memoryType != "mixed" {
		response.Error(c, http.StatusBadRequest, "invalid type, expected text/image/mixed")
		return
	}

	var startTime *time.Time
	if startStr := strings.TrimSpace(c.Query("start")); startStr != "" {
		parsed, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid start format, expected YYYY-MM-DD")
			return
		}
		startTime = &parsed
	}

	var endTime *time.Time
	if endStr := strings.TrimSpace(c.Query("end")); endStr != "" {
		parsed, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid end format, expected YYYY-MM-DD")
			return
		}
		parsed = parsed.Add(24*time.Hour - time.Second)
		endTime = &parsed
	}

	result, err := h.chatPipeline.Invoke(c.Request.Context(), chat_pipeline.ChatInput{
		Message:   q,
		TopK:      topK,
		Type:      memoryType,
		StartTime: startTime,
		EndTime:   endTime,
		Debug:     debug,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.OK(c, result)
}

func (h *MemoryHandler) ReanalyzeMemory(c *gin.Context) {
	idStr := c.Param("id")

	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id64 == 0 {
		response.Error(c, http.StatusBadRequest, "invalid memory id")
		return
	}

	memoryID := uint(id64)

	item, err := h.memoryService.GetByID(c.Request.Context(), memoryID)
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}

	var taskType string

	switch item.Type {
	case "text":
		taskType = service.TaskTypeTextAnalyze
	case "image":
		taskType = service.TaskTypeImageAnalyze
	default:
		response.Error(c, http.StatusBadRequest, "unsupported memory type")
		return
	}

	task, err := h.taskService.CreateTask(c.Request.Context(), taskType, item.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.OK(c, gin.H{
		"memory_id": item.ID,
		"task":      task,
	})
}

func (h *MemoryHandler) ReindexMemories(c *gin.Context) {
	batchSize := 100

	if batchSizeStr := c.DefaultQuery("batch_size", "100"); batchSizeStr != "" {
		if parsed, err := strconv.Atoi(batchSizeStr); err == nil && parsed > 0 {
			batchSize = parsed
		}
	}

	result, err := h.knowledgePipeline.ReindexAll(c.Request.Context(), knowledge_pipeline.ReindexInput{
		BatchSize: batchSize,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.OK(c, result)
}

func (h *MemoryHandler) ListAgentTools(c *gin.Context) {
	infos, err := h.chatPipeline.DebugListTools(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.OK(c, infos)
}
