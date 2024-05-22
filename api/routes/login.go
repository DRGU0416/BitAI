package routes

import (
	"camera/controllers"

	"github.com/gin-gonic/gin"
)

func Login(r *gin.Engine) {
	// 发送短信验证码
	r.GET("api/smss", controllers.SendMessage)
	// 友盟 一键登录 获取手机号
	r.POST("api/umeng/one_token", controllers.UmengOneClickTokenValidate)
	// 登录 短信验证码或一键登录token
	r.POST("api/login", controllers.Login)
}
