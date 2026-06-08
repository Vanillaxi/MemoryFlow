package api

import (
	"memoryflow/internal/mcp"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, memoryHandler *MemoryHandler, taskHandler *TaskHandler, projectHandler *ProjectHandler, agentHandler *AgentHandler, mcpServer *mcp.Server, uploadDir string) {
	r.GET("health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	//加静态文件映射，使得前端能通过浏览器访问
	r.Static("/uploads", uploadDir)
	r.POST("/agent/chat", agentHandler.Chat)
	if mcpServer != nil {
		r.POST("/mcp/rpc", mcpServer.Handler())
	}
	r.POST("/projects", projectHandler.CreateProject)
	r.GET("/projects", projectHandler.ListProjects)
	r.GET("/projects/:id", projectHandler.GetProject)

	api := r.Group("/api")
	{
		api.POST("/memories/text", memoryHandler.CreateTextMemory)
		api.POST("memories/image", memoryHandler.CreateImageMemory)

		api.GET("/memories/recent", memoryHandler.ListRecent)
		api.GET("/memories/search", memoryHandler.SearchMemories)
		api.GET("/memories/summary", memoryHandler.SummarizeMemories)
		api.GET("/timeline", memoryHandler.GetTimeline)

		api.GET("/tasks/:id", taskHandler.GetTask)
		api.POST("/projects", projectHandler.CreateProject)
		api.GET("/projects", projectHandler.ListProjects)
		api.GET("/projects/:id", projectHandler.GetProject)

		api.POST("/memories/:id/reanalyze", memoryHandler.ReanalyzeMemory)
		api.GET("/memories/ask", memoryHandler.Ask)

		api.POST("/memories/reindex", memoryHandler.ReindexMemories)

		api.GET("/agent/tools", memoryHandler.ListAgentTools)
	}
}
