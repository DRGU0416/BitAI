package models

type SdAccount struct {
	ID      int
	AccName string
	AppUrl  string
	Enabled bool
}

// 根据ID获取账号
func (c *SdAccount) GetByID() error {
	return db.Where("id = ?", c.ID).First(c).Error
}

// 获取所有账号
func GetSDAccounts() ([]SdAccount, error) {
	var accounts []SdAccount
	if err := db.Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

// 禁用账号
func (c *SdAccount) Disable() error {
	return db.Model(c).Update("enabled", false).Error
}
