package routes

import (
	"camera/controllers"

	"github.com/gin-gonic/gin"
)

func Web(r *gin.Engine) {
	// 刷缓存
	r.GET("/api/cache_refresh", controllers.CacheRefresh)
}
