package routes

import (
	"camera/controllers"
	"camera/lib"
	"camera/middleware"

	"github.com/gin-gonic/gin"
)

func Pay(r *gin.Engine) {
	// 苹果支付
	appstore := r.Group("/api/appstore", middleware.JWT([]byte(lib.JwtKey)))
	// 支付
	appstore.POST("/confirm", controllers.CheckLogin, controllers.AppStoreConfirm)
}
