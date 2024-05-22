package lib

import (
	"errors"
	"net/url"
	"strings"

	"github.com/spf13/viper"
)

var (
	ApiDeskey     string
	ImageHost     string
	GlobalSmsCode string
	PhotoLimit    int64
	ShareCode     string
	SDBaseModel   string

	BaiduThreadNum       int
	BaiduCensorThreadNum int
	BaiduImgCheck        bool
	BaiduTextCheck       bool

	//七牛云
	QiniuAccessKey string
	QiniuSecretKey string
	QiniuBucket    string
	QiniuHost      string

	// WebUI
	WebUICallback          string
	WebUICallbackCard      string
	WebUICallbackPhoto     string
	WebUICallbackPhotoHr   string
	WebUICallbackRecognize string
	WebUIDeskey            string

	//Web管理后台
	WebToken string
)

func init() {
	ApiDeskey = viper.GetString("common.deskey")

	// 图片通用域名
	ImageHost = viper.GetString("common.img_host")

	// 测试验证码
	GlobalSmsCode = viper.GetString("common.sms_code")

	// 分享二维码地址
	ShareCode = viper.GetString("common.share_code")

	// 训练和分身主模型
	SDBaseModel = viper.GetString("common.sd_base_model")
	if SDBaseModel == "" {
		SDBaseModel = "E:\\sdwebui\\stable-diffusion-webui\\models\\Stable-diffusion\\OnlyRealistic_v30.safetensors"
	}

	// 每人每日写真任务上限
	PhotoLimit = viper.GetInt64("common.photo_limit")
	if PhotoLimit <= 20 {
		PhotoLimit = 20
	}

	// 百度
	BaiduThreadNum = viper.GetInt("baidu.thread")
	BaiduCensorThreadNum = viper.GetInt("baidu.censor_thread")
	BaiduImgCheck = viper.GetBool("baidu.img_censor")
	BaiduTextCheck = viper.GetBool("baidu.text_censor")

	//七牛云
	QiniuAccessKey = viper.GetString("qiniu.access_key")
	QiniuSecretKey = viper.GetString("qiniu.secret_key")
	QiniuBucket = viper.GetString("qiniu.bucket")
	QiniuHost = viper.GetString("qiniu.host")

	// WebUI
	WebUICallback = viper.GetString("webui.callback")
	WebUICallbackCard = viper.GetString("webui.callback_card")
	WebUICallbackPhoto = viper.GetString("webui.callback_photo")
	WebUICallbackPhotoHr = viper.GetString("webui.callback_photo_hr")
	WebUICallbackRecognize = viper.GetString("webui.callback_recognize")
	WebUIDeskey = viper.GetString("webui.deskey")

	//Web管理后台
	WebToken = viper.GetString("web.token")
}

// 获取文件URL
func GetImageUrl(imgurl string) (string, error) {
	if imgurl == "" {
		return "", errors.New("image is empty")
	}

	var err error
	if !strings.HasPrefix(imgurl, "http") {
		imgurl, err = url.JoinPath(ImageHost, imgurl)
	}
	return imgurl, err
}
