package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"camera/lib"
	"camera/models"

	"github.com/gin-gonic/gin"
)

// SendMessage 发送短信验证码
func SendMessage(c *gin.Context) {
	phoneByte, err := lib.DESDecrypt(c.Query("p"), lib.ApiDeskey)
	if err != nil {
		logApi.Warnf("[3DES] decrypt failed: %s, param: %s", err, c.Query("p"))
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}
	phone := string(phoneByte)
	ip := c.ClientIP()

	if !lib.VerifyMobileFormat(phone) {
		c.JSON(http.StatusOK, Response{FAILURE, "手机号格式不正确"})
		return
	}

	// 短信60秒内只能发送一次
	redisPhoneKey := fmt.Sprintf(lib.RedisMessagePhone, phone)
	success, err := lib.RDB.SetNX(ctx, redisPhoneKey, time.Now().Unix(), time.Minute).Result()
	if err != nil {
		logApi.Errorf("[Redis] setnx value failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "操作失败，请重试"})
		return
	}

	if !success {
		timestamp, _ := lib.RDB.Get(ctx, redisPhoneKey).Result()
		tv, _ := strconv.ParseInt(timestamp, 10, 64)
		msec := time.Now().Unix() - tv
		if msec < 60 {
			c.JSON(http.StatusOK, Response{MESSAGE_IS_SEND, 60 - msec})
			return
		}
	}

	// 同一个IP每天最多100条短信
	redisIPKey := fmt.Sprintf(lib.RedisMessageIP, time.Now().Format("0102"), ip)
	mcount, _ := lib.RDB.Incr(ctx, redisIPKey).Result()
	if end, err := lib.DayEnd(time.Now()); err == nil {
		lib.RDB.ExpireAt(ctx, redisIPKey, end)
	} else {
		lib.RDB.ExpireAt(ctx, redisIPKey, time.Now().Add(time.Hour*24))
	}
	if mcount > 100 {
		c.JSON(http.StatusOK, Response{FAILURE, "操作失败，请重试"})
		return
	}

	// 发送短信,有效期5分钟
	vercode := strconv.Itoa(lib.GetRand(100000, 999999))
	if err = lib.RDB.SetEX(ctx, fmt.Sprintf(lib.RedisMessageValue, phone), vercode, time.Minute*5).Err(); err != nil {
		logApi.Errorf("[Redis] set value expire failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "操作失败，请重试"})
		return
	}
	if err = lib.SendPhoneMessage(phone, fmt.Sprintf(lib.MessageCode, vercode)); err != nil {
		logApi.Warnf("[SMS] send phone message failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "操作失败，请重试"})
		return
	}
	c.JSON(http.StatusOK, Response{SUCCESS, ""})
}

// Login 登录
// application/x-www-form-urlencoded
func Login(c *gin.Context) {
	code := c.Request.FormValue("code")
	phone := c.Request.FormValue("phone")

	if len(code) != 6 {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}
	if !lib.VerifyMobileFormat(phone) {
		c.JSON(http.StatusOK, Response{FAILURE, "手机号格式不正确"})
		return
	}
	var err error

	if code != lib.GlobalSmsCode {
		// 验证码
		yzcode, err := lib.GetMobileCode(phone)
		if err != nil {
			logApi.Errorf("[Redis] get value failed: %s", err)
			c.JSON(http.StatusOK, Response{FAILURE, "登录失败"})
			return
		}

		if yzcode != code || code == "" {
			c.JSON(http.StatusOK, Response{FAILURE, "验证码不正确"})
			return
		}
	}

	// 查询数据库
	isReg := false
	customer := &models.UserAccount{Mobile: phone}
	if err = customer.GetCustomerByPhone(); err != nil {
		if err.Error() == models.NoRowError {
			// 新增用户
			newUser := true
			if err = customer.IsMobileExist(); err == nil {
				newUser = false
			}
			cardId, err := lib.GenCardId()
			if err != nil {
				logApi.Errorf("%s", err)
				c.JSON(http.StatusOK, Response{FAILURE, "登录失败"})
				return
			}

			customer.CardId = fmt.Sprintf("%d", cardId)
			customer.CardNum = lib.ToCardStr(cardId)
			customer.NewUser = newUser
			customer.RegIp = c.ClientIP()
			customer.CreatedAt = time.Now()
			customer.AppVer = c.GetHeader("appver")
			customer.RegChannel = c.GetHeader("channel")
			customer.Platform = c.GetHeader("platform")
			device, err := readDeviceFromHeader(c)
			if err == nil {
				customer.Openudid = device.OpenUdid
				customer.OsVer = device.OsVersion
				customer.OsType = device.OsType
			} else {
				logApi.Warn(err)
			}

			isReg = true
			if err = customer.Create(); err != nil {
				logApi.Errorf("[Mysql] add customer by phone failed: %s, customer: %v", err, customer)
				c.JSON(http.StatusOK, Response{FAILURE, "登录失败"})
				return
			}
		} else {
			logApi.Errorf("[Mysql] get customer failed: %s, phone: %s", err, phone)
			c.JSON(http.StatusOK, Response{FAILURE, "登录失败"})
			return
		}
	} else {
		if !customer.Enabled {
			c.JSON(http.StatusOK, Response{FAILURE, "已禁用"})
			return
		}

		customer.LoginIp = c.ClientIP()
		customer.LoginChannel = c.GetHeader("channel")
		customer.AppVer = c.GetHeader("appver")
		if err = customer.UpdateLogin(); err != nil {
			logApi.Errorf("[Mysql] update customer failed: %s, id: %d, ip: %s, appver: %s", err, customer.ID, customer.LoginIp, customer.AppVer)
		}
	}

	tokenValue, err := lib.CreateToken(customer.ID)
	if err != nil {
		logApi.Errorf("[JWT] create jwt token failed: %s, phone: %s", err, customer.Mobile)
		c.JSON(http.StatusOK, Response{FAILURE, "登录失败"})
		return
	}

	// 组装Response
	data := make(map[string]interface{})
	data["token"] = tokenValue
	data["cardid"] = customer.CardId
	data["phone"] = customer.Mobile
	data["headicon"] = ""
	data["is_reg"] = isReg
	if len(customer.Avatar) > 0 {
		if strings.HasPrefix(customer.Avatar, "http") {
			data["headicon"] = customer.Avatar
		} else {
			data["headicon"] = lib.ImageHost + customer.Avatar
		}
	}

	c.JSON(http.StatusOK, Response{SUCCESS, data})
}

// 友盟 一键登录 获取手机号
func UmengOneClickTokenValidate(c *gin.Context) {
	token := c.Request.FormValue("token")
	verifyID := c.Request.FormValue("verid")
	if len(token) == 0 {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}

	phonenum, err := lib.UmengTokenValidate(token, verifyID)
	if err != nil {
		logApi.Errorf("check mobile token validate failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "请求失败"})
		return
	}

	if _, err = lib.SetMobileCodeEX(phonenum, token, time.Second*300); err != nil {
		logApi.Errorf("[Redis] set value expire failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "操作失败，请重试"})
		return
	}

	c.JSON(http.StatusOK, Response{SUCCESS, phonenum})
}
