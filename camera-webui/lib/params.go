package lib

import (
	"camera-webui/logger"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	// WebUI
	WebUIHost          string
	WebUIDeskey        string
	WebUIThread        int
	WebUIWorkPath      string
	WebUITrainPath     string
	WebUICheckPath     string
	WebUIPhotoHrPath   string
	WebUILoraPath      string
	WebUILoraSavePath  string
	WebUIRoopModelPath string

	//CDN
	QiniuBucket    string
	QiniuAccessKey string
	QiniuSecretKey string
	QiniuHost      string

	//Baidu
	BaiduCheckApiKey    string
	BaiduCheckApiSecret string
	BaiduCheckQpslimit  int

	// 任务结束删除中间文件
	DeleteMidFile = true
	MaxWaitMinute = 20
	FolderCount   = 1000

	logApi = logger.New("logs/api.log")
)

func init() {
	initConfig()
	go dynamicConfig()
}

func initConfig() {
	// WebUI
	WebUIHost = viper.GetString("webui.host")
	WebUIDeskey = viper.GetString("webui.deskey")
	WebUIThread = viper.GetInt("webui.thread")
	WebUIWorkPath = viper.GetString("webui.work_path")
	WebUITrainPath = viper.GetString("webui.train_path")
	WebUICheckPath = viper.GetString("webui.check_path")
	WebUIPhotoHrPath = viper.GetString("webui.photohr_path")
	WebUILoraPath = viper.GetString("webui.lora_path")
	WebUILoraSavePath = viper.GetString("webui.lora_save_path")
	WebUIRoopModelPath = viper.GetString("webui.roop_model_path")

	//CDN
	QiniuBucket = viper.GetString("qiniu.bucket")
	QiniuAccessKey = viper.GetString("qiniu.access_key")
	QiniuSecretKey = viper.GetString("qiniu.secret_key")
	QiniuHost = viper.GetString("qiniu.host")

	DeleteMidFile = viper.GetBool("deletemidfile")
	MaxWaitMinute = viper.GetInt("maxWaitMinute")

	//百度图片审核
	BaiduCheckApiKey = viper.GetString("baidu.apikey")
	BaiduCheckApiSecret = viper.GetString("baidu.apisecret")
	BaiduCheckQpslimit = viper.GetInt("baidu.qpslimit")
}

func dynamicConfig() {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		logApi.Infoln("配置文件发生变更!")
		initConfig()
	})
}
