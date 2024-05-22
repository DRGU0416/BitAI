package models

import "gorm.io/gorm"

type UserPhotoTemplate struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Cover       string `json:"cover"`
	Lora        string `json:"-"`
	Width       int    `json:"-"`
	Height      int    `json:"-"`
	RandnSource string `json:"-"`
	MainModel   string `json:"-"`
}

// 根据ID获取
func (t *UserPhotoTemplate) GetByID() error {
	return db.Where("id = ? AND enabled = 1", t.ID).First(t).Error
}

// 分类素材列表
func (t *UserPhotoTemplate) List(page int) ([]UserPhotoTemplate, error) {
	images := []UserPhotoTemplate{}
	err := db.Model(t).Where("enabled = 1").Order("seq desc,id asc").Limit(20).Offset(page * 20).Find(&images).Error
	return images, err
}

// 更新使用次数
func (t *UserPhotoTemplate) IncrUseTimes() error {
	return db.Model(t).Where("id = ?", t.ID).UpdateColumn("used_num", gorm.Expr("used_num + 1")).Error
}

type UserPhotoPose struct {
	ID         int    `json:"id"`
	TemplateId int    `json:"-"`
	ImgUrl     string `json:"-"`
	ThumbUrl   string `json:"thumb_url"`
	PoseType   uint8  `json:"-"`

	//
	Prompt         string  `json:"-"`
	NegativePrompt string  `json:"-"`
	LoraWeight     float64 `json:"-"`
	LoraWeightStep float64 `json:"-"`
	SamplerName    string  `json:"-"`
	Steps          int     `json:"-"`
	CfgScale       float64 `json:"-"`
	Seed           int64   `json:"-"`
	EnableHr       bool    `json:"-"`

	// ControlNet0
	EnableControlNet bool    `json:"-"`
	ControlImage     string  `json:"-"`
	ControlWeight    float64 `json:"-"`
	EndControlStep   float64 `json:"-"`
	ControlMode      string  `json:"-"`
	ResizeMode       string  `json:"-"`
	Preprocessor     string  `json:"-"`
	ControlModel     string  `json:"-"`
	PixelPerfect     bool    `json:"-"`

	// ControlNet1
	ControlImage1   string  `json:"-"`
	ControlWeight1  float64 `json:"-"`
	EndControlStep1 float64 `json:"-"`
	ControlMode1    string  `json:"-"`
	ResizeMode1     string  `json:"-"`
	Preprocessor1   string  `json:"-"`
	ControlModel1   string  `json:"-"`
	PixelPerfect1   bool    `json:"-"`

	// Roop
	EnableRoop             bool    `json:"-"`
	FaceRestorerVisibility float64 `json:"-"`
	FaceRestorerName       string  `json:"-"`

	// ADetailer
	AdModel             string  `json:"-"`
	AdPrompt            string  `json:"-"`
	AdNegativePrompt    string  `json:"-"`
	AdDenoisingStrength float64 `json:"-"`
	AdLoraWeight        float64 `json:"-"`
	AdLoraWeightStep    float64 `json:"-"`
	AdConfidence        float64 `json:"-"`
	AdDilateErode       int     `json:"-"`
}

// 创建
func (p *UserPhotoPose) List(id int) ([]UserPhotoPose, error) {
	images := []UserPhotoPose{}
	err := db.Model(p).Where("template_id = ? AND enabled = 1", id).Find(&images).Error
	return images, err
}

// 根据ID获取
func (p *UserPhotoPose) GetByID() error {
	return db.Where("id = ? AND enabled = 1", p.ID).First(p).Error
}
