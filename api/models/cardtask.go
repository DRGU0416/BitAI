package models

import (
	"time"

	"gorm.io/gorm"
)

type TaskStatus uint8

const (
	DEFAULT TaskStatus = 0
	RUNNING TaskStatus = 1
	SUCCESS TaskStatus = 2
	CANCELD TaskStatus = 3
	FAILED  TaskStatus = 4
)

const (
	CARD_STEP_FRONT  = 0 // 上传正面照
	CARD_STEP_SIDE   = 1 // 上传侧面照
	CARD_STEP_MAKING = 2 // 制作中
	CARD_STEP_SELECT = 3 // 选择分身
	CARD_STEP_OK     = 4 // 完成
)

type UserCardTask struct {
	ID           int
	CusId        int
	Status       TaskStatus
	Reason       string
	FrontUrl     string
	Gender       int
	StartTime    time.Time
	CompleteTime time.Time
	SdAccId      int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// 创建
func (c *UserCardTask) Create() error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(c).Error; err != nil {
			return err
		}
		if err := tx.Table("user_account").Where("id = ?", c.CusId).Updates(UserAccount{
			TempCardTaskId: c.ID,
			Step:           CARD_STEP_MAKING,
		}).Error; err != nil {
			return err
		}

		return nil
	})
}

// 根据ID获取Model
func (c *UserCardTask) GetByID() error {
	return db.Where("id = ?", c.ID).First(c).Error
}

// 是否存在任务
func (c *UserCardTask) GetOneByCusId() error {
	return db.Where("cus_id = ?", c.CusId).First(c).Error
}

// 更新SD账号绑定
func (t *UserCardTask) UpdateSdAccId(accId int) error {
	return db.Model(t).Updates(UserCardTask{SdAccId: accId, StartTime: time.Now()}).Error
}

// 更新任务状态
func (t *UserCardTask) UpdateStatus(status TaskStatus, data string) error {
	switch status {
	case RUNNING:
		return db.Model(t).Updates(UserCardTask{Status: status, StartTime: time.Now()}).Error
	case CANCELD, FAILED:
		return db.Model(t).Select("status", "reason").Updates(UserCardTask{Status: status, Reason: data}).Error
	case SUCCESS:
		return db.Model(t).Select("status", "complete_time").Updates(UserCardTask{Status: status, CompleteTime: time.Now()}).Error
	}
	return nil
}

// 更新性别
func (t *UserCardTask) UpdateGender() error {
	return db.Model(t).Updates(UserCardTask{Gender: t.Gender}).Error
}

// 正面照
type UserFrontImage struct {
	ID       int    `json:"-"`
	CusId    int    `json:"-"`
	ImgUrl   string `json:"-"`
	ThumbUrl string `json:"img_url"`
	Status   int    `json:"status"`
}

// 创建
func (c *UserFrontImage) Create() error {
	return db.Create(c).Error
}

// 更新
func (c *UserFrontImage) Update() error {
	return db.Model(c).Select("img_url", "thumb_url", "status").Where("cus_id = ?", c.CusId).Updates(c).Error
}

// 根据ID获取
func (c *UserFrontImage) GetByID() error {
	return db.Where("id = ?", c.ID).Take(c).Error
}

// 根据CusId获取
func (c *UserFrontImage) GetByCusId() error {
	return db.Where("cus_id = ?", c.CusId).Take(c).Error
}

// 获取待识别图片
func (c *UserFrontImage) GetWaitJudge() ([]*UserFrontImage, error) {
	var images []*UserFrontImage
	if err := db.Where("status = 0").Order("id asc").Limit(20).Find(&images).Error; err != nil {
		return nil, err
	}
	return images, nil
}

// 更新状态
func (c *UserFrontImage) UpdateStatus(status int) error {
	return db.Model(c).Update("status", status).Error
}

// 侧面照
type UserInputImage struct {
	ID       int    `json:"id"`
	CusId    int    `json:"-"`
	ImgUrl   string `json:"-"`
	ThumbUrl string `json:"img_url"`
	Status   int    `json:"status"`
}

// 创建
func (c *UserInputImage) Create() error {
	return db.Create(c).Error
}

// 获取待识别图片
func (c *UserInputImage) GetWaitJudge() ([]*UserInputImage, error) {
	if err := db.Where("status = 0").Order("id asc").First(c).Error; err != nil {
		return nil, err
	}

	var images []*UserInputImage
	if err := db.Where("cus_id = ? AND status = 0", c.CusId).Find(&images).Error; err != nil {
		return nil, err
	}
	return images, nil
}

// 更新状态
func (c *UserInputImage) UpdateStatus(status int) error {
	return db.Model(c).Update("status", status).Error
}

// 根据ID获取
func (c *UserInputImage) GetByID() error {
	return db.Where("id = ?", c.ID).Take(c).Error
}

// 删除
func (c *UserInputImage) Delete() error {
	return db.Delete(c).Error
}

// 删除识别失败照片
func (c *UserInputImage) DeleteFail(cusId int) error {
	return db.Where("cus_id = ? AND status = 4", cusId).Delete(c).Error
}

// 获取Model
func (c *UserInputImage) GetByCusId() ([]*UserInputImage, error) {
	var cards []*UserInputImage
	err := db.Where("cus_id = ?", c.CusId).Find(&cards).Error
	return cards, err
}

type UserCardImage struct {
	ID               int     `json:"id"`
	CusId            int     `json:"-"`
	TaskId           int     `json:"-"`
	ImgUrl           string  `json:"img_url"`
	Lora             string  `json:"-"`
	Weight           float64 `json:"-"`
	PromptWeight     float64 `json:"-"`
	SecondGeneration bool    `json:"-"`
	Seed             int64   `json:"-"`
}

// 创建
func (c *UserCardImage) Create() error {
	return db.Create(c).Error
}

// 根据ID获取Model
func (c *UserCardImage) GetByID() error {
	return db.Where("id = ?", c.ID).First(c).Error
}

// 返回任务生成图片给前端
func (i *UserCardImage) GetByTaskID() ([]*UserCardImage, error) {
	var images []*UserCardImage
	if err := db.Where("task_id = ?", i.TaskId).Find(&images).Error; err != nil {
		return nil, err
	}
	return images, nil
}

// 更新分身图片
func (i *UserCardImage) UpdateImageUrl() error {
	return db.Model(i).Select("img_url", "seed").Updates(UserCardImage{
		ImgUrl: i.ImgUrl,
		Seed:   i.Seed,
	}).Error
}

// 根据用户ID获取分身
func (i *UserCardImage) GetByCusId(cusId int) error {
	sql := `SELECT b.*
			FROM user_account AS a
			INNER JOIN user_card_image AS b ON a.avatar_id = b.id
			WHERE a.id = ?`
	return db.Raw(sql, cusId).Scan(i).Error
}

type UserCardStatus struct {
	ID      int    `json:"id"`
	ImgUrl  string `json:"img_url"`
	Checked bool   `json:"checked"`
}

// 返回任务生成图片给前端
func (i *UserCardImage) GetByTaskIDForWeb(cardId int) ([]UserCardStatus, error) {
	rows, err := db.Table("user_card_image").Select("id", "img_url").Where("task_id = ?", i.TaskId).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]UserCardStatus, 0)
	for rows.Next() {
		res := UserCardStatus{}
		rows.Scan(&res.ID, &res.ImgUrl)
		res.Checked = res.ID == cardId
		records = append(records, res)
	}
	return records, nil
}
