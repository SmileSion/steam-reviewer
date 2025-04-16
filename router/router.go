package router

import (
	"github.com/gin-gonic/gin"
	"steam-reviewer/handler"
	"strings"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 1. API 路由
	api := r.Group("/api")
	{
		api.GET("/review", handler.HandleReview)
		api.GET("/check-data", handler.HandleCheckData)
	}

	// ✅ 静态资源挂载在 /static 路径，避免与 /api 路由冲突
	r.Static("/static", "./public")

	// ✅ 处理 404 路由时返回前端页面
	r.NoRoute(func(c *gin.Context) {
		// 检查请求路径是否是 API 请求，如果不是则返回前端页面
		if !strings.HasPrefix(c.FullPath(), "/api") {
			c.File("./public/index.html")
		} else {
			c.JSON(404, gin.H{"error": "Not Found"})
		}
	})

	return r
}
