package models

import "time"

const (
	EVENT_PHOTO_HIGH       = 1 // 写真高清处理
	EVENT_PHOTO_DOWNLOAD   = 2 // 写真下载
	EVENT_PAYMENT_CARD     = 3 // 充值分身制作
	EVENT_CARD_SPEED       = 4 // 充值分身加速
	EVENT_RECHARGE_DIAMOND = 5 // 充值钻石
)

type DiamondChangeRecord struct {
	ID        int
	CusId     int
	RecordId  int
	EventId   int
	Gap       int
	Quantity  int
	CreatedAt time.Time
}

// 创建
func (c *DiamondChangeRecord) Create() error {
	return db.Create(c).Error
}

type DiamondChangeList struct {
	Gap       int      `json:"gap"`
	CreatedAt JsonDate `json:"created_at"`
	EventName string   `json:"event_name"`
}

// 根据CusId获取列表
func GetDiamondChangeList(cusId, page int) ([]DiamondChangeList, error) {
	rows, err := db.Model(&DiamondChangeRecord{}).
		Select("diamond_change_record.gap,diamond_change_record.created_at,events.event_name").
		Joins("inner join events on events.id = diamond_change_record.event_id").
		Where("diamond_change_record.cus_id = ?", cusId).Order("diamond_change_record.id desc").Limit(10).Offset(page * 10).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]DiamondChangeList, 0)
	for rows.Next() {
		res := DiamondChangeList{}
		rows.Scan(&res.Gap, &res.CreatedAt, &res.EventName)
		records = append(records, res)
	}
	return records, nil
}
