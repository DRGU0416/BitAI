package controllers

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"camera/lib"
	"camera/logger"
	"camera/models"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
)

var (
	logApi   = logger.New("logs/api.log")
	logOrder = logger.New("logs/order.log")

	ctx  = context.Background()
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

// CheckLogin 判断登录
func CheckLogin(c *gin.Context) {
	user, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusOK, Response{RELOGIN, "请重新登录"})
		c.Abort()
		return
	}
	claims := user.(*jwt.Token).Claims.(jwt.MapClaims)
	cusId := int(claims["cus_id"].(float64))
	exp := fmt.Sprintf("%d", int64(claims["exp"].(float64)))

	var err error
	var rexp string
	if rexp, err = lib.RDB.HGet(ctx, lib.RedisUserToken, strconv.Itoa(int(cusId))).Result(); err != nil {
		logApi.Errorf("[Redis] get hash %s - %d failed: %s", lib.RedisUserToken, cusId, err)
		c.JSON(http.StatusOK, Response{RELOGIN, "请重新登录"})
		c.Abort()
		return
	}

	if rexp != exp {
		if rexp != "" {
			c.JSON(http.StatusOK, Response{RELOGIN, "请重新登录"})
			c.Abort()
			return
		}
		lib.RDB.HSet(ctx, lib.RedisUserToken, cusId, exp)
	}

	c.Set("customer_id", cusId)
	c.Next()
}

// 获取用户信息
func GetUser(c *gin.Context) (models.UserAccount, error) {
	cusId, ok := c.Get("customer_id")
	if !ok {
		return models.UserAccount{}, fmt.Errorf("未登录")
	}
	customer := models.UserAccount{ID: cusId.(int)}
	if err := customer.GetByID(); err != nil {
		return customer, err
	}
	if !customer.Enabled {
		return customer, fmt.Errorf("账号被禁用")
	}
	return customer, nil
}

func GetUserID(c *gin.Context) int {
	cusId, ok := c.Get("customer_id")
	if !ok {
		return 0
	}
	return cusId.(int)
}

// 获取 Header 中设备信息
type DeviceHeader struct {
	OpenUdid  string `json:"openudid"`
	Idfa      string `json:"idfa"`
	Idfv      string `json:"idfv"`
	OsType    string `json:"ostype"`
	OsVersion string `json:"osver"`
	Timestamp int64  `json:"timestamp"`
}

func readDeviceFromHeader(c *gin.Context) (DeviceHeader, error) {
	device := DeviceHeader{}
	userAgent := c.GetHeader("ua")
	if userAgent == "" {
		return device, fmt.Errorf("no device info in header")
	}
	body, err := lib.DESDecrypt(userAgent, lib.ApiDeskey)
	if err != nil {
		return device, fmt.Errorf("[Device] 3des decrypt failed: %s, ip: %s", err, c.ClientIP())
	}
	if err = json.Unmarshal(body, &device); err != nil {
		return device, fmt.Errorf("[Device] json unmarshal failed: %s, ip: %s", err, c.ClientIP())
	}

	// 校验时间戳
	tdiff := math.Abs(float64(time.Now().Unix() - device.Timestamp))
	if tdiff > 10 {
		return device, fmt.Errorf("[Device] bad timestamp diff: %f, ip: %s", tdiff, c.ClientIP())
	}
	return device, nil
}
