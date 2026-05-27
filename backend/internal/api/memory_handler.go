package api

import (
	"memoryflow/internal/pkg/response"
	"memoryflow/internal/service"
	"memoryflow/internal/storage"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MemoryHandler struct {
	service *service.MemoryService
	storage *storage.LocalStorage
}

func NewMemoryHandler(service *service.MemoryService, storage *storage.LocalStorage) *MemoryHandler {
	return &MemoryHandler{
		service: service,
		storage: storage,
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

	item, err := h.service.CreateTextMemory(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.OK(c, item)
}

func (h *MemoryHandler) ListRecent(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)

	items, err := h.service.ListRecent(c.Request.Context(), limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, items)
}
