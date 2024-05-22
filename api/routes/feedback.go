package routes

import (
	"camera/controllers"
	"camera/lib"
	"camera/middleware"

	"github.com/gin-gonic/gin"
)

func Feedback(r *gin.Engine) {
	feedback := r.Group("/api/feedback", middleware.JWT([]byte(lib.JwtKey)))

	// 创建反馈
	feedback.POST("/create", controllers.CheckLogin, controllers.CreateFeedback)
	// 反馈列表
	feedback.GET("/list", controllers.CheckLogin, controllers.GetFeedbackList)
}
