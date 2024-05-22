package models

type Barks struct {
	ID    uint
	Token string
}

// 推送列表
func GetBarkTokens() ([]string, error) {
	var tokens []string
	err := db.Model(&Barks{}).Select("token").Where("enabled=1").Find(&tokens).Error
	return tokens, err
}
