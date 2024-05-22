package routes

import (
	"camera/controllers"
	"camera/lib"
	"camera/middleware"

	"github.com/gin-gonic/gin"
)

func Task(r *gin.Engine) {
	card := r.Group("/api/card", middleware.JWT([]byte(lib.JwtKey)))
	// 上传正面照
	card.POST("/front", controllers.CheckLogin, controllers.UploadUserFrontImage)
	// 检查正面照状态
	card.GET("/front", controllers.CheckLogin, controllers.CheckUserFrontImage)
	// 进入下一步，上传侧面照
	card.POST("/in_side", controllers.CheckLogin, controllers.InUploadSideImageStep)

	// 上传侧面照
	card.POST("/side", controllers.CheckLogin, controllers.UploadUserCardImage)
	// 检查侧面照状态
	card.GET("/side", controllers.CheckLogin, controllers.CheckUserCardImage)
	// 删除侧面照
	card.POST("/removeside", controllers.CheckLogin, controllers.DeleteUserCardImage)

	// 创建分身任务
	card.POST("/create", controllers.CheckLogin, controllers.CreateUserCardTask)
	// 单任务状态
	card.GET("/status", controllers.CheckLogin, controllers.GetCardStatus)
	// 4张分身图
	card.GET("/list", controllers.CheckLogin, controllers.GetCardList)
	// 选择一个分身作为主分身
	card.POST("/select", controllers.CheckLogin, controllers.SelectUserCard)

	// 重置分身
	card.POST("/reset", controllers.CheckLogin, controllers.ResetUserCard)
	// 放弃本次分身任务
	card.POST("/back", controllers.CheckLogin, controllers.BackLastUserCard)

	/**
	========== 写真 ==========
	*/
	task := r.Group("/api/task", middleware.JWT([]byte(lib.JwtKey)))
	// 创建写真任务
	task.POST("/create", controllers.CheckLogin, controllers.CreatePhotoTask)
	// 查看写真任务状态
	task.GET("/status", controllers.CheckLogin, controllers.GetPhotoStatus)
	// 写真列表
	task.GET("/history", controllers.CheckLogin, controllers.GetPhotoHistory)
	// 高清处理
	task.POST("/higher", controllers.CheckLogin, controllers.PhotoImageHigher)
	// 删除写真
	task.POST("/remove", controllers.CheckLogin, controllers.DeletePhotoTask)
	// 收藏
	task.POST("/favorite", controllers.CheckLogin, controllers.PhotoImageFavorite)
	// 我的收藏
	task.GET("/favorite", controllers.CheckLogin, controllers.MyPhotoFavorite)
	// 下载
	task.GET("/download", controllers.CheckLogin, controllers.DownloadPhotoImage)
	// 分享
	task.GET("/share", controllers.CheckLogin, controllers.SharePhotoImage)

	/**
	========== 任务分发 ==========
	*/
	work := r.Group("/api/work")
	// 照片识别
	work.GET("/recognize", controllers.GetPhotoRecognizeTask)
	// Lora模型训练
	work.GET("/lora", controllers.GetLoraTask)
	// 写真
	work.GET("/photo", controllers.GetPhotoTask)
	// 写真高清
	work.GET("/photohr", controllers.GetPhotoHrTask)

	/**
	========== 任务上报 ==========
	*/
	// 照片识别上报
	r.POST("/api/recognize/callback", controllers.ReportPhotoRecognizeTask)
	// Lora模型上报
	r.POST("/api/lora/callback", controllers.ReportLoraTask)
	// 分身图片上报
	r.POST("/api/card/callback", controllers.ReportCardTask)
	// 写真上报
	r.POST("/api/photo/callback", controllers.ReportPhotoTask)
	// 写真上报
	r.POST("/api/photohr/callback", controllers.ReportPhotoHrTask)
}
