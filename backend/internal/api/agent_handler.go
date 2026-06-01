package api

import (
	"net/http"
	"strings"

	"memoryflow/internal/ai/agent"
	"memoryflow/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type AgentHandler struct {
	agent *agent.Agent
}

func NewAgentHandler(currentAgent *agent.Agent) *AgentHandler {
	return &AgentHandler{agent: currentAgent}
}

func (h *AgentHandler) Chat(c *gin.Context) {
	var input agent.ChatInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(input.Message) == "" {
		response.Error(c, http.StatusBadRequest, "message is required")
		return
	}
	if h == nil || h.agent == nil {
		response.Error(c, http.StatusInternalServerError, "agent is not initialized")
		return
	}

	output, err := h.agent.Chat(c.Request.Context(), input)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, output)
}
