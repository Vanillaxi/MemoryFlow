package api

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine, memoryHandler *MemoryHandler, taskHandler *TaskHandler, uploadDir string) {
	r.GET("health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	//加静态文件映射，使得前端能通过浏览器访问
	r.Static("/uploads", uploadDir)

	api := r.Group("/api")
	{
		api.POST("/memories/text", memoryHandler.CreateTextMemory)
		api.POST("memories/image", memoryHandler.CreateImageMemory)

		api.GET("/memories/recent", memoryHandler.ListRecent)
		api.GET("/timeline", memoryHandler.GetTimeline)

		api.GET("/tasks/:id", taskHandler.GetTask)
	}
}
