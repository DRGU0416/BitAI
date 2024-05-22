package logger

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LogFormatter struct{}

func (s *LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := time.Now().Local().Format("2006-01-02 15:04:05.000")
	var file, function string
	var line int
	if entry.Caller != nil {
		file = filepath.Base(entry.Caller.File)
		line = entry.Caller.Line
		function = entry.Caller.Function
		if funcs := strings.SplitN(function, ".", 2); len(funcs) == 2 {
			function = funcs[1]
		}
	}
	msg := fmt.Sprintf("%s %s/%s:%d %s -- %s\n", timestamp, strings.ToUpper(entry.Level.String()[:1]), file, line, function, entry.Message)
	return []byte(msg), nil
}

func New(filename string) *logrus.Logger {
	writer := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    4,
		MaxBackups: 20,
		LocalTime:  true,
	}
	log := logrus.New()
	log.SetReportCaller(true)
	log.SetOutput(writer)
	log.SetFormatter(new(LogFormatter))

	switch viper.GetString("loglev") {
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	}
	return log
}
