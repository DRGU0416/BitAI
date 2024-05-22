package cron

import (
	"context"
	"fmt"
	"net/url"
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
)

var (
	logTask = logger.New("logs/task.log")

	json = jsoniter.ConfigCompatibleWithStandardLibrary

	emptySleep = time.Second * 3
	errorSleep = time.Second * 30
)

func TrainService(ctx context.Context, ws *sync.WaitGroup) {
	defer ws.Done()

	if err := models.ResetTrainWork(); err != nil {
		logTask.Errorf("初始化Train任务失败, %s", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			logTask.Debug("stop train service")
			return
		default:
			RunTrain(ctx, nil)
		}
	}
}

func RunTrain(ctx context.Context, tempWork *models.TrainWork) {
	defer func() {
		if e := recover(); e != nil {
			logTask.Errorf("runTrain recover success. err: %s", e)
		}
	}()

	fmt.Println("开始执行train任务......")
	sdwork := &models.TrainWork{}
	if tempWork == nil {
		// 取任务
		if err := sdwork.GetWork(); err != nil && err.Error() != models.NoRowError {
			logTask.Errorf("获取任务失败, %s", err)
			time.Sleep(emptySleep)
			return
		}
		if sdwork.JsonData == "" {
			// 从Web获取任务
			task, err := lib.GetTrainTask()
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

			sdwork.ID = task.TaskId
			sdwork.JsonData = string(data)
			sdwork.Status = 0
			sdwork.CreatedAt = time.Now().Unix()
			sdwork.Callback = task.Callback
			err = sdwork.Create()
			if err != nil {
				logTask.Errorf("创建任务失败, %s", err)
				time.Sleep(errorSleep)
				return
			}
		}
	} else {
		sdwork = tempWork
	}

	fmt.Println("获取到任务，进行解析......")
	task := lib.Task{}
	if err := json.Unmarshal([]byte(sdwork.JsonData), &task); err != nil {
		logTask.Errorf("json解析失败, %s, %s", err, sdwork.JsonData)
		trainFailed(sdwork, INVALID_PARAM, "json解析失败")
		return
	}

	folderName := fmt.Sprintf("%d_%d", time.Now().Unix(), task.TaskId)
	basePath := filepath.Join(lib.WebUITrainPath, folderName)
	sdwork.TaskPath = basePath
	clipPath := filepath.Join(basePath, "clip")
	os.MkdirAll(clipPath, os.ModePerm)

	// 下载图片
	downloadPath := filepath.Join(basePath, "download")
	success := 0
	for _, imgurl := range task.LoraTrain.ImageUrl {
		bimgFullName := filepath.Join(downloadPath, filepath.Base(imgurl))
		if err := lib.DownloadFile(imgurl, bimgFullName); err != nil {
			break
		}
		success++
	}
	if success != len(task.LoraTrain.ImageUrl) {
		trainFailed(sdwork, INVALID_PARAM, "下载图片失败")
		return
	}

	//裁剪头像
	result, err := libsd.ClipFaces(downloadPath, clipPath, false)
	if err != nil {
		logTask.Errorf("裁剪头像: %s 失败, %s", downloadPath, err)
		trainFailed(sdwork, FAILURE, "裁剪头像失败")
		return
	}

	//检查生成图片存不存在
	allOk := true
	for _, imgurl := range task.LoraTrain.ImageUrl {
		bimgFullName := filepath.Join(downloadPath, filepath.Base(imgurl))
		clipResult, ok := result[bimgFullName]
		if !ok || len(clipResult) == 0 {
			allOk = false
			break
		}
	}

	if !allOk {
		logTask.Errorf("裁剪头像: 失败, 图片未裁出")
		trainFailed(sdwork, FAILURE, "裁剪头像失败, 图片未裁出")
		return
	}

	// 识别性别
	gfilepaths, err := filepath.Glob(filepath.Join(clipPath, "*"))
	if err != nil {
		logTask.Errorf("读取文件夹: %s, 失败:%s", clipPath, err)
		trainFailed(sdwork, INVALID_PARAM, "获取文件夹图片失败")
		return
	}
	gender := ""
	for _, path := range gfilepaths {
		if !lib.IsImageFile(path) {
			continue
		}
		gender, _ = getGenderFromTag(path)
		if gender != "" {
			break
		}
	}
	if gender == "" {
		logTask.Errorf("性别提取失败: %s", clipPath)
		trainFailed(sdwork, INVALID_PARAM, "性别提取失败")
		return
	}

	// 人脸去背景
	noBGPath := filepath.Join(basePath, "nobackground")
	if err := libsd.BatchRemoveBackground(clipPath, noBGPath, false); err != nil {
		logTask.Errorf("去背景: %s 失败, %s", clipPath, err)
		trainFailed(sdwork, FAILURE, "去背景失败")
		return
	}

	// 图片统一规格
	baseModePath := task.LoraTrain.BaseModel
	trainPath := filepath.Join(basePath, "train")
	trainImagePath := filepath.Join(trainPath, "images")
	formatPath := filepath.Join(trainImagePath, fmt.Sprintf("%d_%s", libsd.Loop, task.LoraTrain.UUID))
	os.MkdirAll(formatPath, os.ModePerm)
	trainLogPath := filepath.Join(trainPath, "logs")
	trainOutputPath := filepath.Join(trainPath, "output")
	if err := libsd.BatchResizeImages(noBGPath, 512, formatPath, false, true); err != nil {
		logTask.Errorf("格式化图片: %s 失败, %s", noBGPath, err)
		trainFailed(sdwork, FAILURE, "格式化图片失败")
		return
	}

	// 图片提取Tag
	filepaths, err := filepath.Glob(filepath.Join(formatPath, "*"))
	if err != nil {
		logTask.Errorf("读取文件夹: %s, 失败:%s", formatPath, err)
		trainFailed(sdwork, INVALID_PARAM, "获取文件夹图片失败")
		return
	}

	imageCnt := 0
	for _, path := range filepaths {
		if !lib.IsImageFile(path) {
			continue
		}
		imageCnt++
		tag, _ := getImageTags(path)
		tag = strings.ReplaceAll(tag, "girl", "woman")
		tag = strings.ReplaceAll(tag, "boy", "man")
		fileName := filepath.Base(path)
		idx := strings.LastIndex(fileName, ".")
		os.WriteFile(filepath.Join(formatPath, fmt.Sprintf("%s.txt", fileName[:idx])), []byte(task.LoraTrain.UUID+", "+tag), 0644)
	}

	// 训练模型
	modelName := fmt.Sprintf("camera_%d_%d", task.UserId, task.TaskId)
	trainer := libsd.SDLoraModelTrainer160{
		//常用配置
		BaseModel:       baseModePath,
		IsV2:            false,
		ImageFolder:     trainImagePath,
		OutputFolder:    trainOutputPath,
		LoggingFolder:   trainLogPath,
		ModelOutputName: modelName,
	}
	session := lib.GenGUID()
	if err = trainer.TrainLORAModel(session, imageCnt); err != nil {
		logTask.Errorf("训练模型: %s, 失败, %s", trainImagePath, err)
		trainFailed(sdwork, FAILURE, "训练模型失败")
		return
	}

	//取最后1个
	tensorPathLast := filepath.Join(trainOutputPath, modelName+".safetensors")

	waitChan := make(chan int, 1)
	ticker := time.NewTicker(2 * time.Second)
	startTime := time.Now()
	var modelSize int64 = 0
	go func() {
		for {
			select {
			case <-ctx.Done():
				waitChan <- 1
				return
			case <-ticker.C:
				if lib.FileExists(tensorPathLast) {
					size := lib.FileSize(tensorPathLast)
					if modelSize > 0 && size == modelSize {
						waitChan <- 1
						return
					}
					modelSize = size
				}
				if time.Since(startTime) > time.Duration(lib.MaxWaitMinute)*time.Minute {
					waitChan <- 1
					return
				}
			}
		}
	}()
	<-waitChan

	if !lib.FileExists(tensorPathLast) {
		logTask.Errorf("训练模型结果不存在: %s", modelName)
		libsd.CancelTrainTask(session)
		trainFailed(sdwork, FAILURE, "训练模型结果不存在")
		return
	}
	fn := task.UserId % lib.FolderCount
	savePath := filepath.Join(lib.WebUILoraSavePath, fmt.Sprintf("%d", fn))
	os.MkdirAll(savePath, os.ModePerm)
	absPathLast := filepath.Join(savePath, fmt.Sprintf("%s.safetensors", modelName))

	// 上传CDN
	// if _, err := lib.UploadQNCDN(absPathLast, fmt.Sprintf("lora/%d/%s", fn, filepath.Base(absPathLast))); err != nil {
	// 	logTask.Errorf("模型2上传CDN: %s, 失败:%s", absPathLast, err)
	// 	trainFailed(sdwork, FAILURE, "模型2上传CDN失败")
	// 	return
	// }
	//we
	os.Rename(tensorPathLast, absPathLast)
	modelUrl2, err := url.JoinPath(lib.QiniuHost, fmt.Sprintf("lora/%d/%s", fn, filepath.Base(absPathLast)))
	if err != nil {
		logTask.Errorf("模型2路径拼接: %s, 失败:%s", absPathLast, err)
		trainFailed(sdwork, FAILURE, "模型2路径拼接失败")
		return
	}

	// 删除训练目录
	// os.RemoveAll(basePath)

	// 回调
	loras := make([]lib.LoraModel, 4)
	loras[0] = lib.LoraModel{LoraPath: modelUrl2, Weight: 0.8, PromptWeight: 0.7, SecondGeneration: false}
	loras[1] = lib.LoraModel{LoraPath: modelUrl2, Weight: 0.85, PromptWeight: 0.7, SecondGeneration: false}
	loras[2] = lib.LoraModel{LoraPath: modelUrl2, Weight: 0.8, PromptWeight: 0.7, SecondGeneration: true}
	loras[3] = lib.LoraModel{LoraPath: modelUrl2, Weight: 0.85, PromptWeight: 0.7, SecondGeneration: true}

	if gender == "woman" {
		trainSuccess(sdwork, loras, 1)
	} else {
		trainSuccess(sdwork, loras, 2)
	}
}

// 获取图片Tag
func getImageTags(path string) (string, error) {
	base64Str, err := lib.ImageFileToBase64(path)
	if err != nil {
		return "", err
	}
	images := libsd.SDImageTagsGenerator{
		Image: base64Str,
	}
	tags, err := images.GenerateImageTags()
	if err != nil {
		return "", err
	}
	var tagstr strings.Builder
	for tag := range tags {
		if tagstr.Len() > 0 {
			tagstr.WriteString(", ")
		}
		tagstr.WriteString(tag)
	}
	return tagstr.String(), nil
}

// 根据Tag提取性别
func getGenderFromTag(path string) (string, error) {
	base64Str, err := lib.ImageFileToBase64(path)
	if err != nil {
		return "", err
	}
	images := libsd.SDImageTagsGenerator{
		Image: base64Str,
	}
	tags, err := images.GenerateImageTags()
	if err != nil {
		return "", err
	}

	gender := ""
	for tag := range tags {
		if strings.Index(tag, "girl") > -1 {
			return "woman", nil
		}
		if strings.Index(tag, "boy") > -1 {
			return "man", nil
		}
	}
	return gender, nil
}
