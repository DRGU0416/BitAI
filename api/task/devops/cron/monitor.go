package cron

import (
	"context"
	"sync"
	"time"

	"camera/lib"
	"camera/monitor"
)

// 监控SD任务
func MonitorSDTask(ctx context.Context, ws *sync.WaitGroup) {
	defer ws.Done()

	wcs := new(sync.WaitGroup)
	ticker := time.NewTicker(time.Minute * 2)

	for {
		select {
		case <-ctx.Done():
			logOps.Debug("stop monitor")
			wcs.Wait()
			return
		case <-ticker.C:
			wcs.Add(1)
			runMonitor(wcs)
		}
	}
}

func runMonitor(wcs *sync.WaitGroup) {
	defer wcs.Done()

	kv, err := lib.GetTaskRecords()
	if err != nil {
		if err.Error() != lib.RedisNull {
			logOps.Errorf("[Redis] get task records failed: %v", err)
		}
		return
	}

	hasbark := false
	now := time.Now().Unix()
	for k, v := range kv {
		record := &lib.TaskRecord{}
		if err = json.UnmarshalFromString(v, record); err != nil {
			logOps.Errorf("[Redis] unmarshal task record failed: %v", err)
			continue
		}

		if record.ExpiredAt > 0 && record.ExpiredAt < now {
			if record.TryTimes >= 2 {
				logOps.Warnf("%s 任务超时, ===== 丢弃 =====", k)
				// 超时报警
				if !hasbark {
					bark := monitor.Bark{Title: "任务超时", Message: k}
					bark.SendMessage(monitor.TASK_TIMEOUT)
					hasbark = true
				}
				if err = record.Delete(); err != nil {
					logOps.Errorf("[Redis] delete task record: %s failed: %v", k, err)
				}
			} else {
				logOps.Warnf("%s 任务超时, 重试", k)
				if err = record.ClearExpireAt(); err != nil {
					logOps.Errorf("[Redis] clear task record expireAt failed: %v", err)
					continue
				}

				// 重试
				switch record.TaskType {
				case lib.REC_FRONT:
					err = lib.PushSDCheckFrontTask([]int{record.TaskID})
				case lib.REC_SIDE:
					err = lib.PushSDCheckSideTask([]int{record.TaskID})
				case lib.REC_LORA:
					err = lib.PushSDTask(record.TaskID)
				case lib.REC_CARD:
					err = lib.PushSDCardTask(record.TaskID)
				case lib.REC_PHOTO:
					err = lib.PushSDPhotoTask(record.TaskID, false)
				case lib.REC_HR:
					err = lib.PushSDPhotoHrTask(record.TaskID)
				}
				if err != nil {
					logOps.Errorf("[Redis] push task: %s failed: %v", k, err)
				}
			}
		}
	}
}
