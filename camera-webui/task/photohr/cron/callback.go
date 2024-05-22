package cron

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"camera-webui/lib"
)

var (
	cwch   = make(chan lib.PhotoHrCallback, 100)
	client = &http.Client{Timeout: time.Second * 10}

	// 回调重试次数
	retryCT = 3
)

// 任务失败回调
func checkFailed(ckWork lib.TaskPhotoHr, code RespCode, msg string) {
	cb := lib.PhotoHrCallback{
		TaskId:   uint(ckWork.TaskId),
		Code:     int(code),
		Message:  msg,
		Callback: ckWork.Callback,
	}
	cwch <- cb
}

// 任务成功回调
func checkSuccess(ckWork lib.TaskPhotoHr, imageUrl, WImageUrl string) {
	cb := lib.PhotoHrCallback{
		TaskId:        uint(ckWork.TaskId),
		Code:          int(SUCCESS),
		ImageUrl:      imageUrl,
		WaterImageUrl: WImageUrl,
		Callback:      ckWork.Callback,
	}
	cwch <- cb
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

func runCallback(cb lib.PhotoHrCallback) {
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
	logTask.Infof("高清回调: %d 成功", cb.TaskId)
}
