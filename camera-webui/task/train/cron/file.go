package cron

import (
	"camera-webui/lib"
	"camera-webui/models"
	"os"
)

func DeleteTaskPath(trainWork *models.TrainWork) {
	if lib.DeleteMidFile && len(trainWork.TaskPath) > 0 {
		if err := os.RemoveAll(trainWork.TaskPath); err != nil {
			logTask.Warningf("删除检查目录: %s 失败, %s", trainWork.TaskPath, err)
		}
		trainWork.TaskPath = ""
	}
}
