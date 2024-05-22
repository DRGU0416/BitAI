package models

import (
	"fmt"

	"gorm.io/gorm/clause"
)

type Reports struct {
	Repdate       string
	Mau           int
	DayAdd        int
	DayAct        int
	SecondAct     int
	SecondActPec  float64
	DaypayNum     int
	DaypayAmount  float64
	DayrpayNum    int
	DayrpayAmount float64
	Refund        float64
	TotalIncome   float64
}

// Add 新增报表
func (rep *Reports) Add() error {
	return db.Clauses(clause.OnConflict{UpdateAll: true}).Create(rep).Error
}

// GetOne 返回昨日数据新增用户
func (rep *Reports) GetOne(repdate string) error {
	return db.Where("repdate = ?", repdate).First(rep).Error
}

// RetentionReport 留存报表
type RetentionReport struct {
	Repdate    string
	DayAdd     int
	Secact_pec float64
	Thiact_pec float64
	Foract_pec float64
	Fifact_pec float64
	Sixact_pec float64
	Sevact_pec float64
}

func (rep *RetentionReport) Add() error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "repdate"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"day_add": rep.DayAdd}),
	}).Create(rep).Error
}

func (rep *RetentionReport) GetOne(repdate string) error {
	return db.Where("repdate = ?", repdate).First(rep).Error
}

func (rep *RetentionReport) Update(repdate string, day int, retention float64) error {
	var field string
	switch day {
	case 2:
		field = "secact_pec"
	case 3:
		field = "thiact_pec"
	case 4:
		field = "foract_pec"
	case 5:
		field = "fifact_pec"
	case 6:
		field = "sixact_pec"
	case 7:
		field = "sevact_pec"
	}

	return db.Exec(fmt.Sprintf(`UPDATE retention_report SET %s=? WHERE repdate=?`, field), retention, repdate).Error
}

/*
// ChannelReport 分渠道统计
type ChannelReport struct {
	RepDate      string `json:"rep_date"`
	Channel      string `json:"channel"`
	DayAct       int32  `json:"act_num"`
	DayAdd       int32  `json:"reg_num"`
	DayPay       int32  `json:"day_pay"`
	TotalPay     int32  `json:"total_pay"`
	DayPayerNum  int32  `json:"pay_num"`
	DayShare     int32  `json:"share_num"`
	IOSPay       int32  `json:"ios_pay"`
	AndroidPay   int32  `json:"android_pay"`
	DayPayAdd    int32  `json:"regpay_num"`
	DayPayAmount int32  `json:"regpay_amount"`
}

// Add 新增报表
func (rep *ChannelReport) Add() error {
	_, err := DB.Exec(`REPLACE INTO t_channel_reports(repdate,channel,dayact,dayadd,daypay,totalpay,daypay_num,dayshare,iospay,androidpay,daypay_add,daypay_amount) values(?,?,?,?,?,?,?,?,?,?,?,?)`,
		rep.RepDate, rep.Channel, rep.DayAct, rep.DayAdd, rep.DayPay, rep.TotalPay, rep.DayPayerNum, rep.DayShare, rep.IOSPay, rep.AndroidPay, rep.DayPayAdd, rep.DayPayAmount)
	return err
}

// GetOne 返回昨日数据
func (rep *ChannelReport) GetOne(repdate, channel string) (*ChannelReport, error) {
	model := ChannelReport{}
	err := DB.QueryRow(`SELECT dayadd FROM t_channel_reports WHERE repdate=? AND channel=?`, repdate, channel).Scan(&model.DayAdd)
	return &model, err
}

func (rep *ChannelReport) GetReport(start, end string) ([]ChannelReport, error) {
	list := make([]ChannelReport, 0)
	rows, err := DB.Query(`SELECT repdate,channel,dayact,dayadd,daypay,totalpay,daypay_num,dayshare,iospay,androidpay,daypay_add,daypay_amount FROM t_channel_reports WHERE repdate>=? AND repdate<=? AND channel=? LIMIT 1000`, start, end, rep.Channel)
	if err != nil {
		return list, err
	}
	defer rows.Close()

	for rows.Next() {
		report := ChannelReport{}
		err := rows.Scan(&report.RepDate, &report.Channel, &report.DayAct, &report.DayAdd, &report.DayPay, &report.TotalPay, &report.DayPayerNum, &report.DayShare, &report.IOSPay, &report.AndroidPay, &report.DayPayAdd, &report.DayPayAmount)
		if err != nil {
			return list, err
		}

		list = append(list, report)
	}
	return list, err
}

// SubscribeReport 订阅统计
type SubscribeReport struct {
	RepDate  string
	Week     int32
	Month    int32
	Season   int32
	HalfYear int32
	Year     int32
}

// Add 新增报表
func (rep *SubscribeReport) Add() error {
	_, err := DB.Exec(`REPLACE INTO t_report_subscribe(repdate,week_num,month_num,season_num,half_year,year_num) values(?,?,?,?,?,?)`,
		rep.RepDate, rep.Week, rep.Month, rep.Season, rep.HalfYear, rep.Year)
	return err
}

// RenewReport 续订统计
type RenewReport struct {
	RepDate          string
	Platform         string
	ProductId        string
	Total            int32
	Renew1Count      int32
	Renew1Percent    float64
	Renew2Count      int32
	Renew2Percent    float64
	Renew3Count      int32
	Renew3Percent    float64
	Renew4Count      int32
	Renew4Percent    float64
	Renew5Count      int32
	Renew5Percent    float64
	Renew6Count      int32
	Renew6Percent    float64
	RenewMoreCount   int32
	RenewMorePercent float64
}

func (rep *RenewReport) Add() error {
	_, err := DB.Exec(`REPLACE INTO t_report_renew(repdate,platform,product_id,total,renew_1_count,renew_1_percent,renew_2_count,renew_2_percent,renew_3_count,renew_3_percent,renew_4_count,renew_4_percent,renew_5_count,renew_5_percent,renew_6_count,renew_6_percent,renew_more_count,renew_more_percent) values(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		rep.RepDate, rep.Platform, rep.ProductId, rep.Total, rep.Renew1Count, rep.Renew1Percent, rep.Renew2Count, rep.Renew2Percent, rep.Renew3Count, rep.Renew3Percent, rep.Renew4Count, rep.Renew4Percent, rep.Renew5Count, rep.Renew5Percent, rep.Renew6Count, rep.Renew6Percent, rep.RenewMoreCount, rep.RenewMorePercent)
	return err
}

func (rep *RenewReport) GetOne(repdate string, productId string, platform string) (int, error) {
	var total int
	err := DB.QueryRow(`SELECT total FROM t_report_renew WHERE repdate=? AND product_id=? AND platform=?`, repdate, productId, platform).Scan(&total)
	return total, err
}

func (rep *RenewReport) Update(repdate string, productId string, platform string, day int, renewCount int32, renewPercent float64) error {
	var field string
	switch day {
	case 2:
		field = "renew_1"
	case 3:
		field = "renew_2"
	case 4:
		field = "renew_3"
	case 5:
		field = "renew_4"
	case 6:
		field = "renew_5"
	case 7:
		field = "renew_6"
	case 8:
		field = "renew_more"
	}

	_, err := DB.Exec(fmt.Sprintf(`UPDATE t_report_renew SET %s_count=?, %s_percent=? WHERE repdate=? AND platform=? AND product_id=?`, field, field), renewCount, renewPercent, repdate, platform, productId)
	return err
}
*/
