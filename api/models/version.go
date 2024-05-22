package models

type AppVersion struct {
	ID       int
	BundleID string
	Version  string
	IsForce  bool
	DownUrl  string
	DownMd5  string
}

// 根据BundleID获取
func (a *AppVersion) GetByBundleID() error {
	return db.Where("bundle_id = ? AND enabled = 1", a.BundleID).First(a).Error
}

// 获取列表
func (a *AppVersion) GetList() ([]AppVersion, error) {
	versions := []AppVersion{}
	err := db.Model(a).Where("enabled = 1").Find(&versions).Error
	return versions, err
}
