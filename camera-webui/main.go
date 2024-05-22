package main

import (
	"fmt"
	"sync"

	_ "camera-webui/config"
	"camera-webui/routes"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
)

var ws = new(sync.WaitGroup)

func initEcho(e *echo.Echo) {
	e.HideBanner = true
	e.Debug = viper.GetString("loglev") == "debug"
	e.Use(middleware.Recover())
	fmt.Println("echo Version: " + echo.Version)

	e.Static("/image", "work")

	writer := &lumberjack.Logger{
		Filename:   "logs/echo.log",
		MaxSize:    4,
		MaxBackups: 2,
		LocalTime:  true,
	}
	e.Logger.SetOutput(writer)

}

func main() {
	e := echo.New()
	initEcho(e)

	// 任务
	routes.Task(e)

	port := viper.GetString("port")
	e.Logger.Fatal(e.Start(":" + port))
}
