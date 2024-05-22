package models

import (
	"database/sql"
	"sync"

	"camera-webui/logger"

	_ "github.com/mattn/go-sqlite3"
)

var (
	logSql = logger.New("logs/sql.log")

	db         *sql.DB
	NoRowError string = "sql: no rows in result set"

	l sync.Mutex
)

func init() {
	var err error
	db, err = sql.Open("sqlite3", "StableDiffusion.db")
	if err != nil {
		logSql.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		logSql.Fatal(err)
	}
}

type SDWork struct {
	ID           uint     // 任务ID
	JsonData     string   // 任务Json
	Callback     string   // 回调地址
	Status       int      // 0-待执行 1-执行中
	CreatedAt    int64    // 创建时间戳
	TaskPath     string   // 生图时的临时目录，用于删除
	ADModelPaths []string // 最终使用的AD模型路径，用于删除
}

// 创建SD任务
func (w *SDWork) Create() error {
	_, err := db.Exec("INSERT INTO sdwork(id,jsondata,callback,status,created_at) VALUES(?,?,?,?,?)", w.ID, w.JsonData, w.Callback, 0, w.CreatedAt)
	return err
}

// 是否存在
func (w *SDWork) Exist() (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(1) FROM sdwork").Scan(&count)
	return count > 0, err
}

// 获取任务
func (w *SDWork) GetWork() error {
	l.Lock()
	defer l.Unlock()

	err := db.QueryRow("SELECT id,jsondata,callback,status FROM sdwork WHERE status = 0 ORDER BY id asc").Scan(&w.ID, &w.JsonData, &w.Callback, &w.Status)
	if err == nil {
		err = w.UpdateStatus()
	}
	return err
}

// 更新任务状态
func (w *SDWork) UpdateStatus() error {
	_, err := db.Exec("UPDATE sdwork SET status = 1 WHERE id = ?", w.ID)
	return err
}

// 删除任务
func (w *SDWork) Delete() error {
	_, err := db.Exec("DELETE FROM sdwork WHERE id = ?", w.ID)
	return err
}

// 批量重置任务
func ResetSDWork() error {
	_, err := db.Exec("UPDATE sdwork SET status=0")
	return err
}

// 训练
type TrainWork struct {
	ID        uint   // 任务ID
	JsonData  string // 任务Json
	Callback  string // 回调地址
	Status    int    // 0-待执行 1-执行中
	CreatedAt int64  // 创建时间戳
	TaskPath  string //训练时的临时目录
}

// 创建SD任务
func (w *TrainWork) Create() error {
	_, err := db.Exec("INSERT INTO trainwork(id,jsondata,callback,status,created_at) VALUES(?,?,?,?,?)", w.ID, w.JsonData, w.Callback, 0, w.CreatedAt)
	return err
}

// 是否存在
func (w *TrainWork) Exist() (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(1) FROM trainwork").Scan(&count)
	return count > 0, err
}

// 获取任务
func (w *TrainWork) GetWork() error {
	l.Lock()
	defer l.Unlock()

	err := db.QueryRow("SELECT id,jsondata,callback,status FROM trainwork WHERE status = 0 ORDER BY id asc").Scan(&w.ID, &w.JsonData, &w.Callback, &w.Status)
	if err == nil {
		err = w.UpdateStatus()
	}
	return err
}

// 更新任务状态
func (w *TrainWork) UpdateStatus() error {
	_, err := db.Exec("UPDATE trainwork SET status = 1 WHERE id = ?", w.ID)
	return err
}

// 删除任务
func (w *TrainWork) Delete() error {
	_, err := db.Exec("DELETE FROM trainwork WHERE id = ?", w.ID)
	return err
}

// 批量重置任务
func ResetTrainWork() error {
	_, err := db.Exec("UPDATE trainwork SET status=0")
	return err
}

// 检查
type CheckWork struct {
	ID        uint   // 任务ID
	JsonData  string // 任务Json
	Callback  string // 回调地址
	Status    int    // 0-待执行 1-执行中
	CreatedAt int64  // 创建时间戳
	TaskPath  string // 检查临时目录
}

// 创建检查任务
func (w *CheckWork) Create() (int64, error) {
	res, err := db.Exec("INSERT INTO imgcheck(jsondata,callback,status,created_at) VALUES(?,?,?,?)", w.JsonData, w.Callback, 0, w.CreatedAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// 是否存在
func (w *CheckWork) Exist() (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(1) FROM imgcheck").Scan(&count)
	return count > 0, err
}

// 获取任务
func (w *CheckWork) GetWork() error {
	l.Lock()
	defer l.Unlock()

	err := db.QueryRow("SELECT id,jsondata,callback,status FROM imgcheck WHERE status = 0 ORDER BY id asc").Scan(&w.ID, &w.JsonData, &w.Callback, &w.Status)
	if err == nil {
		err = w.UpdateStatus()
	}
	return err
}

// 更新任务状态
func (w *CheckWork) UpdateStatus() error {
	_, err := db.Exec("UPDATE imgcheck SET status = 1 WHERE id = ?", w.ID)
	return err
}

// 删除任务
func (w *CheckWork) Delete() error {
	_, err := db.Exec("DELETE FROM imgcheck WHERE id = ?", w.ID)
	return err
}

// 批量重置任务
func ResetCheckWork() error {
	_, err := db.Exec("UPDATE imgcheck SET status=0")
	return err
}
