package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	_ "camera/config"
	_ "camera/models"
	"camera/task/devops/cron"
)

var ws = new(sync.WaitGroup)

func main() {
	ws.Add(2)
	ctx, cancel := context.WithCancel(context.Background())

	// 头像上传CDN
	go cron.FileCDN(ctx, ws)

	// 监控SD任务
	go cron.MonitorSDTask(ctx, ws)

	// 清理数据
	go cron.ClearData()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(interrupt)
	<-interrupt
	cancel()

	ws.Wait()
}
