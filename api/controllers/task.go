package controllers

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"camera/lib"
	"camera/models"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/nfnt/resize"
)

// 校验素材
func checkMaterial(fheader *multipart.FileHeader, needzip bool, folder, cardid string) (string, string, string, error) {
	imgReader, err := fheader.Open()
	if err != nil {
		return "", "", "", fmt.Errorf("fileheader open failed: %s", err)
	}

	//验证图片格式
	imgBody, err := io.ReadAll(imgReader)
	if err != nil {
		return "", "", "", fmt.Errorf("read file failed: %s", err)
	}
	fext := ""
	orient := 0
	mimeType := http.DetectContentType(imgBody[:512])
	switch mimeType {
	case "image/png":
		fext = ".png"
	case "image/jpeg":
		fext = ".jpg"
		imgReader.Seek(0, 0)
		orient, err = lib.ReadOrientation(imgReader)
		if err != nil {
			logApi.Warnf("read orientation failed: %s", err)
		}
	}
	if fext == "" {
		return "", "", "", fmt.Errorf("filetype not supported: %s", mimeType)
	}

	fname := lib.GenGUID()[:26]
	fileName, fileNameZ := fmt.Sprintf("%s/%s/%s%s", folder, cardid, fname[:15], fext), ""
	realFileName := "images/" + fileName
	if err = os.MkdirAll(filepath.Dir(realFileName), os.ModePerm); err != nil {
		return "", "", "", fmt.Errorf("create dir failed: %s", err)
	}
	if err = os.WriteFile(realFileName, imgBody, 0666); err != nil {
		return "", "", "", fmt.Errorf("save image failed: %s", err)
	}

	// 旋转图片
	if orient > 0 {
		img, err := lib.GetImage(realFileName)
		if err != nil {
			logApi.Warnf("get image failed: %s, path: %s", err, realFileName)
		} else {
			var nrgba *image.NRGBA
			switch orient {
			case 6:
				nrgba = imaging.Rotate270(img)
			case 8:
				nrgba = imaging.Rotate90(img)
			}
			if nrgba != nil {
				os.Remove(realFileName)
				imgfile, _ := os.Create(realFileName)
				defer imgfile.Close()
				jpeg.Encode(imgfile, nrgba, &jpeg.Options{Quality: 85})
			}
		}
	}

	// 压缩
	if needzip {
		img, err := lib.GetImage(realFileName)
		if err != nil {
			return "", "", "", fmt.Errorf("get image failed: %s", err)
		}
		imgz := resize.Resize(512, 0, img, resize.Lanczos2)

		bbb := new(bytes.Buffer)
		switch mimeType {
		case "image/png":
			png.Encode(bbb, imgz)
		case "image/jpeg":
			jpeg.Encode(bbb, imgz, nil)
		}

		fileNameZ = fmt.Sprintf("%s/%s/%s%s", folder, cardid, fname, fext)
		os.WriteFile("images/"+fileNameZ, bbb.Bytes(), 0666)
	}

	return fileName, realFileName, fileNameZ, nil
}

// 上传正面照
func UploadUserFrontImage(c *gin.Context) {
	//校验用户
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "用户不存在"})
		return
	}
	if customer.Step != models.CARD_STEP_FRONT {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "当前不是上传照片步骤"})
		return
	}

	//处理图片
	mulForm, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误2"})
		return
	}
	material, ok := mulForm.File["material"]
	if !ok || len(material) == 0 {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "缺少素材图片"})
		return
	}

	fheader := material[0]
	fileName, _, fileNameZ, err := checkMaterial(fheader, true, "front", customer.CardId)
	if err != nil {
		logApi.Debugf("[IO] %d %s", customer.ID, err)
		c.JSON(http.StatusOK, Response{FAILURE, "上传失败"})
		return
	}

	//保存
	input := &models.UserFrontImage{
		CusId:    customer.ID,
		ImgUrl:   fileName,
		ThumbUrl: fileNameZ,
		Status:   2, //1
	}
	if err = input.GetByCusId(); err != nil {
		if err.Error() != models.NoRowError {
			logApi.Errorf("[Mysql] check front image failed: %s", err)
			c.JSON(http.StatusOK, Response{FAILURE, "上传失败"})
			return
		}
		if err = input.Create(); err != nil {
			logApi.Errorf("[Mysql] create front image failed: %s", err)
			c.JSON(http.StatusOK, Response{FAILURE, "上传失败"})
			return
		}
	} else {
		input.ImgUrl = fileName
		input.ThumbUrl = fileNameZ
		input.Status = 2 // 1
		if err = input.Update(); err != nil {
			logApi.Errorf("[Mysql] update front image failed: %s", err)
			c.JSON(http.StatusOK, Response{FAILURE, "上传失败"})
			return
		}
	}
	// if err = lib.PushSDCheckFrontTask([]int{input.ID}); err != nil {
	// 	logApi.Errorf("[Redis] push check front task failed: %s", err)
	// }

	c.JSON(http.StatusOK, Response{SUCCESS, ""})
}

// 检查正面照
func CheckUserFrontImage(c *gin.Context) {
	cusId := GetUserID(c)

	input := &models.UserFrontImage{CusId: cusId}
	if err := input.GetByCusId(); err != nil {
		if err.Error() == models.NoRowError {
			c.JSON(http.StatusOK, Response{SUCCESS, ""})
			return
		}
		logApi.Errorf("[Mysql] get front image failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "状态获取失败"})
		return
	}

	if !strings.HasPrefix(input.ThumbUrl, "http") {
		input.ThumbUrl, _ = url.JoinPath(lib.ImageHost, input.ThumbUrl)
	}
	c.JSON(http.StatusOK, Response{SUCCESS, input})
}

// 进入上传侧面照步骤
func InUploadSideImageStep(c *gin.Context) {
	//校验用户
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}
	if customer.Step != models.CARD_STEP_FRONT {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "当前不是上传正面照阶段"})
		return
	}

	// 检查正面照
	front := &models.UserFrontImage{CusId: customer.ID}
	if err = front.GetByCusId(); err != nil {
		logApi.Errorf("[Mysql] get front image failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "操作失败"})
		return
	}
	if front.Status != 2 {
		c.JSON(http.StatusOK, Response{FAILURE, "正面照未识别成功"})
		return
	}

	customer.Step = models.CARD_STEP_SIDE
	if err = customer.UpdateStep(); err != nil {
		logApi.Errorf("[Mysql] update step failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "操作失败"})
		return
	}
	c.JSON(http.StatusOK, Response{SUCCESS, ""})
}

// 上传侧面照
func UploadUserCardImage(c *gin.Context) {
	//校验用户
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}
	if customer.Step != models.CARD_STEP_SIDE {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "当前不是上传照片步骤"})
		return
	}

	//处理图片
	mulForm, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误2"})
		return
	}
	material, ok := mulForm.File["material"]
	if !ok || len(material) == 0 {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "缺少素材图片"})
		return
	}

	// 删除识别失败照片
	iim := &models.UserInputImage{}
	if err = iim.DeleteFail(customer.ID); err != nil {
		logApi.Errorf("[Mysql] delete fail image failed: %s", err)
	}

	ids := make([]int, 0)
	for _, fheader := range material {
		fileName, _, fileNameZ, err := checkMaterial(fheader, true, "side", customer.CardId)
		if err != nil {
			logApi.Debugf("[IO] %d %s", customer.ID, err)
			continue
		}

		//保存
		input := &models.UserInputImage{
			CusId:    customer.ID,
			ImgUrl:   fileName,
			ThumbUrl: fileNameZ,
			Status:   2, //1
		}
		if err = input.Create(); err != nil {
			logApi.Errorf("[Mysql] create card input image failed: %s", err)
			continue
		}
		ids = append(ids, input.ID)
	}

	if len(ids) > 0 {
		// if err = lib.PushSDCheckSideTask(ids); err != nil {
		// 	logApi.Errorf("[Redis] push check side task failed: %s", err)
		// }
	}

	c.JSON(http.StatusOK, Response{SUCCESS, ""})
}

// 检查侧面照
func CheckUserCardImage(c *gin.Context) {
	cusId := GetUserID(c)

	input := &models.UserInputImage{CusId: cusId}
	cards, err := input.GetByCusId()
	if err != nil {
		if err.Error() == models.NoRowError {
			c.JSON(http.StatusOK, Response{SUCCESS, cards})
			return
		}
		logApi.Errorf("[Mysql] get front image failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "状态获取失败"})
		return
	}

	for _, card := range cards {
		if !strings.HasPrefix(card.ThumbUrl, "http") {
			card.ThumbUrl, _ = url.JoinPath(lib.ImageHost, card.ThumbUrl)
		}
	}

	c.JSON(http.StatusOK, Response{SUCCESS, cards})
}

// 删除侧面照
func DeleteUserCardImage(c *gin.Context) {
	imgId, _ := strconv.Atoi(c.Request.FormValue("id"))
	input := &models.UserInputImage{ID: imgId}
	if err := input.GetByID(); err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}

	// 校验用户
	if input.CusId != GetUserID(c) {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}

	if err := input.Delete(); err != nil {
		c.JSON(http.StatusOK, Response{FAILURE, "删除失败"})
		return
	}
	c.JSON(http.StatusOK, Response{SUCCESS, ""})
}

// 创建分身任务
func CreateUserCardTask(c *gin.Context) {
	//校验用户
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}
	if customer.RemainTimes < 1 {
		c.JSON(http.StatusOK, Response{NO_CARD_TIMES, "次数不足"})
		return
	}
	if customer.Step != models.CARD_STEP_SIDE {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "当前不是制作分身步骤"})
		return
	}

	//检查正面照是否合格
	front := &models.UserFrontImage{CusId: customer.ID}
	if err = front.GetByCusId(); err != nil {
		logApi.Errorf("[Mysql] get front image failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "任务创建失败，请重试"})
		return
	}
	if front.Status != 2 {
		c.JSON(http.StatusOK, Response{CARD_FRONT_NOT_ENOUGH, "正面照缺失"})
		return
	}

	//检查侧面照是否满足需求
	inputFront := &models.UserInputImage{CusId: customer.ID}
	cards, err := inputFront.GetByCusId()
	if err != nil {
		logApi.Errorf("[Mysql] get front image failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "任务创建失败，请重试"})
		return
	}
	sct := 0
	for _, card := range cards {
		if card.Status == 2 {
			sct++
		}
		if sct >= 9 {
			break
		}
	}
	if sct < 9 {
		c.JSON(http.StatusOK, Response{CARD_SIDE_NOT_ENOUGH, "照片最少9张"})
		return
	}

	// 正面照上传CDN
	frontImage, err := lib.GetImage(fmt.Sprintf("images/%s", front.ThumbUrl))
	if err != nil {
		logApi.Errorf("get front image failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "任务创建失败，请重试"})
		return
	}
	fname := filepath.Base(front.ThumbUrl)
	frontKey := fmt.Sprintf("cfront/%s/%s/%s", fname[:2], fname[2:4], fname[4:])
	if _, err = lib.UploadQNCDN(frontImage, frontKey); err != nil {
		logApi.Errorf("upload front image failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "任务创建失败，请重试"})
		return
	}
	frontUrl, err := url.JoinPath(lib.QiniuHost, frontKey)
	if err != nil {
		logApi.Errorf("url joinpath failed: %s,  %s %s", err, lib.QiniuHost, frontKey)
		c.JSON(http.StatusOK, Response{FAILURE, "任务创建失败，请重试"})
		return
	}

	//先扣除分身次数
	if err = customer.DecrRemainTimes(); err != nil {
		logApi.Errorf("[Mysql] decr remain times failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "任务创建失败"})
		return
	}

	//创建任务
	task := &models.UserCardTask{
		CusId:     customer.ID,
		CreatedAt: time.Now(),
		Status:    models.DEFAULT,
		FrontUrl:  frontUrl,
	}
	if err = task.Create(); err != nil {
		logApi.Errorf("[Mysql] cusid: %d create card task failed: %s", customer.ID, err)
		c.JSON(http.StatusOK, Response{FAILURE, "任务创建失败"})
		return
	}
	if err = lib.PushSDTask(task.ID); err != nil {
		logApi.Errorf("push sd task: %d failed: %s", task.ID, err)
	}
	os.RemoveAll(fmt.Sprintf("images/front/%s", customer.CardId))

	c.JSON(http.StatusOK, Response{SUCCESS, task.ID})
}

// 查看分身任务状态
func GetCardStatus(c *gin.Context) {
	// 校验用户
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}

	tid, _ := strconv.Atoi(c.Query("id"))
	if tid == 0 {
		tid = customer.TempCardTaskId
		if tid == 0 {
			c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
			return
		}
	}

	task := &models.UserCardTask{ID: tid}
	if err := task.GetByID(); err != nil {
		logApi.Errorf("[Mysql] get card task failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "状态获取失败"})
		return
	}
	if task.CusId != customer.ID {
		c.JSON(http.StatusOK, Response{FAILURE, "状态获取失败"})
		return
	}

	if task.Status == models.RUNNING || task.Status == models.DEFAULT {
		c.JSON(http.StatusOK, Response{GEN_TASK_RUNNING, ""})
		return
	}
	if task.Status == models.CANCELD {
		c.JSON(http.StatusOK, Response{GEN_TASK_CANCELED, ""})
		return
	}
	if task.Status == models.FAILED {
		c.JSON(http.StatusOK, Response{GEN_TASK_FAILED, ""})
		return
	}
	if task.Status == models.SUCCESS {
		output := &models.UserCardImage{TaskId: task.ID}
		images, err := output.GetByTaskIDForWeb(customer.AvatarId)
		if err != nil {
			c.JSON(http.StatusOK, Response{FAILURE, "read error"})
			return
		}
		result := make(map[string]any)
		result["images"] = images
		result["last_card"] = customer.Avatar

		c.JSON(http.StatusOK, Response{SUCCESS, result})
		return
	}
	c.JSON(http.StatusOK, Response{FAILURE, ""})
}

// 查看分身列表
func GetCardList(c *gin.Context) {
	// 校验用户
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}

	if customer.CardTaskId == 0 {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}

	task := &models.UserCardTask{ID: customer.CardTaskId}
	if err := task.GetByID(); err != nil {
		logApi.Errorf("[Mysql] get card task failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "状态获取失败"})
		return
	}
	if task.CusId != customer.ID {
		c.JSON(http.StatusOK, Response{FAILURE, "状态获取失败"})
		return
	}

	if task.Status == models.RUNNING || task.Status == models.DEFAULT {
		c.JSON(http.StatusOK, Response{GEN_TASK_RUNNING, ""})
		return
	}
	if task.Status == models.CANCELD {
		c.JSON(http.StatusOK, Response{GEN_TASK_CANCELED, ""})
		return
	}
	if task.Status == models.FAILED {
		c.JSON(http.StatusOK, Response{GEN_TASK_FAILED, ""})
		return
	}
	if task.Status == models.SUCCESS {
		output := &models.UserCardImage{TaskId: task.ID}
		images, err := output.GetByTaskIDForWeb(customer.AvatarId)
		if err != nil {
			c.JSON(http.StatusOK, Response{FAILURE, "read error"})
			return
		}
		c.JSON(http.StatusOK, Response{SUCCESS, images})
		return
	}
	c.JSON(http.StatusOK, Response{FAILURE, ""})
}

// 选择一个分身作为主分身
func SelectUserCard(c *gin.Context) {
	//校验用户
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "用户不存在"})
		return
	}

	//校验分身
	cardId, _ := strconv.Atoi(c.Request.FormValue("card_id"))
	card := &models.UserCardImage{ID: cardId}
	if err = card.GetByID(); err != nil {
		logApi.Errorf("[Mysql] get card image failed: %s", err)
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "分身不存在"})
		return
	}
	if card.CusId != customer.ID || card.ImgUrl == "" {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "分身不存在"})
		return
	}

	//检查分身任务
	task := &models.UserCardTask{ID: card.TaskId}
	if err = task.GetByID(); err != nil {
		logApi.Errorf("[Mysql] get card task: %d failed: %s", task.ID, err)
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "分身不存在"})
		return
	}

	customer.FrontUrl = task.FrontUrl
	customer.AvatarId = card.ID
	customer.Avatar = card.ImgUrl
	if err = customer.UpdateAvatar(); err != nil {
		logApi.Errorf("[Mysql] update avatar: %d failed: %s", customer.ID, err)
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "更新头像失败"})
		return
	}
	if customer.Step == models.CARD_STEP_SELECT {
		if err = customer.FinishCardTask(); err != nil {
			logApi.Warnf("[Mysql] update customer: %d step: 4 failed: %s", customer.ID, err)
		}
	}

	c.JSON(http.StatusOK, Response{SUCCESS, ""})
}

// 重置分身
func ResetUserCard(c *gin.Context) {
	//校验用户
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "用户不存在"})
		return
	}
	// if customer.RemainTimes < 1 {
	// 	c.JSON(http.StatusOK, Response{NO_CARD_TIMES, "重置次数不足"})
	// 	return
	// }
	if customer.Step != 4 {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "当前状态不可重置"})
		return
	}

	// 重置分身
	if err = customer.ResetUserCard(); err != nil {
		c.JSON(http.StatusOK, Response{FAILURE, "重置失败"})
		return
	}
	c.JSON(http.StatusOK, Response{SUCCESS, ""})
}

// 放弃本次分身，返回原先状态
func BackLastUserCard(c *gin.Context) {
	//校验用户
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "用户不存在"})
		return
	}
	if customer.Step != models.CARD_STEP_SELECT {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "当前不在分身选择步骤"})
		return
	}
	if customer.CardTaskId == 0 {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "没有历史分身"})
		return
	}

	customer.Step = models.CARD_STEP_OK
	if err = customer.UpdateStep(); err != nil {
		c.JSON(http.StatusOK, Response{FAILURE, "设置失败"})
		return
	}
	c.JSON(http.StatusOK, Response{SUCCESS, ""})
}

// 创建写真任务
func CreatePhotoTask(c *gin.Context) {
	// 校验用户
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "用户不存在"})
		return
	}

	// 检查分身
	card := &models.UserCardImage{ID: customer.AvatarId}
	if err = card.GetByID(); err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "数字分身不存在"})
		return
	}

	// 更像我一点
	likeMeId, _ := strconv.Atoi(c.Request.FormValue("like_me"))
	likePhoto := &models.UserPhotoImage{ID: likeMeId}
	if likeMeId > 0 {
		if err = likePhoto.GetByID(); err != nil {
			c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误1"})
			return
		}
		if likePhoto.CusId != customer.ID {
			c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
			return
		}
	}

	// 校验模板
	templateId, _ := strconv.Atoi(c.Request.FormValue("template_id"))
	template := &models.UserPhotoTemplate{ID: templateId}
	if err = template.GetByID(); err != nil {
		logApi.Errorf("[Mysql] get template failed: %s", err)
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "该模板已下架"})
		return
	}

	// 校验造型
	var poses []*models.UserPhotoPose
	poseId, _ := strconv.Atoi(c.Request.FormValue("pose_id"))
	if poseId == 0 {
		// 随机4个造型
		pose := &models.UserPhotoPose{}
		list, err := pose.List(templateId)
		if err != nil {
			logApi.Errorf("[Mysql] get pose failed: %s", err)
			c.JSON(http.StatusOK, Response{INVALID_PARAM, "造型不存在"})
			return
		}
		if len(list) == 0 {
			c.JSON(http.StatusOK, Response{FAILURE, "造型缺失"})
			return
		}
		lib.Shuffle[models.UserPhotoPose](list)

		for {
			for _, p := range list {
				poses = append(poses, &p)
				if len(poses) >= 4 {
					break
				}
			}
			if len(poses) >= 4 {
				break
			}
		}
	} else {
		// 指定造型，出4张图
		pose := &models.UserPhotoPose{ID: poseId}
		if err = pose.GetByID(); err != nil {
			c.JSON(http.StatusOK, Response{INVALID_PARAM, "造型不存在"})
			return
		}
		for i := 0; i < 4; i++ {
			poses = append(poses, pose)
		}
	}

	// 创建任务
	task := &models.UserPhotoTask{
		CusId:        customer.ID,
		TemplateId:   templateId,
		ControlImage: likePhoto.ThumbUrl,
		LikeMe:       likeMeId > 0,
		AvatarId:     customer.AvatarId,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if task.ControlImage == "" {
		task.ControlImage = poses[0].ImgUrl
	}
	if err = task.Create(); err != nil {
		logApi.Errorf("[Mysql] cusid: %d create photo task failed: %s", customer.ID, err)
		c.JSON(http.StatusOK, Response{FAILURE, "任务创建失败"})
		return
	}
	pct, err := lib.IncrPhotoCount(time.Now(), strconv.Itoa(customer.ID))
	if err != nil {
		logApi.Errorf("[Redis] incr photo count failed: %s", err)
	}
	for i, p := range poses {
		image := &models.UserPhotoImage{
			CusId:  customer.ID,
			TaskId: task.ID,
			PoseId: p.ID,
			Seed:   p.Seed,
		}
		if image.Seed <= 0 && likePhoto.Seed > 0 {
			image.Seed = likePhoto.Seed
		}
		switch p.PoseType {
		case 1:
			image.SecondGeneration = false
			image.LoraWeight = p.LoraWeight
			image.AdLoraWeight = p.AdLoraWeight
		case 2:
			image.SecondGeneration = false
			image.LoraWeight = p.LoraWeight
			image.AdLoraWeight = p.AdLoraWeight + float64(i)*p.AdLoraWeightStep
			if likeMeId > 0 {
				image.SecondGeneration = likePhoto.SecondGeneration
				image.LoraWeight = likePhoto.LoraWeight + p.LoraWeightStep
				if image.LoraWeight > 1 {
					image.LoraWeight = 1
				}
				image.AdLoraWeight = likePhoto.AdLoraWeight + p.AdLoraWeightStep
				if image.AdLoraWeight > 1 {
					image.AdLoraWeight = 1
				}
			}
		}
		if err = image.Create(); err != nil {
			logApi.Errorf("[Mysql] create photo image failed: %s", err)
			continue
		}

		if err = lib.PushSDPhotoTask(image.ID, pct > lib.PhotoLimit); err != nil {
			logApi.Errorf("[Redis] push photo task failed: %s", err)
		}
	}

	// 统计模板使用次数
	tct, _ := lib.AddTemplateUser(templateId, customer.ID)
	if tct > 0 {
		if err = template.IncrUseTimes(); err != nil {
			logApi.Errorf("[Mysql] incr template: %d use times failed: %s", templateId, err)
		}
	}

	c.JSON(http.StatusOK, Response{SUCCESS, task.ID})
}

// 查看写真任务状态
func GetPhotoStatus(c *gin.Context) {
	tid, _ := strconv.Atoi(c.Query("id"))
	if tid == 0 {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}
	cusId := GetUserID(c)

	task := &models.UserPhotoTask{ID: tid}
	if err := task.GetByID(); err != nil {
		logApi.Errorf("[Mysql] get photo task failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "状态获取失败"})
		return
	}
	if task.CusId != cusId {
		c.JSON(http.StatusOK, Response{FAILURE, "状态获取失败"})
		return
	}

	// 组装数据
	art, err := task.GetInfoByTaskID()
	if err != nil {
		logApi.Errorf("[Mysql] get photo task: %d failed: %s", tid, err)
		c.JSON(http.StatusOK, Response{FAILURE, "状态获取失败"})
		return
	}
	output := &models.UserPhotoImage{TaskId: task.ID}
	images, err := output.GetByTaskID()
	if err != nil {
		logApi.Errorf("[Mysql] get photo image failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "状态获取失败"})
		return
	}
	if len(images) == 0 {
		c.JSON(http.StatusOK, Response{FAILURE, "状态获取失败"})
		return
	}
	poseId := -1
	for _, image := range images {
		if image.PoseId != poseId {
			if poseId == -1 {
				poseId = image.PoseId
			} else {
				poseId = 0
			}
		}

		photo := models.UserPhotoImageWeb{
			ID:        image.ID,
			PoseId:    image.PoseId,
			ImgUrl:    image.ImgUrl,
			ThumbUrl:  image.ThumbUrl,
			EnableHr:  image.EnableHr,
			Favourite: image.Favourite,
		}
		if image.EnableHr {
			if image.HrImgUrl != "" {
				photo.ImgUrl = image.HrImgUrl
			} else {
				photo.Hiresing = true
			}
		}
		art.Photos = append(art.Photos, photo)
	}
	art.PoseId = poseId
	art.PoseImage = task.ControlImage

	c.JSON(http.StatusOK, Response{SUCCESS, art})
}

// 写真历史记录
func GetPhotoHistory(c *gin.Context) {
	// 校验用户
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "用户不存在"})
		return
	}

	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("size"))
	task := &models.UserPhotoTask{}
	list, err := task.GetHistory(customer.ID, page, pageSize)
	if err != nil {
		logApi.Errorf("[Mysql] get photo history failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "获取失败，请重试"})
		return
	}

	for _, t := range list {
		image := &models.UserPhotoImage{TaskId: t.ID}
		photos, err := image.GetByTaskIDWithoutImgUrl()
		if err != nil {
			logApi.Errorf("[Mysql] get photo image failed: %s", err)
			continue
		}
		poseId := -1
		for _, photo := range photos {
			if photo.PoseId != poseId {
				if poseId == -1 {
					poseId = photo.PoseId
				} else {
					poseId = 0
				}
			}

			photoWeb := models.UserPhotoImageWeb{
				ID:        photo.ID,
				PoseId:    photo.PoseId,
				ImgUrl:    photo.ImgUrl,
				ThumbUrl:  photo.ThumbUrl,
				EnableHr:  photo.EnableHr,
				Favourite: photo.Favourite,
			}
			if photo.EnableHr {
				if photo.HrImgUrl != "" {
					photo.ImgUrl = photo.HrImgUrl
				} else {
					photoWeb.Hiresing = true
				}
			}
			t.Photos = append(t.Photos, photoWeb)
		}
		t.PoseId = poseId
	}

	c.JSON(http.StatusOK, Response{SUCCESS, list})
}

// 图片高清化
func PhotoImageHigher(c *gin.Context) {
	// 校验用户
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "用户不存在"})
		return
	}

	imgId, _ := strconv.Atoi(c.Request.FormValue("id"))
	photo := &models.UserPhotoImage{ID: imgId}
	if err = photo.GetByID(); err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "写真不存在"})
		return
	}
	if photo.CusId != customer.ID {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}
	if photo.EnableHr {
		if photo.HrImgUrl != "" {
			c.JSON(http.StatusOK, Response{SUCCESS, photo.HrImgUrl})
			return
		}
		if time.Now().Unix()-photo.HiresAt < 120 {
			c.JSON(http.StatusOK, Response{SUCCESS, ""})
			return
		}
		if lib.CheckListExist(lib.RedisSDPhotoHrList, uint(photo.ID)) {
			c.JSON(http.StatusOK, Response{SUCCESS, ""})
			return
		}
		// 发送高清任务
		if err = lib.PushSDPhotoHrTask(photo.ID); err != nil {
			logApi.Errorf("[Redis] push photo hr task: %d failed: %s", photo.ID, err)
		}
		c.JSON(http.StatusOK, Response{SUCCESS, ""})
		return
	}

	// 检查钻石
	if customer.Diamond < DIAMOND_HIGHER {
		c.JSON(http.StatusOK, Response{DIAMOND_NOT_ENOUGH, "钻石不足"})
		return
	}

	// 高清处理
	if err = photo.UpdateEnableHr(customer.ID, customer.Diamond, DIAMOND_HIGHER); err != nil {
		c.JSON(http.StatusOK, Response{FAILURE, "高清处理失败"})
		return
	}

	// 发送高清任务
	if err = lib.PushSDPhotoHrTask(photo.ID); err != nil {
		logApi.Errorf("[Redis] push photo hr task: %d failed: %s", photo.ID, err)
	}

	c.JSON(http.StatusOK, Response{SUCCESS, photo.HrImgUrl})
}

// 删除写真
func DeletePhotoTask(c *gin.Context) {
	// 校验用户
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "用户不存在"})
		return
	}

	// 校验写真集ID
	taskId, _ := strconv.Atoi(c.Request.FormValue("id"))
	task := &models.UserPhotoTask{ID: taskId}
	if err = task.GetByID(); err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "写真集不存在"})
		return
	}
	if task.CusId != customer.ID {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}

	// 写真照片
	photo := &models.UserPhotoImage{TaskId: taskId}
	photos, err := photo.GetByTaskID()
	if err != nil {
		logApi.Warnf("[Mysql] get photo image by task: %d failed: %s", photo.TaskId, err)
	}

	// 删除写真
	if err = task.Delete(); err != nil {
		c.JSON(http.StatusOK, Response{FAILURE, "删除失败"})
		return
	}

	for _, p := range photos {
		if p.DownUrl == "" {
			// 如果任务未执行，删除任务
			lib.DelSDPhotoTask(p.ID)
		} else {
			imgs := make([]string, 0)
			imgs = append(imgs, p.ImgUrl)
			imgs = append(imgs, p.ThumbUrl)
			imgs = append(imgs, p.DownUrl)
			imgs = append(imgs, p.HrDownUrl)
			imgs = append(imgs, p.HrImgUrl)

			ct := &lib.UploadCDNTask{
				TaskType: lib.CDN_DELETE,
				DelPath:  imgs,
			}
			if err = lib.PushCDNTask(ct); err != nil {
				logApi.Warnf("[CDN] push delete url %s failed: %s", ct.DelPath, err)
			}
		}
	}

	c.JSON(http.StatusOK, Response{SUCCESS, ""})
}

// 收藏
func PhotoImageFavorite(c *gin.Context) {
	// 校验用户
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "用户不存在"})
		return
	}

	// 校验写真ID
	pid, _ := strconv.Atoi(c.Request.FormValue("id"))
	photo := &models.UserPhotoImage{ID: pid}
	if err = photo.GetByID(); err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "写真不存在"})
		return
	}
	if photo.CusId != customer.ID {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}

	// 收藏
	photo.Favourite = c.Request.FormValue("favorite") == "1"
	if photo.Favourite {
		photo.FavouriteAt = time.Now().Unix()
	} else {
		photo.FavouriteAt = 0
	}
	if err = photo.UpdateFavourite(); err != nil {
		c.JSON(http.StatusOK, Response{FAILURE, "操作失败"})
		return
	}
	c.JSON(http.StatusOK, Response{SUCCESS, ""})
}

// 我的收藏
func MyPhotoFavorite(c *gin.Context) {
	// 校验用户
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "用户不存在"})
		return
	}

	page, _ := strconv.Atoi(c.Query("page"))
	photo := &models.UserPhotoImage{}
	list, err := photo.GetFavourite(customer.ID, page)
	if err != nil {
		c.JSON(http.StatusOK, Response{FAILURE, "获取失败"})
		return
	}

	c.JSON(http.StatusOK, Response{SUCCESS, list})
}

// 下载写真图片
func DownloadPhotoImage(c *gin.Context) {
	// 校验用户
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "用户不存在"})
		return
	}

	// 检查写真
	photoId, _ := strconv.Atoi(c.Query("id"))
	photo := &models.UserPhotoImage{ID: photoId}
	if err = photo.GetByID(); err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "写真不存在"})
		return
	}
	if photo.CusId != customer.ID {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}

	// 检查钻石
	if customer.Diamond < DIAMOND_DOWNLOAD {
		c.JSON(http.StatusOK, Response{DIAMOND_NOT_ENOUGH, "钻石不足"})
		return
	}

	downUrl := photo.DownUrl
	if photo.EnableHr {
		downUrl = photo.HrDownUrl
	}

	desUrl, err := lib.DESEncrypt([]byte(downUrl), lib.ApiDeskey)
	if err != nil {
		c.JSON(http.StatusOK, Response{FAILURE, "下载失败"})
		return
	}

	// 扣除钻石
	if err = photo.Download(customer.ID, customer.Diamond, DIAMOND_DOWNLOAD); err != nil {
		c.JSON(http.StatusOK, Response{FAILURE, "钻石扣除失败"})
		return
	}

	c.JSON(http.StatusOK, Response{SUCCESS, desUrl})

	// c.Header("Content-Type", "application/octet-stream")
	// c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%d%s", photoId, fileExt))
	// c.Header("Content-Transfer-Encoding", "binary")
	// io.Copy(c.Writer, resp.Body)
}

// 分享写真图片
func SharePhotoImage(c *gin.Context) {
	// 校验用户
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "用户不存在"})
		return
	}
	photoId, _ := strconv.Atoi(c.Query("id"))
	ptype, _ := strconv.Atoi(c.Query("type"))

	retval := make(map[string]any)
	if photoId > 0 {

		var shareUrl string
		switch ptype {
		case 1:
			// 分身
			card := &models.UserCardImage{ID: photoId}
			if err = card.GetByID(); err != nil {
				c.JSON(http.StatusOK, Response{INVALID_PARAM, "分身不存在"})
				return
			}
			if card.CusId != customer.ID {
				c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
				return
			}
			shareUrl = card.ImgUrl
		case 2:
			// 写真
			photo := &models.UserPhotoImage{ID: photoId}
			if err = photo.GetByID(); err != nil {
				c.JSON(http.StatusOK, Response{INVALID_PARAM, "写真不存在"})
				return
			}
			if photo.CusId != customer.ID {
				c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
				return
			}
			shareUrl = photo.DownUrl
		}

		desUrl, err := lib.DESEncrypt([]byte(shareUrl), lib.ApiDeskey)
		if err != nil {
			c.JSON(http.StatusOK, Response{FAILURE, "分享失败"})
			return
		}
		retval["img_url"] = desUrl
	}

	retval["share_code"] = lib.ShareCode
	c.JSON(http.StatusOK, Response{SUCCESS, retval})
}
