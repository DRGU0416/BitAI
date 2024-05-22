package cron

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"camera-webui/lib"
	"camera-webui/libsd"
	"camera-webui/logger"
	"camera-webui/models"

	jsoniter "github.com/json-iterator/go"
	"golang.org/x/time/rate"
)

var (
	logTask = logger.New("logs/check.log")

	json = jsoniter.ConfigCompatibleWithStandardLibrary

	emptySleep = time.Second * 3
	errorSleep = time.Second * 30
)

func CheckService(ctx context.Context, ws *sync.WaitGroup) {
	defer ws.Done()

	if err := models.ResetCheckWork(); err != nil {
		logTask.Errorf("初始化Check任务失败, %s", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			logTask.Debug("stop check service")
			return
		default:
			RunCheck(nil)
		}
	}
}

func RunCheck(testWork *models.CheckWork) {
	fmt.Println("开始执行check任务......")

	ckWork := &models.CheckWork{}
	if testWork == nil {
		// 取任务
		if err := ckWork.GetWork(); err != nil && err.Error() != models.NoRowError {
			logTask.Errorf("获取数据库任务失败, %s", err)
		}
		if ckWork.JsonData == "" {
			// 从Web获取任务
			task, err := lib.GetCheckTask()
			if err != nil {
				logTask.Errorf("获取WEB任务失败, %s", err)
				time.Sleep(errorSleep)
				return
			}
			if task.Callback == "" {
				time.Sleep(emptySleep)
				return
			}
			data, err := json.Marshal(task)
			if err != nil {
				logTask.Errorf("json序列化失败, %s", err)
				time.Sleep(errorSleep)
				return
			}

			ckWork.JsonData = string(data)
			ckWork.Status = 0
			ckWork.CreatedAt = time.Now().Unix()
			ckWork.Callback = task.Callback
			wid, err := ckWork.Create()
			if err != nil {
				logTask.Errorf("创建任务失败, %s", err)
				time.Sleep(errorSleep)
				return
			}
			ckWork.ID = uint(wid)
		}
	} else {
		ckWork = testWork
	}

	fmt.Println("获取到任务，进行解析......")
	task := lib.TaskCheck{}
	if err := json.Unmarshal([]byte(ckWork.JsonData), &task); err != nil {
		logTask.Errorf("json解析失败, %s, %s", err, ckWork.JsonData)
		checkFailed(ckWork, INVALID_PARAM, "json解析失败")
		return
	}

	// 下载图片
	folderName := fmt.Sprintf("%d_%s", time.Now().Unix(), lib.GenGUID())
	basePath := filepath.Join(lib.WebUICheckPath, folderName)
	downloadPath := filepath.Join(basePath, "download")
	clipPath := filepath.Join(basePath, "clip")
	ckWork.TaskPath = basePath

	// 正面参照图
	frontPath := ""
	if task.BaseImage != "" {
		frontPath = filepath.Join(basePath, fmt.Sprintf("0%s", filepath.Ext(task.BaseImage)))
		if err := lib.DownloadFile(task.BaseImage, frontPath); err != nil {
			logTask.Errorf("下载图片失败, %s url=%s", err, task.BaseImage)
			checkFailed(ckWork, INVALID_PARAM, "下载正面参考图失败")
			return
		}
		//裁剪头像
		frontPath = filepath.Join(basePath, "face_0_0.png")
		_, err := libsd.ClipFaces(basePath, basePath, false)
		if err != nil {
			logTask.Errorf("裁剪头像: %s 失败, %s", downloadPath, err)
			checkFailed(ckWork, FAILURE, "裁剪正面照失败")
			return
		}
	}

	// 待校验图片
	success := 0
	for _, imgurl := range task.ImagesMap {
		bimgFullName := filepath.Join(downloadPath, filepath.Base(imgurl))
		if err := lib.DownloadFile(imgurl, bimgFullName); err != nil {
			logTask.Errorf("下载图片失败, %s url=%s", err, imgurl)
			break
		}
		success++
	}
	if success != len(task.ImagesMap) {
		logTask.Errorf("下载图片失败, success=%d all=%d", success, len(task.ImagesMap))
		checkFailed(ckWork, INVALID_PARAM, "下载图片失败")
		return
	}

	//裁剪头像
	heads, err := libsd.ClipFaces(downloadPath, clipPath, false)
	if err != nil {
		logTask.Errorf("裁剪头像: %s 失败, %s", downloadPath, err)
		checkFailed(ckWork, FAILURE, "裁剪头像失败")
		return
	}
	for _, v := range heads {
		if len(v) > 1 {
			os.Remove(v[0])
		}
	}

	//检查非法图片
	filepaths, err := filepath.Glob(filepath.Join(clipPath, "*"))
	if err != nil {
		logTask.Errorf("读取文件夹: %s, 失败:%s", clipPath, err)
		checkFailed(ckWork, INVALID_PARAM, "获取文件夹图片失败")
		return
	}

	limiter := rate.NewLimiter(rate.Every(time.Second), lib.BaiduCheckQpslimit)
	wscheck := new(sync.WaitGroup)
	forbidChan := make(chan int, lib.BaiduCheckQpslimit)
	for _, path := range filepaths {
		wscheck.Add(1)
		go func(frontP, fileP string) {
			defer wscheck.Done()

			if err := limiter.Wait(context.Background()); err != nil {
				fmt.Printf("[%s] [EXCEPTION] wait err: %v", fileP, err)
			}
			forbidChan <- 1
			identifyImage(frontP, fileP)
			<-forbidChan
		}(frontPath, path)
	}
	wscheck.Wait()

	//检查生成图片存不存在
	status := make(map[uint64]int)
	allOk := true
	for id, imgurl := range task.ImagesMap {
		fname := filepath.Base(imgurl)
		bimgFullName := filepath.Join(clipPath, fmt.Sprintf("face_0_%s.png", fname[:strings.LastIndex(fname, ".")]))
		if !lib.FileExists(bimgFullName) {
			status[id] = 2
			allOk = false
		} else {
			status[id] = 1
		}
	}

	// 删除目录
	// os.RemoveAll(basePath)

	checkSuccess(ckWork, status, allOk)
}

// 判断字符串数组是否包含某个字符串
func containsString(strs []string, str string) bool {
	for _, v := range strs {
		if v == str {
			return true
		}
	}
	return false
}

// 识别图片
func identifyImage(frontPath, filepath string) {
	fmt.Printf("time:%s path:%s\n", time.Now().String(), filepath)

	tp, err := lib.BaiduImageCensor(lib.BaiduCheckApiKey, lib.BaiduCheckApiSecret, filepath)
	if err != nil {
		logTask.Errorf("图片非法检查: %s, 失败:%s", filepath, err)
		os.Remove(filepath)
		return
	}

	if tp != 1 {
		logTask.Errorf("图片非法检查: %s, 不合规:%d", filepath, tp)
		os.Remove(filepath)
		return
	}
	fmt.Printf("endtime: %s path:%s\n", time.Now().String(), filepath)

	// if frontPath != "" && lib.Bai.FileExists(frontPath) {
	// 	same, _ := libsd.IsSameFace(frontPath, filepath)
	// 	if !same {
	// 		os.Remove(filepath)
	// 		return
	// 	}
	// }
	// hit, _ := libsd.ImageIsIllegal(filepath)
	// if hit {
	// 	os.Remove(filepath)
	// 	return
	// }
	// if hit, _ = libsd.ImageIsInKnownFaces(filepath); hit {
	// 	os.Remove(filepath)
	// }
}
