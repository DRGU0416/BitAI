package routes

import (
	"camera/controllers"
	"camera/lib"
	"camera/middleware"

	"github.com/gin-gonic/gin"
)

func Center(r *gin.Engine) {
	center := r.Group("/api/center", middleware.JWT([]byte(lib.JwtKey)))

	// 用户进度
	center.GET("/progress", controllers.CheckLogin, controllers.GetUserProgress)

	// 个人中心
	center.GET("/personal", controllers.CheckLogin, controllers.PersonalCenter)

	// 用户注销
	center.GET("/logout", controllers.CheckLogin, controllers.Logout)

	// 系统消息
	center.GET("/sys_message", controllers.CheckLogin, controllers.SystemMessage)

	// 钻石变动记录
	center.GET("/diamond", controllers.CheckLogin, controllers.DiamondChangeRecord)

	// 上报基础数据(首次启动APP)
	// center.POST("/bai", controllers.ReportBaseApp, controllers.CheckLogin)
}
