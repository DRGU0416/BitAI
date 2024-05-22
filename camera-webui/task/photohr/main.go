package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	_ "camera-webui/config"
	"camera-webui/task/photohr/cron"
)

var (
	ws   = new(sync.WaitGroup)
	wscb = new(sync.WaitGroup)
)

func main() {
	ws.Add(1)
	ctx, cancel := context.WithCancel(context.Background())

	// stable-diffusion
	go cron.PhotoHrService(ctx, ws)

	// 回调
	ctxCB, cancelCB := context.WithCancel(context.Background())
	wscb.Add(1)
	go cron.CallbackTask(ctxCB, wscb)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(interrupt)
	<-interrupt
	cancel()
	ws.Wait()

	cancelCB()
	wscb.Wait()
}
