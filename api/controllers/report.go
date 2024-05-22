package controllers

import (
	"camera/lib"
	"camera/models"
	"camera/monitor"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/nfnt/resize"
)

// 写真上报
func ReportPhotoTask(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logApi.Errorf("[IO] read body failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "写真任务上报失败"})
		return
	}
	b, err := lib.DESDecrypt(string(body), lib.WebUIDeskey)
	if err != nil {
		logApi.Warnf("des decrypt failed: %s, body: %s", err, string(body))
		c.JSON(http.StatusOK, Response{SUCCESS, "解密失败"})
		return
	}
	callback := lib.SDCallback{}
	if err = json.Unmarshal(b, &callback); err != nil {
		logApi.Errorf("[IO] json unmarshal failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "写真任务json解析失败"})
		return
	}

	record := &lib.TaskRecord{TaskType: lib.REC_PHOTO, TaskID: int(callback.TaskId)}
	record.Delete()

	// 校验写真图片
	output := &models.UserPhotoImage{ID: int(callback.TaskId)}
	if err = output.GetByID(); err != nil {
		if err.Error() == models.NoRowError {
			c.JSON(http.StatusOK, Response{SUCCESS, ""})
			return
		}
		logApi.Errorf("[Mysql] get photo image failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "写真Image获取失败"})
		return
	}
	if output.ImgUrl != "" {
		c.JSON(http.StatusOK, Response{SUCCESS, ""})
		return
	}
	if len(callback.Images) == 0 {
		c.JSON(http.StatusOK, Response{FAILURE, "写真图片缺失"})
		return
	}

	// 水印图
	wfname := lib.GenGUID()
	wkey := fmt.Sprintf("cphoto/%s/%s/%s.png", wfname[:2], wfname[2:4], wfname[4:])
	keyz2 := fmt.Sprintf("cphoto/%s/%s/%s.png", wfname[:2], wfname[2:4], wfname[4:26])

	// 下载无水印图片
	imgurl := callback.Images[0]
	img, err := lib.DownloadImage(imgurl)
	if err != nil {
		logApi.Errorf("[IO] download image failed: %s, url: %s", err, imgurl)
		c.JSON(http.StatusOK, Response{FAILURE, "写真图片下载失败"})
		return
	}
	output.DownUrl = imgurl

	imgz2 := resize.Resize(300, 0, img, resize.Lanczos2)
	if _, err = lib.UploadQNCDN(imgz2, keyz2); err != nil {
		logApi.Errorf("[IO] upload cdn failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "上传CDN失败"})
		return
	}
	output.ThumbUrl, err = url.JoinPath(lib.QiniuHost, keyz2)
	if err != nil {
		logApi.Errorf("url joinpath failed: %s, %s %s", err, lib.QiniuHost, keyz2)
		c.JSON(http.StatusOK, Response{FAILURE, "url拼接失败"})
		return
	}

	// 下载水印图
	wimgurl := callback.WaterMarks[0]
	if wimgurl == "" {
		wimg, err := lib.AddWatermark(img, `watermark.png`)
		if err != nil {
			logApi.Errorf("[IO] add watermark failed: %s", err)
			c.JSON(http.StatusOK, Response{FAILURE, "添加水印失败"})
			return
		}

		// 上传CDN
		if _, err = lib.UploadQNCDN(wimg, wkey); err != nil {
			logApi.Errorf("[IO] upload cdn failed: %s", err)
			c.JSON(http.StatusOK, Response{FAILURE, "上传CDN失败"})
			return
		}
		output.ImgUrl, err = url.JoinPath(lib.QiniuHost, wkey)
		if err != nil {
			logApi.Errorf("url joinpath failed: %s, %s %s", err, lib.QiniuHost, wkey)
			c.JSON(http.StatusOK, Response{FAILURE, "url拼接失败"})
			return
		}
	} else {
		output.ImgUrl = wimgurl
	}

	// 更新
	if output.Seed <= 0 {
		output.Seed = callback.Seed
	}
	if err = output.UpdateImageUrl(); err != nil {
		logApi.Errorf("[Mysql] update image url failed: %s, id: %d, url: %s", err, output.ID, output.ImgUrl)
		c.JSON(http.StatusOK, Response{FAILURE, "更新CDN失败"})
		return
	}

	//校验4张分身是否全部完成，更新分身任务状态
	images, err := output.GetByTaskID()
	if err != nil {
		logApi.Errorf("[Mysql] get card image to check complete failed: %s", err)
		c.JSON(http.StatusOK, Response{SUCCESS, ""})
		return
	}
	completed := true
	for _, image := range images {
		if image.ImgUrl == "" {
			completed = false
			break
		}
	}
	if completed {
		task := &models.UserPhotoTask{ID: output.TaskId}
		if err = task.UpdateStatus(models.SUCCESS, ""); err != nil {
			logApi.Warnf("update task %d complete failed: %s", task.ID, err.Error())
		}
	}
	c.JSON(http.StatusOK, Response{SUCCESS, ""})
}

// 上报Lora模型
func ReportLoraTask(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logApi.Errorf("[IO] read body failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "分身任务上报失败"})
		return
	}
	b, err := lib.DESDecrypt(string(body), lib.WebUIDeskey)
	if err != nil {
		logApi.Warnf("des decrypt failed: %s, body: %s", err, string(body))
		c.JSON(http.StatusOK, Response{SUCCESS, "解密失败"})
		return
	}
	callback := lib.SDCallback{}
	if err = json.Unmarshal(b, &callback); err != nil {
		logApi.Errorf("[IO] json unmarshal failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "分身任务json解析失败"})
		return
	}

	record := &lib.TaskRecord{TaskType: lib.REC_LORA, TaskID: int(callback.TaskId)}
	record.Delete()

	//校验任务
	task := &models.UserCardTask{ID: int(callback.TaskId)}
	if err := task.GetByID(); err != nil {
		logApi.Errorf("[Mysql] get card task failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "任务获取失败"})
		return
	}
	if task.Status > models.RUNNING {
		c.JSON(http.StatusOK, Response{FAILURE, "任务状态不匹配"})
		return
	}

	//处理结果
	switch callback.Code {
	case 1:
		//成功
		task.Gender = callback.Gender
		if err = task.UpdateGender(); err != nil {
			logApi.Warnf("update task %d gender: %d failed: %s", task.ID, task.Gender, err.Error())
		}

		sct := 0
		for _, lora := range callback.Loras {
			output := &models.UserCardImage{
				CusId:            task.CusId,
				TaskId:           task.ID,
				Lora:             lora.LoraPath,
				Weight:           lora.Weight,
				PromptWeight:     lora.PromptWeight,
				SecondGeneration: lora.SecondGeneration,
			}
			if err = output.Create(); err != nil {
				logApi.Errorf("create lora model taskId: %d failed: %s", task.ID, err)
				continue
			}

			if err = lib.PushSDCardTask(output.ID); err != nil {
				logApi.Errorf("push sd card task: %d failed: %s", output.ID, err)
			}
			sct++
		}

		// 删除用户上传图片
		if sct > 0 {
			cdn := &lib.UploadCDNTask{
				TaskType: lib.TRAIN_SIDE,
				TaskId:   task.CusId,
			}
			if err = lib.PushCDNTask(cdn); err != nil {
				logApi.Warnf("delete train image %d failed: %s", task.CusId, err)
			}
		}
	// case 2:
	default:
		//失败,且无需重试
		if err = task.UpdateStatus(models.FAILED, callback.Message); err != nil {
			logApi.Warnf("update task %d failed: %s", task.ID, err.Error())
			c.JSON(http.StatusOK, Response{FAILURE, "操作失败"})
			return
		}
		// 报警
		bark := monitor.Bark{Title: "Lora失败", Message: fmt.Sprintf("任务ID:%d, 错误信息:%s", task.ID, callback.Message)}
		bark.SendMessage(monitor.CARD_LORA)
		// default:
		// 	logApi.Warnf("bad account, code: %d, message: %s, task_id: %d", callback.Code, callback.Message, callback.TaskId)
		// 	if task.SdAccId > 0 {
		// 		if fct, _ := lib.SDAccountError(task.SdAccId); fct >= 3 {
		// 			//禁用账号
		// 			account := &models.SdAccount{ID: task.SdAccId}
		// 			if err = account.Disable(); err != nil {
		// 				logApi.Warnf("update sd account %d disabled failed: %s", task.SdAccId, err.Error())
		// 			}
		// 		}
		// 	}
		// 	if err = lib.PushSDTask(task.ID); err != nil {
		// 		logApi.Errorf("push sd task: %d failed: %s", task.ID, err)
		// 		c.JSON(http.StatusOK, Response{FAILURE, "操作失败"})
		// 		return
		// 	}
	}
	c.JSON(http.StatusOK, Response{SUCCESS, ""})
}

// 分身上报
func ReportCardTask(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logApi.Errorf("[IO] read body failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "分身任务上报失败"})
		return
	}
	b, err := lib.DESDecrypt(string(body), lib.WebUIDeskey)
	if err != nil {
		logApi.Warnf("des decrypt failed: %s, body: %s", err, string(body))
		c.JSON(http.StatusOK, Response{SUCCESS, "解密失败"})
		return
	}
	callback := lib.SDCallback{}
	if err = json.Unmarshal(b, &callback); err != nil {
		logApi.Errorf("[IO] json unmarshal failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "分身任务json解析失败"})
		return
	}

	record := &lib.TaskRecord{TaskType: lib.REC_CARD, TaskID: int(callback.TaskId)}
	record.Delete()

	// 校验分身图片
	output := &models.UserCardImage{ID: int(callback.TaskId)}
	if err = output.GetByID(); err != nil {
		logApi.Errorf("[Mysql] get card image failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "分身Image获取失败"})
		return
	}
	if output.ImgUrl != "" {
		c.JSON(http.StatusOK, Response{SUCCESS, ""})
		return
	}
	if len(callback.Images) == 0 {
		c.JSON(http.StatusOK, Response{FAILURE, "分身图片缺失"})
		return
	}

	// 下载图片
	output.ImgUrl = callback.Images[0]
	output.Seed = callback.Seed
	if err = output.UpdateImageUrl(); err != nil {
		logApi.Errorf("[Mysql] update image url failed: %s, id: %d, url: %s", err, output.ID, output.ImgUrl)
		c.JSON(http.StatusOK, Response{FAILURE, "更新CDN失败"})
		return
	}

	//校验4张分身是否全部完成，更新分身任务状态
	images, err := output.GetByTaskID()
	if err != nil {
		logApi.Errorf("[Mysql] get card image to check complete failed: %s", err)
		c.JSON(http.StatusOK, Response{SUCCESS, ""})
		return
	}
	if len(images) != 4 {
		logApi.Warnf("bad images count, task_id: %d", output.TaskId)
		c.JSON(http.StatusOK, Response{SUCCESS, ""})
		return
	}
	completed := true
	for _, image := range images {
		if image.ImgUrl == "" {
			completed = false
			break
		}
	}
	if completed {
		task := &models.UserCardTask{ID: output.TaskId}
		if err = task.UpdateStatus(models.SUCCESS, ""); err != nil {
			logApi.Warnf("update task %d complete failed: %s", task.ID, err.Error())
		}

		//校验用户
		customer := &models.UserAccount{ID: output.CusId}
		if err = customer.GetByID(); err != nil {
			logApi.Warnf("get customer: %d for step: 3 failed: %s", output.CusId, err.Error())
		} else {
			customer.Step = models.CARD_STEP_SELECT
			if err = customer.UpdateStep(); err != nil {
				logApi.Warnf("customer: %d step: 3 failed: %s", output.CusId, err.Error())
			}
			// 短信通知
			if err = lib.SendPhoneMessage(customer.Mobile, lib.MessageCardSuccess); err != nil {
				logApi.Warnf("[SMS] send phone message failed: %s", err)
			}
		}
	}
	c.JSON(http.StatusOK, Response{SUCCESS, ""})
}

// 照片识别上报
func ReportPhotoRecognizeTask(c *gin.Context) {
	ptype := c.Query("type")
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logApi.Errorf("[IO] read body failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "照片识别任务上报失败"})
		return
	}
	b, err := lib.DESDecrypt(string(body), lib.WebUIDeskey)
	if err != nil {
		logApi.Warnf("des decrypt failed: %s, body: %s", err, string(body))
		c.JSON(http.StatusOK, Response{SUCCESS, "解密失败"})
		return
	}
	callback := lib.SDCallback{}
	if err = json.Unmarshal(b, &callback); err != nil {
		logApi.Errorf("[IO] json unmarshal failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "照片识别任务json解析失败"})
		return
	}

	// 0-未识别,1-识别成功,2-识别失败
	switch ptype {
	case "front":
		for k, v := range callback.CheckStatus {
			record := &lib.TaskRecord{TaskType: lib.REC_FRONT, TaskID: k}
			record.Delete()

			front := &models.UserFrontImage{ID: k}
			if err = front.GetByID(); err != nil {
				logApi.Errorf("[Mysql] get front image failed: %s", err)
				continue
			}
			if front.Status != 1 {
				continue
			}

			status := 0
			switch v {
			case 1:
				status = 2
			case 2:
				status = 4
			}
			if err = front.UpdateStatus(status); err != nil {
				logApi.Errorf("[Mysql] update front image status failed: %s", err)
			}
		}
	case "side":
		for k, v := range callback.CheckStatus {
			record := &lib.TaskRecord{TaskType: lib.REC_SIDE, TaskID: k}
			record.Delete()

			input := &models.UserInputImage{ID: k}
			if err = input.GetByID(); err != nil {
				logApi.Errorf("[Mysql] get input image failed: %s", err)
				continue
			}
			if input.Status != 1 {
				continue
			}

			status := 0
			switch v {
			case 1:
				status = 2
			case 2:
				status = 4
			}
			if err = input.UpdateStatus(status); err != nil {
				logApi.Errorf("[Mysql] update input image status failed: %s", err)
			}
		}
	}
	c.JSON(http.StatusOK, Response{SUCCESS, ""})
}

// 高清上报
func ReportPhotoHrTask(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logApi.Errorf("[IO] read body failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "高清任务上报失败"})
		return
	}
	b, err := lib.DESDecrypt(string(body), lib.WebUIDeskey)
	if err != nil {
		logApi.Warnf("des decrypt failed: %s, body: %s", err, string(body))
		c.JSON(http.StatusOK, Response{SUCCESS, "解密失败"})
		return
	}
	callback := lib.PhotoHrCallback{}
	if err = json.Unmarshal(b, &callback); err != nil {
		logApi.Errorf("[IO] json unmarshal failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "高清任务json解析失败"})
		return
	}

	record := &lib.TaskRecord{TaskType: lib.REC_HR, TaskID: int(callback.TaskId)}
	record.Delete()

	// 校验写真图片
	output := &models.UserPhotoImage{ID: int(callback.TaskId)}
	if err = output.GetByID(); err != nil {
		if err.Error() == models.NoRowError {
			c.JSON(http.StatusOK, Response{SUCCESS, ""})
			return
		}
		logApi.Errorf("[Mysql] get photo image failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "写真Image获取失败"})
		return
	}
	if output.HrDownUrl != "" {
		c.JSON(http.StatusOK, Response{SUCCESS, ""})
		return
	}
	output.HrDownUrl = callback.ImageUrl

	// 水印图
	if callback.WaterImageUrl == "" {
		waterImage, err := lib.DownloadImage(callback.ImageUrl)
		if err == nil {
			wimg, err := lib.AddWatermark(waterImage, `watermark.png`)
			if err == nil {
				// 上传CDN
				wfname := lib.GenGUID()
				wkey := fmt.Sprintf("cphoto/%s/%s/%s.png", wfname[:2], wfname[2:4], wfname[4:])
				if _, err = lib.UploadQNCDN(wimg, wkey); err == nil {
					output.HrImgUrl, _ = url.JoinPath(lib.QiniuHost, wkey)
				}
			}
		}
	} else {
		output.HrImgUrl = callback.WaterImageUrl
	}
	if output.HrImgUrl == "" {
		output.HrImgUrl = output.ImgUrl
	}

	// 更新
	if err = output.UpdateHrImageUrl(); err != nil {
		logApi.Errorf("[Mysql] update image url failed: %s, id: %d, url: %s", err, output.ID, output.ImgUrl)
		c.JSON(http.StatusOK, Response{FAILURE, "更新CDN失败"})
		return
	}
	c.JSON(http.StatusOK, Response{SUCCESS, ""})
}
