package controllers

import (
	"camera/models"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	versionList map[string]models.AppVersion
)

func init() {
	//预加载产品
	if err := loadAppVersion(); err != nil {
		logApi.Fatal(err)
	}
}

// 加载版本列表
func loadAppVersion() error {
	versionList = make(map[string]models.AppVersion)
	version := &models.AppVersion{}
	list, err := version.GetList()
	if err != nil {
		return err
	}

	for _, ver := range list {
		v := models.AppVersion{
			ID:       ver.ID,
			BundleID: ver.BundleID,
			Version:  ver.Version,
			IsForce:  ver.IsForce,
			DownUrl:  ver.DownUrl,
			DownMd5:  ver.DownMd5,
		}
		versionList[ver.BundleID] = v
	}

	if len(versionList) == 0 {
		return fmt.Errorf("init version failed")
	}
	return nil
}

// 刷缓存
func CacheRefresh(c *gin.Context) {
	res := make(map[string]interface{}, 2)
	res["status"] = false
	res["message"] = "加载失败"

	// 刷缓存
	param := c.Query("task")
	switch param {
	case "product":
		if err := loadProduct(); err != nil {
			logApi.Warnf("reload product failed, error: %s", err)
			c.JSON(http.StatusOK, res)
			return
		} else {
			logApi.Info("reload product success")
		}
	case "version":
		if err := loadAppVersion(); err != nil {
			logApi.Warnf("reload version failed, error: %s", err)
			c.JSON(http.StatusOK, res)
			return
		} else {
			logApi.Info("reload version success")
		}
	default:
		logApi.Warnf("bad cache task %s", param)
		c.JSON(http.StatusOK, res)
		return
	}

	res["status"] = true
	res["message"] = ""
	c.JSON(http.StatusOK, res)
}

// 上报基础数据
// func ReportBaseApp(c *gin.Context) {
// body, err := io.ReadAll(c.Request().Body)
// if err != nil {
// 	return c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
// }
// bdata, err := lib.DESDecrypt(string(body), lib.ApiDeskey)
// if err != nil {
// 	logApi.Warnf("[base app data] des decrypt failed, body: %s", string(body))
// 	return c.JSON(http.StatusOK, Response{FAILURE, "参数错误"})
// }
// appinfo := BaseAppModel{}
// if err = json.Unmarshal(bdata, &appinfo); err != nil {
// 	logApi.Warnf("[base app data] json unmarshal failed: %s, body: %s", err, string(bdata))
// 	return c.JSON(http.StatusOK, Response{FAILURE, "参数错误"})
// }

// // 验证用户
// customer, err := GetUser(c)
// if err != nil {
// 	return c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
// }

// //验证时间戳
// now := time.Now()
// if math.Abs(float64(now.Unix()-appinfo.Timestamp)) > 10 {
// 	logApi.Warnf("[base app data] bad timestamp ")
// 	return c.JSON(http.StatusOK, Response{FAILURE, "参数错误"})
// }

// 	return c.JSON(http.StatusOK, Response{SUCCESS, ""})
// }

// 检查版本更新
func CheckVersion(c *gin.Context) {
	cusver := c.GetHeader("appver")
	bundleId := c.GetHeader("bundleid")

	retval := make(map[string]any)
	// 有新版本
	retval["hasnew"] = false
	// 强制更新
	retval["force"] = false
	retval["version"] = ""
	retval["down_url"] = ""
	retval["down_md5"] = ""

	if version, ok := versionList[bundleId]; ok {
		retval["hasnew"] = version.Version != cusver
		retval["force"] = version.IsForce
		retval["version"] = version.Version
		retval["down_url"] = version.DownUrl
		retval["down_md5"] = version.DownMd5
	}

	c.JSON(http.StatusOK, Response{SUCCESS, retval})
	return
}
