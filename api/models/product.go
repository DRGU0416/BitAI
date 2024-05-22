package models

type Product struct {
	ID          int     `json:"-"`
	ProductType uint8   `json:"product_type"`
	ProductId   string  `json:"product_id"`
	Price       float64 `json:"price"`
	BundleId    string  `json:"-"`
	PayType     uint8   `json:"pay_type"`
	Diamond     int     `json:"diamond"`
	CardTimes   int     `json:"card_times"`
	Enabled     bool    `json:"-"`
}

// 获取包名列表
func (p *Product) GetBundleIds() ([]string, error) {
	var bundleids []string
	err := db.Model(p).Distinct("bundle_id").Find(&bundleids).Error
	return bundleids, err
}

// 根据包名获取列表
func (p *Product) GetListByBundleId(bundleId string) ([]Product, error) {
	products := []Product{}
	err := db.Model(p).Where("enabled = 1 AND bundle_id = ?", bundleId).Order("price asc").Find(&products).Error
	return products, err
}

// 根据产品ID获取
func (p *Product) GetByProductID() error {
	return db.Where("product_id = ?", p.ProductId).First(p).Error
}
