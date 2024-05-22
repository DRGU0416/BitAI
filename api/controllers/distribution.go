package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"camera/lib"
	"camera/models"

	"github.com/gin-gonic/gin"
)

// 照片识别任务分发
func GetPhotoRecognizeTask(c *gin.Context) {
	ctype, ids, err := lib.PopSDCheckTask()
	if err != nil {
		if err.Error() != lib.RedisNull {
			logApi.Errorf("[Redis] pop sd check task error: %v", err)
		}
		c.JSON(http.StatusOK, Response{FAILURE, lib.TaskCheck{}})
		return
	}

	ctask := lib.TaskCheck{}
	arrs := make(map[uint64]string)
	switch ctype {
	case 1:
		// 正面照
		for _, id := range ids {
			// 加入维护
			record := &lib.TaskRecord{TaskType: lib.REC_FRONT, TaskID: id}
			if err = record.Set(); err != nil {
				logApi.Errorf("[Redis] set task record error: %v, %+v", err, record)
			}

			// 处理业务
			task := models.UserFrontImage{ID: id}
			if err := task.GetByID(); err != nil {
				logApi.Errorf("[Mysql] get front image: %d failed: %s", id, err)
				continue
			}

			imgurl, err := lib.GetImageUrl(task.ImgUrl)
			if err != nil {
				logApi.Errorf("[WebUI] get image url failed: %s", err)
				continue
			}
			arrs[uint64(task.ID)] = imgurl
		}
		ctask.ImagesMap = arrs
		ctask.Callback = fmt.Sprintf(lib.WebUICallbackRecognize, "front")
	case 2:
		// 侧面照
		cusId := 0
		for _, id := range ids {
			// 加入维护
			record := &lib.TaskRecord{TaskType: lib.REC_SIDE, TaskID: id}
			if err = record.Set(); err != nil {
				logApi.Errorf("[Redis] set task record error: %v, %+v", err, record)
			}

			// 处理业务
			task := models.UserInputImage{ID: id}
			if err := task.GetByID(); err != nil {
				logApi.Errorf("[Mysql] get side image: %d failed: %s", id, err)
				continue
			}
			if cusId == 0 {
				cusId = task.CusId
			}

			imgurl, err := lib.GetImageUrl(task.ImgUrl)
			if err != nil {
				logApi.Errorf("[WebUI] get image url failed: %s", err)
				continue
			}
			arrs[uint64(task.ID)] = imgurl
		}

		// 正面参考照
		front := &models.UserFrontImage{CusId: cusId}
		if err = front.GetByCusId(); err != nil {
			logApi.Errorf("[Mysql] get front image by cusid: %d failed: %s", cusId, err)
			c.JSON(http.StatusOK, Response{FAILURE, lib.TaskCheck{}})
			return
		}
		frontUrl, err := lib.GetImageUrl(front.ImgUrl)
		if err != nil {
			logApi.Errorf("bad front image: %s", front.ImgUrl)
			c.JSON(http.StatusOK, Response{FAILURE, lib.TaskCheck{}})
			return
		}

		ctask.BaseImage = frontUrl
		ctask.ImagesMap = arrs
		ctask.Callback = fmt.Sprintf(lib.WebUICallbackRecognize, "side")
	}
	if len(arrs) == 0 {
		c.JSON(http.StatusOK, Response{FAILURE, lib.TaskCheck{}})
		return
	}
	c.JSON(http.StatusOK, Response{SUCCESS, ctask})
}

// Lora模型训练
func GetLoraTask(c *gin.Context) {
	taskId, err := lib.PopSDTask()
	if err != nil {
		if err.Error() != lib.RedisNull {
			logApi.Errorf("[Redis] pop sd task error: %v", err)
		}
		c.JSON(http.StatusOK, Response{FAILURE, lib.Task{}})
		return
	}

	task := &models.UserCardTask{ID: taskId}
	if err = task.GetByID(); err != nil {
		if err.Error() == models.NoRowError {
			c.JSON(http.StatusOK, Response{FAILURE, lib.Task{}})
			return
		}
		logApi.Errorf("[Mysql] get train task: %d error: %v", taskId, err)
		lib.PushSDTask(taskId)
		c.JSON(http.StatusOK, Response{FAILURE, lib.Task{}})
		return
	}
	if task.Status > models.RUNNING {
		c.JSON(http.StatusOK, Response{FAILURE, lib.Task{}})
		return
	}

	// 检查用户
	customer := &models.UserAccount{ID: task.CusId}
	if err = customer.GetByID(); err != nil {
		if err.Error() == models.NoRowError {
			task.UpdateStatus(models.FAILED, "用户不存在")
			c.JSON(http.StatusOK, Response{FAILURE, lib.Task{}})
			return
		}
		logApi.Errorf("[Mysql] get customer: %d error: %v", task.CusId, err)
		lib.PushSDTask(taskId)
		c.JSON(http.StatusOK, Response{FAILURE, lib.Task{}})
		return
	}
	if !customer.Enabled {
		task.UpdateStatus(models.FAILED, "用户被禁用")
		c.JSON(http.StatusOK, Response{FAILURE, lib.Task{}})
		return
	}

	// 检查正面照
	lora := lib.LoraTrain{
		BaseModel: lib.SDBaseModel,
		UUID:      customer.CardNum,
	}
	lora.ImageUrl = []string{task.FrontUrl}

	// 检查补充照
	input := &models.UserInputImage{CusId: task.CusId}
	images, err := input.GetByCusId()
	if err != nil {
		logApi.Errorf("[Mysql] get pass input images error: %v", err)
		lib.PushSDTask(taskId)
		c.JSON(http.StatusOK, Response{FAILURE, lib.Task{}})
		return
	}
	imgct := 0
	for _, image := range images {
		if image.Status != 2 {
			continue
		}
		if imgurl, err := lib.GetImageUrl(image.ImgUrl); err == nil {
			lora.ImageUrl = append(lora.ImageUrl, imgurl)
			imgct++
			if imgct >= 9 {
				break
			}
		}
	}
	if len(lora.ImageUrl) < 10 {
		logApi.Warnf("taskId: %d image < 10", taskId)
		task.UpdateStatus(models.FAILED, "训练照不足10张")
		c.JSON(http.StatusOK, Response{FAILURE, lib.Task{}})
		return
	}

	if err = task.UpdateStatus(models.RUNNING, ""); err != nil {
		logApi.Errorf("[Mysql] update running status: %d failed: %s", task.ID, err)
	}

	record := &lib.TaskRecord{TaskType: lib.REC_LORA, TaskID: taskId}
	record.Set()

	// 任务
	webuiTask := lib.Task{
		TaskId:    uint(taskId),
		Callback:  lib.WebUICallback,
		LoraTrain: lora,
		UserId:    task.CusId,
	}
	c.JSON(http.StatusOK, Response{SUCCESS, webuiTask})
}

// 写真
func GetPhotoTask(c *gin.Context) {
	task := getCardTask(c)
	if task.TaskId > 0 {
		record := &lib.TaskRecord{TaskType: lib.REC_CARD, TaskID: int(task.TaskId)}
		record.Set()

		c.JSON(http.StatusOK, Response{SUCCESS, task})
		return
	}

	task = getPhotoTask(c)
	if task.TaskId > 0 {
		record := &lib.TaskRecord{TaskType: lib.REC_PHOTO, TaskID: int(task.TaskId)}
		record.Set()

		c.JSON(http.StatusOK, Response{SUCCESS, task})
		return
	}
	c.JSON(http.StatusOK, Response{FAILURE, lib.Task{}})
}

// 分身任务
func getCardTask(c *gin.Context) lib.Task {
	webuiTask := lib.Task{}

	taskId, err := lib.PopSDCardTask()
	if err != nil {
		if err.Error() != lib.RedisNull {
			logApi.Errorf("[Redis] pop card task error: %v", err)
		}
		return webuiTask
	}

	task := &models.UserCardImage{ID: taskId}
	if err = task.GetByID(); err != nil {
		logApi.Errorf("[Mysql] get card task: %d error: %v", taskId, err)
		lib.PushSDCardTask(taskId)
		return webuiTask
	}
	if task.ImgUrl != "" {
		return webuiTask
	}

	itask := &models.UserCardTask{ID: task.TaskId}
	if err = itask.GetByID(); err != nil {
		logApi.Errorf("[Mysql] get card task: %d error: %v", taskId, err)
		lib.PushSDCardTask(taskId)
		return webuiTask
	}

	// 获取正面照
	front := &models.UserFrontImage{CusId: task.CusId}
	if err = front.GetByCusId(); err != nil {
		logApi.Errorf("[Mysql] get front image failed: %s", err)
		lib.PushSDCardTask(taskId)
		return webuiTask
	}
	frontUrl, err := lib.GetImageUrl(front.ImgUrl)
	if err != nil {
		logApi.Errorf("bad front image: %s", front.ImgUrl)
		lib.PushSDCardTask(taskId)
		return webuiTask
	}
	customer := &models.UserAccount{ID: front.CusId}
	if err = customer.GetByID(); err != nil {
		if err.Error() == models.NoRowError {
			logApi.Warnf("[Mysql] get customer: %d not exist", front.CusId)
			return webuiTask
		}
		logApi.Errorf("[Mysql] get customer: %d error: %v", front.CusId, err)
		lib.PushSDCardTask(taskId)
		return webuiTask
	}

	// ADetailer(自己训练的Lora模型)
	start := strings.LastIndex(task.Lora, "/")
	end := strings.LastIndex(task.Lora, ".")
	adetailer := &lib.ADetailer{
		AdModel:             "mediapipe_face_full",
		ModelUrl:            task.Lora,
		AdPrompt:            fmt.Sprintf("%s, <lora:%s:%.2f> masterpiece, best quality", customer.CardNum, task.Lora[start+1:end], task.Weight),
		AdNegativePrompt:    "verybadimagenegative_v1.3",
		AdInpaintWidth:      512,
		AdInpaintHeight:     512,
		AdDenoisingStrength: 0.2,
		AdConfidence:        0.35,
		AdDilateErode:       80,
	}
	if itask.Gender == 1 {
		adetailer.AdPrompt += ", a woman makeup"
	}

	// ControlNet(造型底图)
	ctrlNet := &lib.ControlNet{
		ImagePath:     getImageByGender(itask.Gender),
		ModelName:     "control_v11p_sd15_lineart [43d4be0d]",
		Preprocessor:  "lineart_realistic",
		Weight:        0.45,
		StartCtrlStep: 0,
		EndCtrlStep:   0.3,
		PreprocRes:    512,
		ControlMode:   "Balanced",
		ResizeMode:    "Crop and Resize",
		PixelPerfect:  true,
	}

	// 风格
	stype := lib.Stype{
		Width:          512,
		Height:         512,
		Prompt:         getPromptByGender(itask.Gender, customer.CardNum, task.Lora[start+1:end], task.PromptWeight),
		NegativePrompt: "(worst quality, low quality, illustration, 3d, 2d, painting, cartoons, sketch), tooth, open mouth,hands,fingers,nsfw verybadimagenegative_v1.3",
		SamplerName:    "DPM++ 2M SDE Karras",
		Steps:          20,
		RestoreFace:    true,
		Tiling:         false,
		CfgScale:       7,
		Seed:           -1,
		BatchCount:     1,
		BatchSize:      1,
		MainModelPath:  `E:\sdwebui\stable-diffusion-webui\models\Stable-diffusion\OnlyRealistic_v30.safetensors`,
		SubModelUrl:    "",

		EnableHr:          false,
		HiresUpscaler:     "4x-UltraSharp",
		HrSecondPassSteps: 20,
		HrScale:           2,
		DenoisingStrength: 0.1,

		ControlNets: []*lib.ControlNet{ctrlNet},
		Roop:        &lib.Roop{ImagePath: frontUrl, FaceRestorerVisibility: 0.9, FaceRestorerName: "CodeFormer"}, // 用户正面照
		ADetailer:   []*lib.ADetailer{adetailer},
	}

	// 任务
	webuiTask.TaskType = 1
	webuiTask.TaskId = uint(taskId)
	webuiTask.Callback = lib.WebUICallbackCard
	webuiTask.UserId = task.CusId
	webuiTask.Stype = stype
	webuiTask.SecondGeneration = task.SecondGeneration

	return webuiTask
}

// 根据性别获取底图
func getImageByGender(gender int) string {
	switch gender {
	case 2:
		return "http://down.haitaotaopa.com/camera/template/316d41336765db9d4bb84eb921d0a8e1c7d8fa6a755182b054f92f0373559266.png"
	default:
		return "http://down.haitaotaopa.com/camera/template/7962440c7000170df9d07e33a6.png"
	}
}

// 根据性别获取提示词
func getPromptByGender(gender int, card, lora string, weight float64) string {
	switch gender {
	case 2:
		return fmt.Sprintf("1 man, ((male, boy)), solo, focus on face, front view photo, (((sky blue background))), (clean background), studio lighting, smile, portrait, upper body, %s, <lora:%s:%.2f>", card, lora, weight)
	default:
		return fmt.Sprintf("1 woman, solo, focus on face, front view photo, pink background, (clean background), studio lighting, smile, portrait, upper body, %s, <lora:%s:%.2f>", card, lora, weight)
	}
}

// 写真
func getPhotoTask(c *gin.Context) lib.Task {
	webuiTask := lib.Task{}

	taskId, err := lib.PopSDPhotoTask()
	if err != nil {
		if err.Error() != lib.RedisNull {
			logApi.Errorf("[Redis] pop photo task error: %v", err)
		}
		return webuiTask
	}

	task := &models.UserPhotoImage{ID: taskId}
	if err = task.GetByID(); err != nil {
		logApi.Errorf("[Mysql] get photo task: %d error: %v", taskId, err)
		lib.PushSDPhotoTask(taskId, false)
		return webuiTask
	}
	if task.ImgUrl != "" {
		return webuiTask
	}
	ptask := &models.UserPhotoTask{ID: task.TaskId}
	if err = ptask.GetByID(); err != nil {
		logApi.Errorf("[Mysql] get photo task: %d error: %v", taskId, err)
		lib.PushSDPhotoTask(taskId, false)
		return webuiTask
	}
	if ptask.Status != models.DEFAULT && ptask.Status != models.RUNNING {
		return webuiTask
	}

	// 自己训练的模型
	card := &models.UserCardImage{ID: ptask.AvatarId}
	if err = card.GetByID(); err != nil {
		logApi.Errorf("[Mysql] get card image: %d error: %v", card.ID, err)
		lib.PushSDPhotoTask(taskId, false)
		return webuiTask
	}

	//用户信息
	customer := &models.UserAccount{ID: ptask.CusId}
	if err = customer.GetByID(); err != nil {
		logApi.Errorf("[Mysql] get account error: %v", err)
		lib.PushSDPhotoTask(taskId, false)
		return webuiTask
	}

	// 造型
	pose := &models.UserPhotoPose{ID: task.PoseId}
	if err = pose.GetByID(); err != nil {
		logApi.Errorf("[Mysql] get pose: %d error: %v", pose.ID, err)
		lib.PushSDPhotoTask(taskId, false)
		return webuiTask
	}

	// 风格模板
	template := &models.UserPhotoTemplate{ID: pose.TemplateId}
	if err = template.GetByID(); err != nil {
		logApi.Errorf("[Mysql] get template: %d error: %v", template.ID, err)
		lib.PushSDPhotoTask(taskId, false)
		return webuiTask
	}

	frontUrl := customer.Avatar

	// ADetailer
	start := strings.LastIndex(card.Lora, "/")
	end := strings.LastIndex(card.Lora, ".")
	adetailer := &lib.ADetailer{
		AdModel:             pose.AdModel,
		ModelUrl:            card.Lora,
		AdPrompt:            fmt.Sprintf("%s <lora:%s:%.2f>", pose.AdPrompt, card.Lora[start+1:end], task.AdLoraWeight),
		AdNegativePrompt:    pose.AdNegativePrompt,
		AdInpaintWidth:      512,
		AdInpaintHeight:     512,
		AdDenoisingStrength: pose.AdDenoisingStrength,
		AdConfidence:        pose.AdConfidence,
		AdDilateErode:       pose.AdDilateErode,
	}

	// 风格
	stype := lib.Stype{
		Width:          template.Width,
		Height:         template.Height,
		Prompt:         pose.Prompt,
		NegativePrompt: pose.NegativePrompt,
		SamplerName:    pose.SamplerName,
		Steps:          pose.Steps,
		RestoreFace:    true,
		Tiling:         false,
		CfgScale:       pose.CfgScale,
		Seed:           pose.Seed,
		BatchCount:     1,
		BatchSize:      1,
		MainModelPath:  template.MainModel,
		SubModelUrl:    template.Lora,

		EnableHr:          false,
		HiresUpscaler:     "4x-UltraSharp",
		HrSecondPassSteps: 20,
		HrScale:           2,
		DenoisingStrength: 0.1,

		RandnSource: template.RandnSource,

		ControlNets: make([]*lib.ControlNet, 0),
		// Roop:        &lib.Roop{ImagePath: frontUrl, FaceRestorerVisibility: pose.FaceRestorerVisibility},
		ADetailer: []*lib.ADetailer{adetailer},
	}
	if task.LoraWeight > 0 {
		stype.Prompt = fmt.Sprintf("%s, %s, <lora:%s:%.2f>", stype.Prompt, customer.CardNum, card.Lora[start+1:end], task.LoraWeight)
	}
	if pose.EnableControlNet {
		// ControlNet0
		ctrlNet := &lib.ControlNet{
			ImagePath:     pose.ImgUrl,
			ModelName:     pose.ControlModel,
			Preprocessor:  pose.Preprocessor,
			Weight:        pose.ControlWeight,
			StartCtrlStep: 0,
			EndCtrlStep:   pose.EndControlStep,
			PreprocRes:    512,
			ControlMode:   pose.ControlMode,
			ResizeMode:    pose.ResizeMode,
			PixelPerfect:  pose.PixelPerfect,
		}
		if pose.ControlImage != "" {
			ctrlNet.ImagePath = pose.ControlImage
		}
		stype.ControlNets = append(stype.ControlNets, ctrlNet)

		// ControlNet1
		if pose.ControlWeight1 > 0 {
			ctrlNet1 := &lib.ControlNet{
				ImagePath:     pose.ImgUrl,
				ModelName:     pose.ControlModel1,
				Preprocessor:  pose.Preprocessor1,
				Weight:        pose.ControlWeight1,
				StartCtrlStep: 0,
				EndCtrlStep:   pose.EndControlStep1,
				PreprocRes:    512,
				ControlMode:   pose.ControlMode1,
				ResizeMode:    pose.ResizeMode1,
				PixelPerfect:  pose.PixelPerfect1,
			}
			if pose.ControlImage1 != "" {
				ctrlNet1.ImagePath = pose.ControlImage1
			}
			stype.ControlNets = append(stype.ControlNets, ctrlNet1)
		}
	}
	if pose.EnableRoop {
		stype.Roop = &lib.Roop{
			ImagePath:              frontUrl,
			FaceRestorerVisibility: pose.FaceRestorerVisibility,
			FaceRestorerName:       pose.FaceRestorerName,
		}
	}
	if stype.Seed <= 0 && task.Seed > 0 {
		stype.Seed = task.Seed
	}

	// 任务
	webuiTask.TaskType = 2
	webuiTask.TaskId = uint(taskId)
	webuiTask.Callback = lib.WebUICallbackPhoto
	webuiTask.UserId = task.CusId
	webuiTask.Stype = stype
	webuiTask.SecondGeneration = task.SecondGeneration

	// 更新任务状态
	if ptask.Status == models.DEFAULT {
		if err = ptask.UpdateStatus(models.RUNNING, ""); err != nil {
			logApi.Errorf("[Mysql] update status %d:1 failed: %s", ptask.ID, err)
		}
	}

	return webuiTask
}

// 高清任务
func GetPhotoHrTask(c *gin.Context) {
	taskId, err := lib.PopSDPhotoHrTask()
	if err != nil {
		if err.Error() != lib.RedisNull {
			logApi.Errorf("[Redis] pop photo hr task error: %v", err)
		}
		c.JSON(http.StatusOK, Response{FAILURE, lib.TaskPhotoHr{}})
		return
	}

	photo := &models.UserPhotoImage{ID: taskId}
	if err = photo.GetByID(); err != nil {
		if err.Error() == models.NoRowError {
			c.JSON(http.StatusOK, Response{FAILURE, lib.TaskPhotoHr{}})
			return
		}
		logApi.Errorf("[Mysql] get photo image: %d error: %v", taskId, err)
		lib.PushSDPhotoHrTask(taskId)
		c.JSON(http.StatusOK, Response{FAILURE, lib.TaskPhotoHr{}})
		return
	}
	if photo.HrDownUrl != "" {
		c.JSON(http.StatusOK, Response{FAILURE, lib.TaskPhotoHr{}})
		return
	}

	// 检查用户
	customer := &models.UserAccount{ID: photo.CusId}
	if err = customer.GetByID(); err != nil {
		if err.Error() == models.NoRowError {
			c.JSON(http.StatusOK, Response{FAILURE, lib.TaskPhotoHr{}})
			return
		}
		logApi.Errorf("[Mysql] get customer: %d error: %v", photo.CusId, err)
		lib.PushSDPhotoHrTask(taskId)
		c.JSON(http.StatusOK, Response{FAILURE, lib.TaskPhotoHr{}})
		return
	}
	if !customer.Enabled {
		c.JSON(http.StatusOK, Response{FAILURE, lib.TaskPhotoHr{}})
		return
	}

	record := &lib.TaskRecord{TaskType: lib.REC_HR, TaskID: taskId}
	record.Set()

	// 任务
	webuiTask := lib.TaskPhotoHr{
		TaskId:   taskId,
		Callback: lib.WebUICallbackPhotoHr,
		ImageUrl: photo.DownUrl,
	}
	c.JSON(http.StatusOK, Response{SUCCESS, webuiTask})
}
