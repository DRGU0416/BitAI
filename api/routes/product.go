package routes

import (
	"camera/controllers"
	"camera/lib"
	"camera/middleware"

	"github.com/gin-gonic/gin"
)

func Product(r *gin.Engine) {
	product := r.Group("/api/product", middleware.JWT([]byte(lib.JwtKey)))

	// 产品列表
	product.GET("/list", controllers.ProductList)

	// 支付
	product.POST("/pay", controllers.CheckLogin, controllers.UserProductPay)
}
