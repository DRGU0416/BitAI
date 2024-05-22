package cron

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"camera-webui/lib"
	"camera-webui/libsd"
	"camera-webui/logger"

	jsoniter "github.com/json-iterator/go"
)

var (
	logTask = logger.New("logs/photohr.log")

	json = jsoniter.ConfigCompatibleWithStandardLibrary

	emptySleep = time.Second * 3
	errorSleep = time.Second * 30
)

func PhotoHrService(ctx context.Context, ws *sync.WaitGroup) {
	defer ws.Done()

	for {
		select {
		case <-ctx.Done():
			logTask.Debug("stop photo hr service")
			return
		default:
			RunPhotoHr()
		}
	}
}

func RunPhotoHr() {
	fmt.Println("开始执行高清任务......")

	// 从Web获取任务
	task, err := lib.GetPhotoHrTask()
	if err != nil {
		logTask.Errorf("获取WEB任务失败, %s", err)
		time.Sleep(errorSleep)
		return
	}
	if task.Callback == "" {
		time.Sleep(emptySleep)
		return
	}

	fmt.Println("获取到任务，进行解析......")

	// 下载图片
	folderName := fmt.Sprintf("%d_%d", time.Now().Unix(), task.TaskId)
	fileExt := filepath.Ext(task.ImageUrl)
	basePath := filepath.Join(lib.WebUIPhotoHrPath, folderName)
	downloadPath := filepath.Join(basePath, "0"+fileExt)
	if err := lib.DownloadFile(task.ImageUrl, downloadPath); err != nil {
		logTask.Errorf("下载图片失败, %s url=%s", err, task.ImageUrl)
		checkFailed(task, INVALID_PARAM, "下载原图失败")
		return
	}

	// 高清处理
	hrPath := filepath.Join(basePath, "hr"+fileExt)
	if err = libsd.ImageHires(downloadPath, hrPath); err != nil {
		logTask.Errorf("高清处理: %s 失败, %s", task.ImageUrl, err)
		checkFailed(task, FAILURE, "高清处理失败")
		return
	}
	// 上传CDN
	key := lib.GenGUID()
	hrKey := fmt.Sprintf("cphoto/%s/%s/%s.png", key[:2], key[2:4], key[4:])
	if _, err = lib.UploadQNCDN(hrPath, hrKey); err != nil {
		logTask.Errorf("上传CDN失败, %s", err)
		checkFailed(task, FAILURE, "上传CDN失败")
		return
	}
	hrUrl, err := url.JoinPath(lib.QiniuHost, hrKey)
	if err != nil {
		logTask.Errorf("url拼接失败: %s", err)
		checkFailed(task, FAILURE, "上传CDN失败")
		return
	}

	// 添加水印
	waterUrl := ""
	waterFileName := filepath.Join(basePath, "water"+fileExt)
	if err = libsd.AddWatermark(hrPath, waterFileName); err == nil {
		// 上传CDN
		waterKey := fmt.Sprintf("cphoto/%s/%s/%s.png", key[:2], key[2:4], key[4:26])
		if _, err = lib.UploadQNCDN(waterFileName, waterKey); err == nil {
			waterUrl, _ = url.JoinPath(lib.QiniuHost, waterKey)
		}
	}

	// 删除目录
	os.RemoveAll(basePath)

	checkSuccess(task, hrUrl, waterUrl)
}
