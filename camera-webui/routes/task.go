package routes

import (
	"camera-webui/controllers"

	"github.com/labstack/echo/v4"
)

func Task(e *echo.Echo) {
	task := e.Group("/api/task")
	// 创建任务
	task.POST("/create", controllers.CreateSDTask)

	check := e.Group("/api/check")
	check.POST("/create", controllers.CreateCheckTask)
}
