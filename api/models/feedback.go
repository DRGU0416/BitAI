package models

import "time"

type UserFeedback struct {
	ID        int      `json:"-"`
	CusId     int      `json:"-"`
	Platform  string   `json:"-"`
	Contact   string   `json:"contact"`
	Content   string   `json:"content"`
	Reply     string   `json:"reply"`
	CreatedAt JsonDate `json:"created_at"`
	UpdatedAt JsonDate `json:"updated_at"`
}

// 创建反馈
func (f *UserFeedback) Create() error {
	// return db.Create(f).Error
	sql := `INSERT INTO user_feedback (cus_id, platform, contact, content, reply, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	return db.Exec(sql, f.CusId, f.Platform, f.Contact, f.Content, f.Reply, time.Time(f.CreatedAt), time.Time(f.UpdatedAt)).Error
}

// 反馈列表
func (f *UserFeedback) List(page int) ([]UserFeedback, error) {
	var list []UserFeedback
	if err := db.Model(f).Where("cus_id = ?", f.CusId).Order("id desc").Offset(page * 10).Limit(10).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
