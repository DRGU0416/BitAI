package cron

import (
	"camera/lib"
	"os"
	"path"
	"time"

	"github.com/robfig/cron/v3"
)

// 清理数据
func ClearData() {
	c := cron.New()
	c.AddFunc("1 0 * * *", func() {
		// 清除注册CardID的数据
		now := time.Now()
		yesterday := now.Add(time.Hour * -24)
		day := yesterday.Format("20060102")
		logOps.Debugf("[Redis] clear reg count: %s", day)
		if _, err := lib.DelRegCount(day); err != nil {
			logOps.Errorf("[Redis] del reg count error: %v", err)
		}
		// 清除用户写真任务数统计
		if err := lib.DelPhotoCount(day); err != nil {
			logOps.Errorf("[Redis] del photo count error: %v", err)
		}

		// 清除用户上传的原图,保留10天
		// removeTempFile("/data/service/api/camera/images/material", now.Format("0601"), 240)
		// removeTempFile(fmt.Sprintf("/data/service/api/camera/images/material/%s", now.Format("0601")), "", 240)
	})
	c.Start()
}

// 清除临时图片文件
func removeTempFile(dirpath, exclude string, maxhour float64) {
	entries, err := os.ReadDir(dirpath)
	if err != nil {
		return
	}

	now := time.Now()
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == exclude {
			continue
		}

		secpath := path.Join(dirpath, entry.Name())
		f, err := os.Stat(secpath)
		if err != nil {
			logOps.Errorf("[OS] bad sec path: %s", secpath)
			return
		}
		if now.Sub(f.ModTime()).Hours() < maxhour {
			continue
		}
		logOps.Debugf("[OS] remove path: %s", secpath)
		os.RemoveAll(secpath)
	}
}
