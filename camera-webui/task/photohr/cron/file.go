package cron

import (
	"camera-webui/lib"
	"camera-webui/models"
	"os"
	"path/filepath"
	"regexp"
)

func DeleteTaskPath(ckWork *models.CheckWork) {
	if lib.DeleteMidFile && len(ckWork.TaskPath) > 0 {
		if err := os.RemoveAll(ckWork.TaskPath); err != nil {
			logTask.Warningf("删除检查目录: %s 失败, %s", ckWork.TaskPath, err)
		}
		ckWork.TaskPath = ""
	}
}

func MoveClipImages(ckWork *models.CheckWork) {
	if len(ckWork.TaskPath) > 0 {

		cachePath := filepath.Join(lib.WebUICheckPath, "cache")
		os.MkdirAll(cachePath, 0755)

		clipPath := filepath.Join(ckWork.TaskPath, "clip")
		filepaths, err := filepath.Glob(filepath.Join(clipPath, "*"))
		if err != nil {
			logTask.Errorf("clip转移 读取文件夹: %s, 失败:%s", clipPath, err)
			return
		}
		for _, path := range filepaths {
			if !lib.IsImageFile(path) {
				continue
			}
			oldName := filepath.Base(path)
			compileRegex := regexp.MustCompile(`face_\d+_(.*)`)
			matchArr := compileRegex.FindStringSubmatch(oldName)
			if len(matchArr) > 0 {
				oldName = matchArr[len(matchArr)-1]
				//使用名字前四位作为文件夹名
				folderName := oldName
				if len(folderName) > 4 {
					folderName = folderName[:4]
				}

				os.MkdirAll(filepath.Join(cachePath, folderName), 0755)
				newPath := filepath.Join(cachePath, folderName, oldName)
				err := lib.CopyFile(path, newPath, true)
				if err != nil {
					logTask.Errorf("clip转移失败: %s, 失败:%s", path, err)
				}
			}
		}
	}
}
