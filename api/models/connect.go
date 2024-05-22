package models

import (
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var (
	db *gorm.DB

	NoRowError string = "record not found"

	PageSize = 10
)

type JsonDate time.Time

func (t JsonDate) MarshalJSON() ([]byte, error) {
	timeStr := fmt.Sprintf("%d", time.Time(t).Unix())
	return []byte(timeStr), nil
}

func init() {
	PageSize = viper.GetInt("mysql.pagenum")

	logger := newGormLogger()
	var err error
	config := viper.GetStringMapString("mysql")
	uri := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local", config["user"], config["password"], config["host"], config["port"], config["dbname"])
	db, err = gorm.Open(mysql.Open(uri), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 logger,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		time.Sleep(time.Second * 5)
		log.Fatal(err)
	}

	DB, _ := db.DB()
	DB.SetMaxIdleConns(10)
	DB.SetMaxOpenConns(25)
	DB.SetConnMaxLifetime(110 * time.Second)

	err = DB.Ping()
	if err != nil {
		log.Fatal(err)
	}
}

func newGormLogger() logger.Interface {
	writer := &lumberjack.Logger{
		Filename:   "logs/mysql.log",
		MaxSize:    4,
		MaxBackups: 20,
		LocalTime:  true,
	}

	mlog := logger.New(log.New(writer, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             500 * time.Millisecond,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: true,
		Colorful:                  true,
	})
	return mlog
}
