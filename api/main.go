package main

import (
	"net/http"

	_ "camera/config"
	"camera/middleware"
	_ "camera/models"
	"camera/routes"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
		MaxAge:           1000,
	}))
	recoverWriter := &lumberjack.Logger{
		Filename:   "logs/gin.log",
		MaxSize:    4,
		MaxBackups: 2,
		LocalTime:  true,
	}
	r.Use(gin.RecoveryWithWriter(recoverWriter))
	middleware.ErrJWTMissing = gin.H{
		"code": 3,
		"data": "请重新登录",
	}

	// Web
	routes.Web(r)

	// Common
	routes.Common(r)

	// 登录
	routes.Login(r)

	// 模板
	routes.Material(r)

	// 任务
	routes.Task(r)

	// 产品
	routes.Product(r)

	// 支付
	routes.Pay(r)

	// 用户中心
	routes.Center(r)

	// 用户反馈
	routes.Feedback(r)

	port := viper.GetString("port")
	r.Run(":" + port)
}
