package api

import "github.com/gin-gonic/gin"

func RegisterRouters(r *gin.Engine, memoryHandler *MemoryHandler) {
	r.GET("health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	api := r.Group("/api")
	{
		api.POST("/memories/text", memoryHandler.CreateTextMemory)
		api.GET("/memories/recent", memoryHandler.ListRecent)
	}
}
