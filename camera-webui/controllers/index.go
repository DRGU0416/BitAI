package controllers

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"camera-webui/lib"
	"camera-webui/logger"
	"camera-webui/models"

	jsoniter "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
)

var (
	logApi = logger.New("logs/api.log")

	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

// 创建SD任务
func CreateSDTask(c echo.Context) error {
	sdtype := 0
	if c.QueryParam("type") != "" {
		var err error
		sdtype, err = strconv.Atoi(c.QueryParam("type"))
		if err != nil {
			logApi.Errorf("type转换失败, %s", err)
			return c.JSON(http.StatusOK, Response{FAILURE, "type转换失败"})
		}
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		logApi.Errorf("body读取失败, %s", err)
		return c.JSON(http.StatusOK, Response{FAILURE, "body读取失败"})
	}
	reqbody, err := lib.DESDecrypt(string(body), lib.WebUIDeskey)
	if err != nil {
		logApi.Errorf("3DES解密失败, %s", string(body))
		return c.JSON(http.StatusOK, Response{FAILURE, "解密失败"})
	}

	logApi.Debugf("%s", string(reqbody))
	task := &lib.Task{}
	if err = json.Unmarshal(reqbody, task); err != nil {
		logApi.Errorf("json解析失败, %s, %s", err, string(reqbody))
		return c.JSON(http.StatusOK, Response{FAILURE, "json解析失败"})
	}

	data, err := json.Marshal(task)
	if err != nil {
		logApi.Errorf("json序列化失败, %s", err)
		return c.JSON(http.StatusOK, Response{FAILURE, "json序列化失败"})
	}

	switch sdtype {
	case 0:
		sdword := &models.SDWork{
			ID:        task.TaskId,
			JsonData:  string(data),
			Status:    0,
			CreatedAt: time.Now().Unix(),
			Callback:  task.Callback,
		}
		err = sdword.Create()
	case 1:
		trainword := &models.TrainWork{
			ID:        task.TaskId,
			JsonData:  string(data),
			Status:    0,
			CreatedAt: time.Now().Unix(),
			Callback:  task.Callback,
		}
		err = trainword.Create()
	}
	if err != nil {
		logApi.Errorf("创建任务失败, %s", err)
		return c.JSON(http.StatusOK, Response{FAILURE, "任务创建失败"})
	}

	return c.JSON(http.StatusOK, Response{SUCCESS, ""})
}

// 创建检查任务
func CreateCheckTask(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		logApi.Errorf("body读取失败, %s", err)
		return c.JSON(http.StatusOK, Response{FAILURE, "body读取失败"})
	}
	reqbody := body
	if len(lib.WebUIDeskey) > 0 {
		reqbody, err = lib.DESDecrypt(string(body), lib.WebUIDeskey)
		if err != nil {
			logApi.Errorf("3DES解密失败, %s", string(body))
			return c.JSON(http.StatusOK, Response{FAILURE, "解密失败"})
		}
	}

	logApi.Debugf("%s", string(reqbody))
	task := &lib.TaskCheck{}
	if err = json.Unmarshal(reqbody, task); err != nil {
		logApi.Errorf("json解析失败, %s, %s", err, string(reqbody))
		return c.JSON(http.StatusOK, Response{FAILURE, "json解析失败"})
	}

	data, err := json.Marshal(task)
	if err != nil {
		logApi.Errorf("json序列化失败, %s", err)
		return c.JSON(http.StatusOK, Response{FAILURE, "json序列化失败"})
	}

	ckWork := &models.CheckWork{
		JsonData:  string(data),
		Status:    0,
		CreatedAt: time.Now().Unix(),
		Callback:  task.Callback,
	}
	_, err = ckWork.Create()
	if err != nil {
		logApi.Errorf("创建任务失败, %s", err)
		return c.JSON(http.StatusOK, Response{FAILURE, "任务创建失败"})
	}

	return c.JSON(http.StatusOK, Response{SUCCESS, ""})
}
