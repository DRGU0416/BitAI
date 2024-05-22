package controllers

import (
	"fmt"
	"net/http"

	"camera/models"

	"github.com/gin-gonic/gin"
)

var (
	productList map[string][]models.Product
)

func init() {
	//预加载产品
	if err := loadProduct(); err != nil {
		logApi.Fatal(err)
	}
}

// 加载产品列表
func loadProduct() error {
	productList = make(map[string][]models.Product)
	product := &models.Product{}
	bundleids, err := product.GetBundleIds()
	if err != nil {
		return err
	}

	for _, bundleId := range bundleids {
		list, err := product.GetListByBundleId(bundleId)
		if err != nil {
			return err
		}
		productList[bundleId] = list
	}

	if len(productList) == 0 {
		return fmt.Errorf("init product failed")
	}
	return nil
}

// 产品列表
func ProductList(c *gin.Context) {
	bundleId := c.GetHeader("bundleid")
	if products, ok := productList[bundleId]; ok {
		c.JSON(http.StatusOK, Response{SUCCESS, products})
		return
	}
	c.JSON(http.StatusOK, Response{FAILURE, "加载失败"})
}

func UserProductPay(c *gin.Context) {
	c.JSON(http.StatusOK, Response{SUCCESS, ""})
}
