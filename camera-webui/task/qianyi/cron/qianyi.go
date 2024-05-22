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
	logTask = logger.New("logs/qianyi.log")

	json = jsoniter.ConfigCompatibleWithStandardLibrary

	emptySleep = time.Second * 3
	errorSleep = time.Second * 30
)

func TaskService(ctx context.Context, ws *sync.WaitGroup) {
	defer ws.Done()
	wch := make(chan int, lib.WebUIThread)
	wcs := new(sync.WaitGroup)

	if err := models.ResetSDWork(); err != nil {
		logTask.Errorf("初始化SD任务失败, %s", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			logTask.Debug("stop service task")
			wcs.Wait()
			return
		case wch <- 1:
			wcs.Add(1)
			go RunTask(wch, wcs, nil)
		}
	}
}

func RunTask(wch chan int, wcs *sync.WaitGroup, testWork *models.SDWork) {
	defer func() {
		<-wch
		wcs.Done()
	}()
	fmt.Println("开始执行任务......")
	sdwork := &models.SDWork{}
	if testWork == nil {
		// 取任务
		if err := sdwork.GetWork(); err != nil && err.Error() != models.NoRowError {
			logTask.Errorf("获取任务失败, %s", err)
			time.Sleep(emptySleep)
			return
		}
		if sdwork.JsonData == "" {
			// 从Web获取任务
			task, err := lib.GetPhotoTask()
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
		sdwork = testWork
	}

	fmt.Println("获取到任务，进行解析......")
	task := lib.Task{}
	if err := json.Unmarshal([]byte(sdwork.JsonData), &task); err != nil {
		logTask.Errorf("json解析失败, %s, %s", err, sdwork.JsonData)
		taskFailed(sdwork, INVALID_PARAM, "json解析失败")
		return
	}

	// 下载图片
	var err error
	folderName := fmt.Sprintf("%d_%d", time.Now().Unix(), task.TaskId)
	basePath := filepath.Join(lib.WebUIWorkPath, folderName)
	controlNetPath := filepath.Join(lib.WebUIWorkPath, "controlNet")
	downloadPath := filepath.Join(basePath, "download")
	sdwork.TaskPath = basePath

	// 参数
	stype := task.Stype

	//下载子模型
	if stype.SubModelUrl != "" {
		name := filepath.Base(stype.SubModelUrl)
		dstLoraModelPath := filepath.Join(lib.WebUILoraPath, name)
		if !lib.FileExists(dstLoraModelPath) {
			//下载
			if err := lib.DownloadFile(stype.SubModelUrl, dstLoraModelPath); err != nil {
				logTask.Errorf("下载子模型: %s 失败, %s", stype.SubModelUrl, err)
				taskFailed(sdwork, INVALID_PARAM, "下载子模型失败")
				return
			}
		}
	}

	// 下载ControlNet底图
	for _, cnet := range stype.ControlNets {
		if cnet.ImagePath == "" {
			continue
		}

		cnPath := cnet.ImagePath
		if strings.HasPrefix(cnet.ImagePath, "http") {
			name := filepath.Base(cnet.ImagePath)
			cnPath = filepath.Join(controlNetPath, name)
			if !lib.FileExists(cnPath) {
				//下载
				if err := lib.DownloadFile(cnet.ImagePath, cnPath); err != nil {
					logTask.Errorf("下载底图: %s 失败, %s", cnet.ImagePath, err)
					taskFailed(sdwork, INVALID_PARAM, "下载底图失败")
					return
				}
			}
		}
		if exist := lib.FileExists(cnPath); !exist {
			logTask.Errorf("底图不存在: %s", cnPath)
			taskFailed(sdwork, INVALID_PARAM, "底图不存在")
			return
		}
		bgbase64Str, err := lib.ImageFileToBase64(cnPath)
		if err != nil {
			logTask.Errorf("底图转Base64: %s, 失败, %s", cnPath, err)
			taskFailed(sdwork, FAILURE, "底图转Base64失败")
			return
		}
		cnet.ImagePath = bgbase64Str
	}

	//下载roop换脸底图
	roop := stype.Roop
	if roop != nil && roop.ImagePath != "" {
		roopImgPath := roop.ImagePath
		if strings.HasPrefix(roop.ImagePath, "http") {
			name := filepath.Base(roop.ImagePath)
			roopImgPath = filepath.Join(downloadPath, name)
			//下载
			if err := lib.DownloadFile(roop.ImagePath, roopImgPath); err != nil {
				logTask.Errorf("下载roop底图: %s 失败, %s", roop.ImagePath, err)
				taskFailed(sdwork, INVALID_PARAM, "下载底图失败")
				return
			}
		}
		if exist := lib.FileExists(roopImgPath); !exist {
			logTask.Errorf("roop底图不存在: %s", roopImgPath)
			taskFailed(sdwork, INVALID_PARAM, "roop底图不存在")
			return
		}

		// 分身Roop底图，要截取脸部
		if task.TaskType == 1 {
			result, err := libsd.ClipFaces(downloadPath, downloadPath, false)
			if err != nil {
				logTask.Errorf("裁剪头像: %s 失败, %s", downloadPath, err)
			} else {
				if clips, ok := result[roopImgPath]; ok && len(clips) > 0 {
					roopImgPath = clips[0]
				}
			}
		}

		roopBase64Str, err := lib.ImageFileToBase64(roopImgPath)
		if err != nil {
			logTask.Errorf("roop底图转Base64: %s, 失败, %s", roopImgPath, err)
			taskFailed(sdwork, FAILURE, "roop底图转Base64失败")
			return
		}
		roop.ImagePath = roopBase64Str
	}

	//下载ADetailer模型
	for _, adetailer := range stype.ADetailer {
		if adetailer.ModelUrl != "" {
			name := filepath.Base(adetailer.ModelUrl)
			paths := strings.Split(adetailer.ModelUrl, "/")
			if len(paths) <= 2 {
				logTask.Errorf("ADetailer模型url解析: %s, 失败", adetailer.ModelUrl)
				taskFailed(sdwork, FAILURE, "ADetailer模型url解析失败")
				return
			}
			fn := paths[len(paths)-2]
			modelPath := filepath.Join(lib.WebUILoraSavePath, fn, name)
			if !lib.FileExists(modelPath) {
				//下载
				if err := lib.DownloadFile(adetailer.ModelUrl, modelPath); err != nil {
					logTask.Errorf("下载ADetailer模型: %s 失败, %s", adetailer.ModelUrl, err)
					taskFailed(sdwork, INVALID_PARAM, "下载ADetailer模型失败")
					return
				}
			}
			//拷贝模型至Lora使用目录
			dstLoraModelPath := filepath.Join(lib.WebUILoraPath, name)
			if !lib.FileExists(dstLoraModelPath) {
				if err := lib.CopyFile(modelPath, dstLoraModelPath, true); err != nil {
					logTask.Errorf("拷贝模型至Lora: %s, 失败, %s", modelPath, err)
					taskFailed(sdwork, FAILURE, "拷贝模型至Lora失败")
					return
				}
			}

			sdwork.ADModelPaths = append(sdwork.ADModelPaths, dstLoraModelPath)
		}
	}

	// 设置模型
	fmt.Println(stype.MainModelPath)
	options := libsd.SDOptions{
		SDModelCheckpoint: filepath.Base(stype.MainModelPath),
	}
	options.ApplySDOptions()

	//出图
	savePath := filepath.Join(basePath, "output_image")
	var images []string
	var seed int64

	tig, err := createText2img(task)
	if err != nil {
		taskFailed(sdwork, FAILURE, "出图失败")
		return
	}

	images, seed, err = tig.GenerateImages()
	if err != nil {
		logTask.Errorf("生成图像: %s, 失败, %s", basePath, err)
		taskFailed(sdwork, FAILURE, "生成图像失败")
		return
	}

	// 二次生成
	if task.SecondGeneration && len(images) > 0 {
		// 替换ControlNet底图
		if len(tig.ControlNetUnits) > 0 {
			cnts := make([]libsd.SDControlNetUnit, 0)
			for _, cnt := range tig.ControlNetUnits {
				cnt.InputImage = images[0]
				cnts = append(cnts, cnt)
			}
			tig.ControlNetUnits = cnts
		}

		//关闭Roop
		tig.RoopUnit = libsd.RoopUnit{}

		// 二次生成
		images, seed, err = tig.GenerateImages()
		if err != nil {
			logTask.Errorf("生成图像: %s, 失败, %s", basePath, err)
			taskFailed(sdwork, FAILURE, "生成图像失败")
			return
		}
	}

	os.MkdirAll(savePath, 0644)

	urls, wurls := make([]string, 0), make([]string, 0)
	for i, imgb64 := range images {
		if i >= stype.BatchSize {
			break
		}

		imgName, waterImgName := fmt.Sprintf("%s.png", lib.GenGUID()), fmt.Sprintf("%s.png", lib.GenGUID())

		imgPath := filepath.Join(basePath, imgName)
		if err = lib.Base64ToPNG(imgb64, imgPath); err != nil {
			logTask.Errorf("保存图像失败, %s", err)
			taskFailed(sdwork, FAILURE, "保存图像失败")
			return
		}

		// 上传CDN
		imgKey := fmt.Sprintf("cphoto/%s/%s/%s", imgName[:2], imgName[2:4], imgName[4:])
		if _, err = lib.UploadQNCDN(imgPath, imgKey); err != nil {
			logTask.Errorf("上传CDN失败, %s", err)
			taskFailed(sdwork, FAILURE, "上传CDN失败")
			return
		}

		imgUrl, err := url.JoinPath(lib.QiniuHost, imgKey)
		if err != nil {
			logTask.Errorf("图片Url合并失败, %s", err)
			taskFailed(sdwork, FAILURE, "图片Url合并失败")
			return
		}
		urls = append(urls, imgUrl)

		// 添加水印
		waterPath := filepath.Join(basePath, waterImgName)
		if err = libsd.AddWatermark(imgPath, waterPath); err != nil {
			logTask.Errorf("添加水印失败, %s", err)
		} else {
			// 上传CDN
			waterKey := fmt.Sprintf("cphoto/%s/%s/%s", waterImgName[:2], waterImgName[2:4], waterImgName[4:])
			if _, err = lib.UploadQNCDN(waterPath, waterKey); err == nil {
				if waterUrl, err := url.JoinPath(lib.QiniuHost, waterKey); err == nil {
					wurls = append(wurls, waterUrl)
				}
			}
		}
		if len(wurls) == 0 {
			wurls = append(wurls, "")
		}
	}

	// 删除出图目录
	os.RemoveAll(basePath)

	if len(urls) == 0 {
		taskFailed(sdwork, FAILURE, "生成图像失败")
		return
	}
	taskSuccess(sdwork, urls, wurls, seed)
}

// 创建生图器
func createText2img(task lib.Task) (libsd.SDTextToImageGenerator, error) {
	tig := libsd.SDTextToImageGenerator{
		DenoisingStrength: task.Stype.DenoisingStrength,
		Prompt:            task.Stype.Prompt,

		NegativePrompt: task.Stype.NegativePrompt,
		SamplerName:    task.Stype.SamplerName,
		Width:          task.Stype.Width,
		Height:         task.Stype.Height,
		Seed:           task.Stype.Seed,
		Steps:          task.Stype.Steps,
		CfgScale:       task.Stype.CfgScale,
		RestoreFaces:   task.Stype.RestoreFace,

		NIter:      1,
		BatchSize:  task.Stype.BatchSize,
		BatchCount: task.Stype.BatchCount,
		Tiling:     task.Stype.Tiling,

		SubSeed:                           -1,
		SeedResizeFromH:                   -1,
		SeedResizeFromW:                   -1,
		SNoise:                            1.0,
		SamplerIndex:                      "Euler",
		SendImages:                        true,
		SaveImages:                        false,
		OverrideSettingsRestoreAfterwards: true,
		ScriptArgs:                        make([]string, 0),
		OverrideSettings:                  make(map[string]any),
		Styles:                            make([]string, 0),
	}
	//加入高清放大设置
	if task.Stype.EnableHr {
		tig.HrScale = task.Stype.HrScale
		tig.EnableHr = task.Stype.EnableHr
		tig.HrUpscaler = task.Stype.HiresUpscaler
		tig.HrSecondPassSteps = task.Stype.HrSecondPassSteps
		tig.DenoisingStrength = task.Stype.DenoisingStrength
	}
	//加入ControlNet设置
	for _, cnet := range task.Stype.ControlNets {
		unit := libsd.SDControlNetUnit{
			Weight:        cnet.Weight,
			Model:         cnet.ModelName,
			Module:        cnet.Preprocessor,
			Enabled:       true,
			PixelPerfect:  cnet.PixelPerfect,
			ProcessorRes:  cnet.PreprocRes,
			GuidanceStart: cnet.StartCtrlStep,
			GuidanceEnd:   cnet.EndCtrlStep,
			ControlMode:   cnet.ControlMode,
			ResizeMode:    cnet.ResizeMode,
			InputImage:    cnet.ImagePath,
			ThresholdA:    64,
			ThresholdB:    64,
		}
		if unit.Module == "canny" {
			unit.ThresholdA = 100
			unit.ThresholdB = 200
		}
		tig.ControlNetUnits = append(tig.ControlNetUnits, unit)
	}
	//加入roop设置
	if task.Stype.Roop != nil && len(task.Stype.Roop.ImagePath) > 0 {
		unit := libsd.RoopUnit{
			ImgBase64:              task.Stype.Roop.ImagePath,
			Enabled:                true,
			FacesIndex:             "0",
			FaceRestorerName:       task.Stype.Roop.FaceRestorerName,
			FaceRestorerVisibility: task.Stype.Roop.FaceRestorerVisibility,
			UpscalerName:           "None",
			UpscalerScale:          1,
			UpscalerVisibility:     1,
			SwapInSource:           false,
			SwapInGenerated:        true,
			Model:                  lib.WebUIRoopModelPath,
		}
		tig.RoopUnit = unit
	}

	//加入ADetailer设置
	for _, adetailer := range task.Stype.ADetailer {
		unit := libsd.ADetailerUnit{
			AdModel:                    adetailer.AdModel,
			AdPrompt:                   adetailer.AdPrompt,
			AdNegativePrompt:           adetailer.AdNegativePrompt,
			AdConfidence:               adetailer.AdConfidence,
			AdMaskMinRatio:             0,
			AdMaskMaxRatio:             1,
			AdXOffset:                  0,
			AdYOffset:                  0,
			AdDilateErode:              adetailer.AdDilateErode,
			AdMaskMergeInvert:          "None",
			AdMaskBlur:                 20,
			AdDenoisingStrength:        adetailer.AdDenoisingStrength,
			AdInpaintOnlyMasked:        true,
			AdInpaintOnlyMaskedPadding: 40,
			AdUseInpaintWidthHeight:    false,
			AdInpaintWidth:             adetailer.AdInpaintWidth,
			AdInpaintHeight:            adetailer.AdInpaintHeight,
			AdUseSteps:                 false,
			AdSteps:                    28,
			AdUseCfgScale:              false,
			AdCfgScale:                 7,
			AdRestoreFace:              false,
			AdControlnetModel:          "None",
			AdControlnetWeight:         1,
			AdControlnetGuidanceStart:  0,
			AdControlnetGuidanceEnd:    1,
			AdUseClipSkip:              true,
			AdClipSkip:                 2,
		}
		tig.ADetailerUnits = append(tig.ADetailerUnits, unit)
	}

	tig.OverrideSettings = make(map[string]any)
	tig.OverrideSettings["token_merging_ratio"] = 0.4
	tig.OverrideSettings["token_merging_ratio_hr"] = 0.4
	tig.OverrideSettings["CLIP_stop_at_last_layers"] = 2
	tig.OverrideSettings["sd_model_checkpoint"] = filepath.Base(task.Stype.MainModelPath)

	//加入override_settings设置
	if task.Stype.RandnSource != "" {
		tig.OverrideSettings["randn_source"] = "CPU"
	}

	return tig, nil
}

// // 判断字符串数组是否包含某个字符串
// func containsString(strs []string, str string) bool {
// 	for _, v := range strs {
// 		if v == str {
// 			return true
// 		}
// 	}
// 	return false
// }
