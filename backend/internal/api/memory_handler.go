package api

import (
	"memoryflow/internal/pkg/response"
	"memoryflow/internal/service"
	"memoryflow/internal/storage"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type MemoryHandler struct {
	memoryService *service.MemoryService
	taskService   *service.TaskService
	storage       *storage.LocalStorage
}

func NewMemoryHandler(memoryService *service.MemoryService, taskService *service.TaskService, storage *storage.LocalStorage) *MemoryHandler {
	return &MemoryHandler{
		memoryService: memoryService,
		taskService:   taskService,
		storage:       storage,
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
	}

	task, err := h.taskService.CreateTask(c.Request.Context(), service.TaskTypeImageAnalyze, item.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
	}

	response.OK(c, gin.H{
		"memory": item,
		"task":   task,
	})

}

func (h *MemoryHandler) ListRecent(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)

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

	start, err := time.Parse("2006-01-02 ", startStr)
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
