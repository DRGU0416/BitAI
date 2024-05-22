package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type UserPhotoTask struct {
	ID           int
	CusId        int
	TemplateId   int
	ControlImage string
	LikeMe       bool
	AvatarId     int
	Status       TaskStatus
	Reason       string
	StartTime    time.Time
	CompleteTime time.Time
	SdAccId      int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// 创建
func (c *UserPhotoTask) Create() error {
	return db.Create(c).Error
}

// 根据ID获取
func (c *UserPhotoTask) GetByID() error {
	return db.Where("id = ?", c.ID).First(c).Error
}

// 设置SD账号绑定
func (t *UserPhotoTask) UpdateSdAccId(accId int) error {
	return db.Model(t).Updates(UserPhotoTask{SdAccId: accId, StartTime: time.Now()}).Error
}

// 更新状态
func (t *UserPhotoTask) UpdateStatus(status TaskStatus, data string) error {
	switch status {
	case RUNNING:
		return db.Model(t).Select("status", "start_time").Updates(UserPhotoTask{Status: status, StartTime: time.Now()}).Error
	case CANCELD, FAILED:
		return db.Model(t).Select("status", "reason").Updates(UserCardTask{Status: status, Reason: data}).Error
	case SUCCESS:
		return db.Model(t).Select("status", "complete_time").Updates(UserCardTask{Status: status, CompleteTime: time.Now()}).Error
	}
	return nil
}

// 删除
func (t *UserPhotoTask) Delete() error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("task_id", t.ID).Delete(UserPhotoImage{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(t).Error; err != nil {
			return err
		}

		return nil
	})
}

type UserPhotoHistory struct {
	ID           int                 `json:"id"`
	TemplateId   int                 `json:"template_id"`
	TemplateName string              `json:"template_name"`
	PoseId       int                 `json:"pose_id"`
	LikeMe       bool                `json:"like_me"`
	UsedNum      int                 `json:"used_num"`
	PoseImage    string              `json:"pose_image"`
	CreatedAt    JsonDate            `json:"created_at"`
	Photos       []UserPhotoImageWeb `json:"photos"`
}

// 获取写真任务详情
func (t *UserPhotoTask) GetInfoByTaskID() (*UserPhotoHistory, error) {
	history := &UserPhotoHistory{ID: t.ID}

	sql := `SELECT a.created_at, a.like_me, b.id as template_id, b.title, b.used_num
			FROM user_photo_task AS a
			INNER JOIN user_photo_template AS b ON a.template_id = b.id
			WHERE a.id = ?`
	err := db.Raw(sql, t.ID).Row().Scan(&history.CreatedAt, &history.LikeMe, &history.TemplateId, &history.TemplateName, &history.UsedNum)
	return history, err
}

// 写真列表
func (i *UserPhotoTask) GetHistory(cusId, page, pageSize int) ([]*UserPhotoHistory, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}

	sql := `SELECT a.id, a.like_me, a.created_at, b.id as template_id, b.title
			FROM user_photo_task AS a
			INNER JOIN user_photo_template AS b ON a.template_id = b.id
			WHERE a.cus_id=? AND status < 3 ORDER BY a.id DESC LIMIT ? OFFSET ?`

	result := make([]*UserPhotoHistory, 0)
	rows, err := db.Raw(sql, cusId, pageSize, page*pageSize).Rows()
	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		res := UserPhotoHistory{}
		rows.Scan(&res.ID, &res.LikeMe, &res.CreatedAt, &res.TemplateId, &res.TemplateName)
		result = append(result, &res)
	}

	return result, err
}

type UserPhotoImage struct {
	ID          int    `json:"id"`
	CusId       int    `json:"-"`
	TaskId      int    `json:"-"`
	PoseId      int    `json:"pose_id"`
	HrImgUrl    string `json:"-"`
	ImgUrl      string `json:"img_url"`
	ThumbUrl    string `json:"thumb_url"`
	HrDownUrl   string `json:"-"`
	DownUrl     string `json:"-"`
	EnableHr    bool   `json:"enable_hr"`
	HiresAt     int64  `json:"-"`
	Favourite   bool   `json:"favourite"`
	FavouriteAt int64  `json:"-"`

	SecondGeneration bool    `json:"-"`
	LoraWeight       float64 `json:"-"`
	AdLoraWeight     float64 `json:"-"`
	Seed             int64   `json:"-"`
}

type UserPhotoImageWeb struct {
	ID        int    `json:"id"`
	PoseId    int    `json:"pose_id"`
	ImgUrl    string `json:"img_url"`
	ThumbUrl  string `json:"thumb_url"`
	EnableHr  bool   `json:"enable_hr"`
	Favourite bool   `json:"favourite"`
	Hiresing  bool   `json:"hiresing"`
}

// 创建
func (c *UserPhotoImage) Create() error {
	return db.Create(c).Error
}

// 根据ID获取
func (c *UserPhotoImage) GetByID() error {
	return db.Where("id = ?", c.ID).First(c).Error
}

// 根据任务ID获取
func (i *UserPhotoImage) GetByTaskID() ([]*UserPhotoImage, error) {
	var images []*UserPhotoImage
	if err := db.Where("task_id = ?", i.TaskId).Find(&images).Error; err != nil {
		return nil, err
	}
	return images, nil
}

func (i *UserPhotoImage) GetByTaskIDWithoutImgUrl() ([]*UserPhotoImage, error) {
	var images []*UserPhotoImage
	if err := db.Where("task_id = ?", i.TaskId).Find(&images).Error; err != nil {
		return nil, err
	}
	return images, nil
}

// 更新图片
func (i *UserPhotoImage) UpdateImageUrl() error {
	return db.Model(i).Updates(UserPhotoImage{
		ImgUrl:   i.ImgUrl,
		ThumbUrl: i.ThumbUrl,
		DownUrl:  i.DownUrl,
		Seed:     i.Seed,
	}).Error
}

// 更新高清图片
func (i *UserPhotoImage) UpdateHrImageUrl() error {
	return db.Model(i).Updates(UserPhotoImage{
		HrImgUrl:  i.HrImgUrl,
		HrDownUrl: i.HrDownUrl,
	}).Error
}

// 设置高清
func (i *UserPhotoImage) UpdateEnableHr(cusId, diamond, gap int) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(i).Updates(UserPhotoImage{
			EnableHr: true,
			HiresAt:  time.Now().Unix(),
		}).Error; err != nil {
			return err
		}
		if err := tx.Table("user_account").Where("id = ?", cusId).UpdateColumn("diamond", gorm.Expr(fmt.Sprintf("diamond - %d", gap))).Error; err != nil {
			return err
		}
		//插入钻石消耗记录
		record := &DiamondChangeRecord{
			CusId:     cusId,
			RecordId:  i.ID,
			EventId:   EVENT_PHOTO_HIGH,
			Gap:       -gap,
			Quantity:  diamond - gap,
			CreatedAt: time.Now(),
		}
		if err := tx.Create(record).Error; err != nil {
			return err
		}

		return nil
	})
}

// 下载图片，扣钻石
func (i *UserPhotoImage) Download(cusId, diamond, gap int) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("user_account").Where("id = ?", cusId).UpdateColumn("diamond", gorm.Expr(fmt.Sprintf("diamond - %d", gap))).Error; err != nil {
			return err
		}
		//插入钻石消耗记录
		record := &DiamondChangeRecord{
			CusId:     cusId,
			RecordId:  i.ID,
			EventId:   EVENT_PHOTO_DOWNLOAD,
			Gap:       -gap,
			Quantity:  diamond - gap,
			CreatedAt: time.Now(),
		}
		if err := tx.Create(record).Error; err != nil {
			return err
		}

		return nil
	})
}

// 设置收藏
func (i *UserPhotoImage) UpdateFavourite() error {
	return db.Model(i).Select("favourite", "favourite_at").Updates(UserPhotoImage{
		Favourite:   i.Favourite,
		FavouriteAt: i.FavouriteAt,
	}).Error
}

type UserFavouriteImage struct {
	ID           int    `json:"id"`
	TaskId       int    `json:"task_id"`
	ImgUrl       string `json:"img_url"`
	ThumbUrl     string `json:"thumb_url"`
	EnableHr     bool   `json:"enable_hr"`
	Favourite    bool   `json:"favourite"`
	TemplateName string `json:"template_name"`
}

// 我的收藏
func (i *UserPhotoImage) GetFavourite(cusId, page int) ([]UserFavouriteImage, error) {
	sql := `SELECT a.id,a.task_id,a.img_url,a.thumb_url,a.enable_hr,a.favourite,c.title
			FROM user_photo_image AS a
			INNER JOIN user_photo_pose AS b on a.pose_id=b.id
			INNER JOIN user_photo_template AS c on b.template_id=c.id
			WHERE a.cus_id = ? AND a.favourite = 1 ORDER BY a.favourite_at DESC LIMIT ? OFFSET ?`
	rows, err := db.Raw(sql, cusId, 10, page*10).Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]UserFavouriteImage, 0)
	for rows.Next() {
		var photo UserFavouriteImage
		rows.Scan(&photo.ID, &photo.TaskId, &photo.ImgUrl, &photo.ThumbUrl, &photo.EnableHr, &photo.Favourite, &photo.TemplateName)
		if !photo.EnableHr {
			photo.ImgUrl = ""
		}

		result = append(result, photo)
	}
	return result, err
}
