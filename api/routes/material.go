package routes

import (
	"camera/controllers"

	"github.com/gin-gonic/gin"
)

func Material(r *gin.Engine) {
	material := r.Group("/api/material")
	// 模板列表
	material.GET("/template", controllers.MaterialTemplate)
	// 造型列表
	material.GET("/pose", controllers.MaterialPose)
}
