package cron

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"camera/lib"
	"camera/logger"
	"camera/models"

	jsoniter "github.com/json-iterator/go"
)

var (
	logOps = logger.New("logs/devops.log")

	noCDNFile = "no such file or directory"

	emptySleep = time.Second * 10
	errorSleep = time.Second * 60

	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

// 文件CDN
func FileCDN(ctx context.Context, ws *sync.WaitGroup) {
	defer ws.Done()
	wcs := new(sync.WaitGroup)
	ticker := time.NewTicker(time.Minute * 5)

	for {
		select {
		case <-ctx.Done():
			logOps.Debug("stop upload2cdn task")
			wcs.Wait()
			return
		case <-ticker.C:
			wcs.Add(1)
			runCDNTask(wcs)
		}
	}
}

func runCDNTask(wcs *sync.WaitGroup) {
	defer wcs.Done()

	for {
		task, err := lib.PopCDNTask()
		if err != nil {
			if err.Error() != lib.RedisNull {
				logOps.Errorf("[Redis] pop upload head2cdn error: %v", err)
			}
			break
		}
		logOps.Debugf("cdn task: %+v", task)

		switch task.TaskType {
		case lib.TRAIN_SIDE:
			runTrainSide(&task)
		case lib.CDN_DELETE:
			runCDNDelete(task.DelPath)
		}
	}
}

// 删除补充图
func runTrainSide(task *lib.UploadCDNTask) {
	customer := &models.UserAccount{ID: task.TaskId}
	if err := customer.GetByID(); err != nil {
		if err.Error() == models.NoRowError {
			return
		}
		logOps.Errorf("[Mysql] get customer error: %v", err)
		lib.PushCDNTask(task)
		time.Sleep(errorSleep)
		return
	}

	if customer.Step <= 2 {
		logOps.Warnf("customer: %d bad step: %d when remove image", customer.ID, customer.Step)
		return
	}

	// 删除图片
	imgpath := fmt.Sprintf("images/side/%s", customer.CardId)
	if err := os.RemoveAll(imgpath); err != nil {
		logOps.Errorf("[IO] remove image: %s failed: %s", imgpath, err)
		lib.PushCDNTask(task)
		time.Sleep(errorSleep)
		return
	}
}

// 删除CDN图片
func runCDNDelete(images []string) {
	for _, url := range images {
		if url == "" {
			continue
		}
		if !strings.HasPrefix(url, "http") {
			logOps.Warnf("invalid url: %s", url)
			continue
		}

		suburl := url[10:]
		err := lib.DeleteQNCDN(suburl[strings.Index(suburl, "/")+1:])
		if err != nil && err.Error() != noCDNFile {
			logOps.Errorf("[CDN] delete image: %s failed: %s", url, err)
		}
	}
}

// 处理头像
func runHeadCDN(task *lib.UploadCDNTask) {
	// 先上传，再删除
	// customer := &models.UserAccount{ID: task.TaskId}
	// if err := customer.GetByID(); err != nil {
	// 	logOps.Errorf("[Mysql] get customer error: %v", err)
	// 	return
	// }

	// if customer.Avatar != "" && !strings.HasPrefix(customer.Avatar, "http") {
	// 	fileName := filepath.Join("images", customer.Avatar)
	// 	f, err := os.Stat(fileName)
	// 	if err != nil || f.IsDir() {
	// 		logOps.Warnf("upload file error: %s, body: %+v", err, task)
	// 		return
	// 	}

	// 	// 上传
	// 	img, err := lib.GetImage(fileName)
	// 	if err != nil {
	// 		logOps.Errorf("[IO] get image failed: %s", err)
	// 		return
	// 	}

	// 	_, err = lib.UploadQNCDN(img, customer.Avatar)
	// 	if err != nil {
	// 		logOps.Errorf("[CDN] upload image failed: %s", err)
	// 		return
	// 	}
	// 	time.Sleep(time.Millisecond * 100)

	// 	customer.Avatar, _ = url.JoinPath(lib.QiniuHost, customer.Avatar)
	// 	if err = customer.UpdateAvatar(); err != nil {
	// 		logOps.Errorf("[Mysql] update avatar error: %v", err)
	// 		return
	// 	}
	// 	os.Remove(fileName)
	// }

	// // 删除
	// if task.DelPath != "" {
	// 	if strings.HasPrefix(task.DelPath, lib.QiniuHost) {
	// 		// 直接删除CDN
	// 		fileName := strings.Replace(task.DelPath, lib.QiniuHost, "", 1)
	// 		fileName = strings.TrimPrefix(fileName, "/")
	// 		if err := lib.DeleteQNCDN(fileName); err != nil {
	// 			logOps.Errorf("[CDN] delete image failed: %s", err)
	// 		}
	// 		time.Sleep(time.Millisecond * 100)
	// 		return
	// 	}

	// 	// 删本地文件
	// 	f, err := os.Stat(task.DelPath)
	// 	if err != nil || f.IsDir() {
	// 		logOps.Warnf("del file error: %s, body: %+v", err, task)
	// 		return
	// 	}
	// 	os.Remove(task.DelPath)
	// }
}
