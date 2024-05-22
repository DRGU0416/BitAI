package cron

import (
	"camera-webui/lib"
	"camera-webui/models"
	"os"
)

func DeleteTaskPath(sdWork *models.SDWork) {
	if lib.DeleteMidFile && len(sdWork.TaskPath) > 0 {
		if err := os.RemoveAll(sdWork.TaskPath); err != nil {
			logTask.Warningf("删除生成目录: %s 失败, %s", sdWork.TaskPath, err)
		}
		sdWork.TaskPath = ""
	}
}

func DeleteADModelPaths(sdWork *models.SDWork) {
	if len(sdWork.ADModelPaths) > 0 {
		for _, path := range sdWork.ADModelPaths {
			if err := os.RemoveAll(path); err != nil {
				logTask.Warningf("删除AD模型: %s 失败, %s", path, err)
			}
		}
		sdWork.ADModelPaths = nil
	}
}
