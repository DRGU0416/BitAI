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
	cwch   = make(chan lib.SDCallback, 100)
	client = &http.Client{Timeout: time.Second * 30}

	// 回调重试次数
	retryCT = 20
)

// 任务失败回调
func taskFailed(sdwork *models.SDWork, code RespCode, msg string) {
	cb := lib.SDCallback{
		TaskId:   sdwork.ID,
		Code:     int(code),
		Message:  msg,
		Callback: sdwork.Callback,
	}
	cwch <- cb
	if err := sdwork.Delete(); err != nil {
		logTask.Errorf("删除任务: %d 失败, %s", sdwork.ID, err)
	}

	DeleteADModelPaths(sdwork)
	DeleteTaskPath(sdwork)

}

// 任务成功回调
func taskSuccess(sdwork *models.SDWork, images, watermarks []string, seed int64) {
	logTask.Infof("gen任务: %d 成功", sdwork.ID)
	cb := lib.SDCallback{
		TaskId:     sdwork.ID,
		Code:       int(SUCCESS),
		Images:     images,
		WaterMarks: watermarks,
		Callback:   sdwork.Callback,
		Seed:       seed,
	}
	cwch <- cb
	if err := sdwork.Delete(); err != nil {
		logTask.Errorf("删除任务: %d 失败, %s", sdwork.ID, err)
	}
	DeleteADModelPaths(sdwork)
	DeleteTaskPath(sdwork)
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

func runCallback(cb lib.SDCallback) {
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
	logTask.Infof("gen回调: %d 成功", cb.TaskId)
}
