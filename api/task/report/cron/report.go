package cron

import (
	"fmt"
	"sync"
	"time"

	"camera/logger"
	"camera/models"

	"github.com/robfig/cron/v3"
)

var (
	logReport = logger.New("logs/report.log")
)

// 每日0:10，统计昨天报表
func SyncYesterdayReport() {
	c := cron.New()
	c.AddFunc("@daily", func() {
		now := time.Now()

		// 统计报表
		if err := beginReport(now); err != nil {
			logReport.Errorf("report yesterday %s", err)
		}

		// 留存
		if err := retentionReport(time.Now()); err != nil {
			logReport.Errorf("retention report %s", err)
		}
	})
	c.Start()
}

// 每隔1小时，统计今日报表
func SyncTodayReport(ws *sync.WaitGroup) {
	defer ws.Done()
	ticker := time.NewTicker(time.Hour)

	for range ticker.C {
		now := time.Now()
		if now.Hour() < 1 {
			continue
		}

		// 统计报表
		if err := beginReport(now.AddDate(0, 0, 1)); err != nil {
			logReport.Errorf("report today %s", err)
		}

		// 留存
		if err := retentionReport(now.AddDate(0, 0, 1)); err != nil {
			logReport.Errorf("retention report %s", err)
		}
	}
}

/**
 * 统计
 * @param now 截止时间
 * @param platform 平台
 */
func beginReport(now time.Time) error {
	now, err := time.ParseInLocation("2006-01-02", now.Format("2006-01-02"), time.Local)
	if err != nil {
		return fmt.Errorf("[Time] time parse failed: %s", err)
	}

	// 月活跃
	customer := &models.UserAccount{}
	monthAu, err := customer.GetReportActive(now.AddDate(0, -1, 0), now)
	if err != nil {
		return fmt.Errorf("[Mysql] get report month active user failed: %s", err)
	}

	// 日新增
	dayAdd, err := customer.GetReportDayAdd(now.AddDate(0, 0, -1), now)
	if err != nil {
		return fmt.Errorf("[Mysql] get report day add failed: %s", err)
	}

	// 当日活跃
	dayAct, err := customer.GetReportActive(now.AddDate(0, 0, -1), now)
	if err != nil {
		return fmt.Errorf("[Mysql] get report day active user failed: %s", err)
	}

	// 次留
	secAu, err := customer.GetReportSecondActive(now.AddDate(0, 0, -2), now.AddDate(0, 0, -1), now.AddDate(0, 0, -1), now)
	if err != nil {
		return fmt.Errorf("[Mysql] get report second active user failed: %s", err)
	}

	// 当日支付金额
	order := &models.RechargeRecord{}
	dayPay, err := order.GetReportPayAmount(now.AddDate(0, 0, -1), now)
	if err != nil {
		return fmt.Errorf("[Mysql] get report day pay amount failed: %s", err)
	}

	// 当日支付人数
	dayPayerNum, err := order.GetReportPayerNum(now.AddDate(0, 0, -1), now)
	if err != nil {
		return fmt.Errorf("[Mysql] get report day payer num failed: %s", err)
	}

	// 当日注册支付人数
	dayPayReg, err := order.GetReportPayerRegNum(now.AddDate(0, 0, -1), now)
	if err != nil {
		return fmt.Errorf("[Mysql] get report day payer reg num failed: %s", err)
	}

	// 当日注册支付金额
	dayAmountReg, err := order.GetReportPayerRegAmount(now.AddDate(0, 0, -1), now)
	if err != nil {
		return fmt.Errorf("[Mysql] get report day payer reg amount failed: %s", err)
	}

	// 总支付金额
	totalPay, err := order.GetReportIncome()
	if err != nil {
		return fmt.Errorf("[Mysql] get report day pay amount failed: %s", err)
	}

	report := &models.Reports{
		Repdate:       now.AddDate(0, 0, -1).Format("20060102"),
		Mau:           monthAu,
		DayAdd:        dayAdd,
		DayAct:        dayAct,
		SecondAct:     secAu,
		DaypayNum:     dayPayerNum,
		DaypayAmount:  dayPay,
		DayrpayNum:    dayPayReg,
		DayrpayAmount: dayAmountReg,
		TotalIncome:   totalPay,
	}

	// 次留率
	if secAu > 0 {
		yesRep := &models.Reports{}
		if err = yesRep.GetOne(now.AddDate(0, 0, -2).Format("20060102")); err != nil {
			if err.Error() != models.NoRowError {
				return fmt.Errorf("[Mysql] get yesterday report failed: %s", err)
			}
		} else {
			report.SecondActPec = float64(secAu) / float64(yesRep.DayAdd)
		}
	}

	if err = report.Add(); err != nil {
		return fmt.Errorf("[Mysql] add report failed: %s", err)
	}
	return nil
}

// 留存报表
func retentionReport(now time.Time) error {
	now, err := time.ParseInLocation("2006-01-02", now.Format("2006-01-02"), time.Local)
	if err != nil {
		return fmt.Errorf("[Time] time parse failed: %s", err)
	}

	// 日新增
	customer := &models.UserAccount{}
	dayAdd, err := customer.GetReportDayAdd(now.AddDate(0, 0, -1), now)
	if err != nil {
		return fmt.Errorf("[Mysql] get report day add failed: %s", err)
	}

	report := &models.RetentionReport{
		Repdate: now.AddDate(0, 0, -1).Format("20060102"),
		DayAdd:  dayAdd,
	}

	if err = report.Add(); err != nil {
		return fmt.Errorf("[Mysql] add retention report failed: %s", err)
	}

	// 2留
	if err = calcRetentionDetail(now, 2); err != nil {
		return err
	}

	// 3留
	if err = calcRetentionDetail(now, 3); err != nil {
		return err
	}

	// 4留
	if err = calcRetentionDetail(now, 4); err != nil {
		return err
	}

	// 5留
	if err = calcRetentionDetail(now, 5); err != nil {
		return err
	}

	// 6留
	if err = calcRetentionDetail(now, 6); err != nil {
		return err
	}

	// 7留
	if err = calcRetentionDetail(now, 7); err != nil {
		return err
	}

	return nil
}

// 计算留存
func calcRetentionDetail(now time.Time, day int) error {
	customer := &models.UserAccount{}
	activeNum, err := customer.GetReportSecondActive(now.AddDate(0, 0, -day), now.AddDate(0, 0, -(day-1)), now.AddDate(0, 0, -1), now)
	if err != nil {
		return fmt.Errorf("[Mysql] get report second active user failed: %s", err)
	}

	if activeNum > 0 {
		report := &models.RetentionReport{}
		repdate := now.AddDate(0, 0, -day).Format("20060102")
		if err = report.GetOne(repdate); err != nil {
			if err.Error() != models.NoRowError {
				return fmt.Errorf("[Mysql] get yesterday report failed: %s", err)
			}
		} else if report.DayAdd > 0 {
			retention := float64(activeNum) / float64(report.DayAdd)
			if err = report.Update(repdate, day, retention); err != nil {
				logReport.Errorf("[Mysql] update retention failed: %s, day: %d", err, day)
			}
		}
	}
	return nil
}
