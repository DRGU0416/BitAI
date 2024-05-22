package models

import (
	"time"

	"gorm.io/gorm"
)

type RechargeRecord struct {
	ID        int
	CusId     int
	OrderNum  string
	ProductId string
	Amount    float64
	Diamond   int
	CardTimes int
	Sandbox   bool
	PayType   uint8
	Status    uint8
	CreatedAt time.Time
	UpdatedAt time.Time
}

// 创建
func (c *RechargeRecord) Create(receipt string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(c).Error; err != nil {
			return err
		}
		orderReceipt := &RechargeReceipt{
			ID:      c.ID,
			Receipt: receipt,
		}
		if err := tx.Create(orderReceipt).Error; err != nil {
			return err
		}

		return nil
	})
}

// 更新
func (c *RechargeRecord) Update(sandbox bool, status uint8) error {
	return db.Model(c).Select("status", "sandbox").Updates(RechargeRecord{
		Sandbox: sandbox,
		Status:  status,
	}).Error
}

// 查询订单号是否存在
func (c *RechargeRecord) GetByOrderNum() error {
	return db.Where("order_num = ?", c.OrderNum).First(c).Error
}

type RechargeReceipt struct {
	ID      int
	Receipt string
}

// 创建
func (c *RechargeReceipt) Create() error {
	return db.Create(c).Error
}

// ******** 报表统计 **********

// GetReportPayerNum 当日支付人数
// 时间范围 [start,end)
func (c *RechargeRecord) GetReportPayerNum(start, end time.Time) (int, error) {
	var num int
	err := db.Raw(`SELECT COUNT(1) FROM recharge_record AS a INNER JOIN product AS b ON a.product_id=b.product_id WHERE a.sandbox=0 AND a.created_at>=? AND a.created_at<? AND a.amount > 0`, start, end).Scan(&num).Error
	return num, err
}

// GetReportPayAmount 支付金额
// 时间范围 [start,end)
func (c *RechargeRecord) GetReportPayAmount(start, end time.Time) (float64, error) {
	var num float64
	err := db.Raw(`SELECT IFNULL(SUM(a.amount),0) FROM recharge_record AS a INNER JOIN product AS b ON a.product_id=b.product_id WHERE a.sandbox=0 AND a.created_at>=? AND a.created_at<? AND a.amount > 0`, start, end).Scan(&num).Error
	return num, err
}

// GetReportIncome 支付总金额
// 时间范围 [start,end)
func (c *RechargeRecord) GetReportIncome() (float64, error) {
	var num float64
	err := db.Raw(`SELECT IFNULL(SUM(a.amount),0) FROM recharge_record AS a INNER JOIN product AS b ON a.product_id=b.product_id WHERE a.sandbox=0`).Scan(&num).Error
	return num, err
}

// GetReportPayerRegNum 当日注册支付人数
// 时间范围 [start,end)
func (c *RechargeRecord) GetReportPayerRegNum(start, end time.Time) (int, error) {
	var num int
	err := db.Raw(`SELECT COUNT(1) FROM recharge_record AS a INNER JOIN product AS b ON a.product_id=b.product_id INNER JOIN user_account as c ON a.cus_id=c.id WHERE a.sandbox=0 AND a.amount > 0 AND a.created_at>=? AND a.created_at<? AND c.created_at>=? AND c.created_at<?`, start, end, start, end).Scan(&num).Error
	return num, err
}

// GetReportPayerRegAmount 当日注册支付金额
// 时间范围 [start,end)
func (c *RechargeRecord) GetReportPayerRegAmount(start, end time.Time) (float64, error) {
	var num float64
	err := db.Raw(`SELECT IFNULL(SUM(a.amount),0) FROM recharge_record AS a INNER JOIN product AS b ON a.product_id=b.product_id INNER JOIN user_account as c ON a.cus_id=c.id WHERE a.sandbox=0 AND a.amount > 0 AND a.created_at>=? AND a.created_at<? AND c.created_at>=? AND c.created_at<?`, start, end, start, end).Scan(&num).Error
	return num, err
}
