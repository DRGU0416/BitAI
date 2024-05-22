package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type UserAccount struct {
	ID             int
	CardId         string
	CardNum        string
	Mobile         string
	Avatar         string
	AvatarId       int
	Openudid       string
	Idfa           string
	Idfv           string
	OsType         string
	OsVer          string
	RegIp          string
	LoginIp        string
	AppVer         string
	Platform       string
	RegChannel     string
	LoginChannel   string
	RemainTimes    int
	Diamond        int
	MessageId      int
	Paid           bool
	NewUser        bool
	Enabled        bool
	Deleted        bool
	CardTaskId     int
	TempCardTaskId int
	FrontUrl       string
	Step           uint8
	LoginTime      time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// 新增
func (c *UserAccount) Create() error {
	if c.RegChannel == "" {
		c.RegChannel = "gw"
	}
	c.LoginIp = c.RegIp
	c.LoginChannel = c.RegChannel
	c.UpdatedAt = c.CreatedAt
	c.LoginTime = c.CreatedAt
	c.Enabled = true

	return db.Create(c).Error
}

// 通过ID查询
func (c *UserAccount) GetByID() error {
	return db.Where("id = ? AND deleted = 0", c.ID).First(c).Error
}

// 通过手机号查询
func (c *UserAccount) GetCustomerByPhone() error {
	return db.Where("mobile = ? AND deleted = 0", c.Mobile).First(c).Error
}

// 手机号是否存在
func (c *UserAccount) IsMobileExist() error {
	var cusId int
	return db.Model(c).Select("id").Where("mobile = ?", c.Mobile).Take(&cusId).Error
}

// 充值时间
func (c *UserAccount) PayEvent(event, diamond, cardTimes int) error {
	values := make(map[string]any)
	if diamond > 0 {
		values["diamond"] = gorm.Expr(fmt.Sprintf("diamond + %d", diamond))
	}
	if cardTimes > 0 {
		values["remain_times"] = gorm.Expr(fmt.Sprintf("remain_times + %d", cardTimes))
	}
	values["paid"] = true

	return db.Transaction(func(tx *gorm.DB) error {
		if len(values) > 0 {
			if err := tx.Model(c).Updates(values).Error; err != nil {
				return err
			}
		}

		//插入钻石获得记录
		if diamond > 0 {
			record := &DiamondChangeRecord{
				CusId:     c.ID,
				EventId:   event,
				Gap:       diamond,
				Quantity:  c.Diamond + diamond,
				CreatedAt: time.Now(),
			}
			if err := tx.Create(record).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// 扣除分身次数
func (c *UserAccount) DecrRemainTimes() error {
	return db.Model(c).UpdateColumn("remain_times", gorm.Expr("remain_times - 1")).Error
}

// 更新登录时间，版本号，IP
func (c *UserAccount) UpdateLogin() error {
	return db.Model(c).Updates(UserAccount{
		LoginTime:    time.Now(),
		AppVer:       c.AppVer,
		LoginIp:      c.LoginIp,
		LoginChannel: c.LoginChannel,
	}).Error
}
func (c *UserAccount) UpdateLoginTime(ip string) error {
	return db.Model(c).Updates(UserAccount{
		LoginTime: time.Now(),
		LoginIp:   ip,
	}).Error
}

// 更新STEP
func (c *UserAccount) UpdateStep() error {
	return db.Model(c).Update("step", c.Step).Error
}

// 完成分身任务
func (c *UserAccount) FinishCardTask() error {
	return db.Model(c).Updates(UserAccount{
		Step:       CARD_STEP_OK,
		CardTaskId: c.TempCardTaskId,
	}).Error
}

// 更新头像
func (c *UserAccount) UpdateAvatar() error {
	return db.Model(c).Updates(UserAccount{
		AvatarId: c.AvatarId,
		Avatar:   c.Avatar,
		FrontUrl: c.FrontUrl,
	}).Error
}

// 删除用户
func (c *UserAccount) Delete() error {
	return db.Model(c).Update("deleted", 1).Error
}

// 更新消息ID
func (c *UserAccount) UpdateMessageId() error {
	return db.Model(c).Update("message_id", c.MessageId).Error
}

// 分身重置
func (c *UserAccount) ResetUserCard() error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(c).Select("step").Update("step", 0).Error; err != nil {
			return err
		}

		// 删除正面照
		if err := tx.Table("user_front_image").Where("cus_id = ?", c.ID).Delete(UserFrontImage{}).Error; err != nil {
			return err
		}

		// 删除侧面照
		if err := tx.Table("user_input_image").Where("cus_id = ?", c.ID).Delete(UserInputImage{}).Error; err != nil {
			return err
		}

		return nil
	})
}

// ******** 报表统计 **********

// GetReportActive 活跃用户数
func (c *UserAccount) GetReportActive(start, end time.Time) (int, error) {
	var num int
	err := db.Raw(`SELECT COUNT(1) FROM user_account WHERE login_time>? AND login_time<?`, start, end).Scan(&num).Error
	return num, err
}

// GetReportDayAdd 日新增
func (c *UserAccount) GetReportDayAdd(start, end time.Time) (int, error) {
	var num int
	err := db.Raw(`SELECT COUNT(1) FROM user_account WHERE created_at>? AND created_at<?`, start, end).Scan(&num).Error
	return num, err
}

// GetReportSecondActive 次留
func (c *UserAccount) GetReportSecondActive(rstart, rend, lstart, lend time.Time) (int, error) {
	var num int
	err := db.Raw(`SELECT COUNT(1) FROM user_account WHERE created_at>? AND created_at<? AND login_time>? AND login_time<?`, rstart, rend, lstart, lend).Scan(&num).Error
	return num, err
}
