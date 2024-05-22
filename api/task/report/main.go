package main

import (
	"sync"

	_ "camera/config"
	_ "camera/models"
	"camera/task/report/cron"
)

var ws = new(sync.WaitGroup)

func main() {
	ws.Add(1)

	// 昨日统计
	go cron.SyncYesterdayReport()

	// 今日统计
	go cron.SyncTodayReport(ws)

	ws.Wait()
}
