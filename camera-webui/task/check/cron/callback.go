package cron

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"camera-webui/lib"
	"camera-webui/models"
)

var (
	cwch   = make(chan lib.CheckCallback, 100)
	client = &http.Client{Timeout: time.Second * 10}

	// 回调重试次数
	retryCT = 3
)

// 任务失败回调
func checkFailed(ckWork *models.CheckWork, code RespCode, msg string) {
	cb := lib.CheckCallback{
		TaskId:   ckWork.ID,
		Code:     int(code),
		Message:  msg,
		Callback: ckWork.Callback,
	}
	cwch <- cb
	if err := ckWork.Delete(); err != nil {
		logTask.Errorf("删除任务: %d 失败, %s", ckWork.ID, err)
	}

	DeleteTaskPath(ckWork)
}

// 任务成功回调
func checkSuccess(ckWork *models.CheckWork, imagesStatus map[uint64]int, allOk bool) {
	logTask.Infof("check任务: %d 成功", ckWork.ID)
	cb := lib.CheckCallback{
		TaskId:   ckWork.ID,
		Code:     int(SUCCESS),
		Status:   imagesStatus,
		Callback: ckWork.Callback,
	}
	cwch <- cb
	if err := ckWork.Delete(); err != nil {
		logTask.Errorf("删除任务: %d 失败, %s", ckWork.ID, err)
	}
	if allOk {
		MoveClipImages(ckWork)
	}
	DeleteTaskPath(ckWork)
}

// 回调
func CallbackTask(ctx context.Context, ws *sync.WaitGroup) {
	defer ws.Done()

	for {
		select {
		case <-ctx.Done():
			logTask.Debug("stop callback")
			for {
				select {
				case cb := <-cwch:
					runCallback(cb)
				case <-time.After(time.Second * 3):
					return
				}
			}
		case cb := <-cwch:
			runCallback(cb)
		}
	}
}

func runCallback(cb lib.CheckCallback) {
	byteMsg, err := json.Marshal(cb)
	if err != nil {
		logTask.Errorf("json序列化失败, %s, %+v", err, cb)
		return
	}
	body, err := lib.DESEncrypt(byteMsg, lib.WebUIDeskey)
	if err != nil {
		logTask.Errorf("3DES加密失败, %s, %+v", err, cb)
		return
	}

	request, _ := http.NewRequest("POST", cb.Callback, bytes.NewReader([]byte(body)))
	request.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(request)
	if err != nil {
		logTask.Errorf("回调失败, %s, %+v", err, cb)
		if cb.FCT+1 < retryCT {
			cb.IncrFCT()
			cwch <- cb
		}
		time.Sleep(errorSleep)
		return
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		logTask.Errorf("回调失败, %s, %+v", err, cb)
		if cb.FCT+1 < retryCT {
			cb.IncrFCT()
			cwch <- cb
		}
		time.Sleep(errorSleep)
		return
	}

	data := Response{}
	if err := json.Unmarshal(result, &data); err != nil {
		logTask.Errorf("回调解析失败, %s, %+v", err, cb)
		if cb.FCT+1 < retryCT {
			cb.IncrFCT()
			cwch <- cb
		}
		time.Sleep(errorSleep)
		return
	}
	if data.Code != SUCCESS {
		logTask.Errorf("回调失败, %s, %+v", data.Data.(string), cb)
		if cb.FCT+1 < retryCT {
			cb.IncrFCT()
			cwch <- cb
		}
		time.Sleep(errorSleep)
		return
	}
	logTask.Infof("check回调: %d 成功", cb.TaskId)
}
