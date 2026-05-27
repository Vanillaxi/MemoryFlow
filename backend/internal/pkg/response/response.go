package response

import "github.com/gin-gonic/gin"

// 创建统一响应
func OK(c *gin.Context, data any) {
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": data,
	})
}

func Error(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{
		"code": status,
		"msg":  msg,
		"data": nil,
	})
}
