package routes

import (
	"camera/controllers"

	"github.com/gin-gonic/gin"
)

func Common(r *gin.Engine) {
	// 检查版本更新
	r.GET("api/checkver", controllers.CheckVersion)
}
