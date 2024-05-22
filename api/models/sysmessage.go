package models

type SysMessage struct {
	ID        int      `json:"-"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	CreatedAt JsonDate `json:"created_at"`
}

// 列表
func (m *SysMessage) List(page int) ([]SysMessage, error) {
	var msg []SysMessage
	err := db.Select("id", "title", "content", "created_at").Order("id desc").Offset(page * 10).Limit(10).Find(&msg).Error
	return msg, err
}

// 未读消息数
func (m *SysMessage) UnReadCount(id int) (int64, error) {
	var count int64
	err := db.Model(m).Where("id > ?", id).Count(&count).Error
	return count, err
}
