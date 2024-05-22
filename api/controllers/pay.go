package controllers

import (
	"net/http"
	"time"

	"camera/lib"
	"camera/models"
	"camera/monitor"

	"github.com/gin-gonic/gin"
)

// 苹果支付
func AppStoreConfirm(c *gin.Context) {
	// 校验用户
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "用户不存在"})
		return
	}

	// 校验参数
	bundleId := c.GetHeader("bundleid")
	orderNum := c.Request.FormValue("order_num")
	productId := c.Request.FormValue("product_id")
	receipt := c.Request.FormValue("receipt")

	if orderNum == "" || productId == "" || receipt == "" {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}

	// 校验产品
	product := &models.Product{ProductId: productId}
	if err := product.GetByProductID(); err != nil {
		logApi.Errorf("[Mysql] get product failed: %s, proid: %s", err, productId)
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "产品不存在"})
		return
	}

	// 限制重复订单
	locked, err := lib.LockOrder(orderNum)
	if err != nil {
		logApi.Errorf("[Redis] lock order: %s failed: %s", orderNum, err)
		c.JSON(http.StatusOK, Response{PAY_CONFIRM_RETRY, "订单校验失败"})
		return
	}
	if !locked {
		c.JSON(http.StatusOK, Response{FAILURE, "订单重复提交1"})
		return
	}
	defer lib.UnlockOrder(orderNum)

	// 校验订单
	order := &models.RechargeRecord{OrderNum: orderNum}
	if err := order.GetByOrderNum(); err != nil {
		if err.Error() != models.NoRowError {
			logApi.Errorf("[Mysql] get order: %s failed: %s", orderNum, err)
			c.JSON(http.StatusOK, Response{INVALID_PARAM, "订单不存在"})
			return
		} else {
			order.CusId = customer.ID
			order.OrderNum = orderNum
			order.ProductId = productId
			order.Amount = product.Price
			order.Diamond = product.Diamond
			order.CardTimes = product.CardTimes
			order.Sandbox = false
			order.PayType = product.PayType
			order.CreatedAt = time.Now()
			order.UpdatedAt = time.Now()
			if product.ProductType == 1 && customer.NewUser && !customer.Paid {
				order.Diamond += 10
				order.CardTimes += 1
			}
		}
	} else {
		if order.Status != 0 {
			c.JSON(http.StatusOK, Response{FAILURE, "订单重复提交"})
			return
		}
		if order.CusId != customer.ID {
			c.JSON(http.StatusOK, Response{FAILURE, "订单不匹配"})
			return
		}
	}

	// 请求AppStore
	sandbox := false
	respData, err := lib.ConfirmAppStorePay(receipt, sandbox, false, "")
	if err != nil {
		if order.ID == 0 {
			logOrder.Errorf("[Http] check appstore order failed: %s, cus_id: %d, trans_id: %s, pro_id: %s, receipt: %s", err, customer.ID, orderNum, productId, receipt)

			bark := monitor.Bark{Title: "苹果支付确认失败", Message: orderNum}
			bark.SendMessage(monitor.ORDER_PAY)

			// 超时补单
			if err = order.Create(receipt); err != nil {
				logOrder.Errorf("[Mysql] create order: %s failed: %s", orderNum, err)
			}
		}

		c.JSON(http.StatusOK, Response{PAY_CONFIRM_RETRY, "操作失败，请重试"})
		return
	}

	// 沙盒
	if respData.Status == 21007 {
		sandbox = true
		respData, err = lib.ConfirmAppStorePay(receipt, sandbox, false, "")
		if err != nil {
			logOrder.Errorf("[Http][Sandbox] check appstore order failed: %s, cus_id: %d, trans_id: %s, pro_id: %s, receipt: %s", err, customer.ID, orderNum, productId, receipt)

			c.JSON(http.StatusOK, Response{PAY_CONFIRM_RETRY, "操作失败，请重试"})
			return
		}
	}
	logApi.Debugf("%+v", respData)

	// 验证交易信息
	receiptLen := len(respData.LatestReceiptInfo)
	receiptInAppLen := len(respData.Receipt.InApp)
	if respData.Status != 0 || respData.Receipt.BundleID != bundleId || (receiptLen == 0 && receiptInAppLen == 0) {
		logOrder.Errorf("[Confirm] appstore confirm failed, bundle_id:%s, cusid: %d, trans_id: %s, proid: %s, receipt: %s", bundleId, customer.ID, orderNum, productId, receipt)
		c.JSON(http.StatusOK, Response{FAILURE, "订单验证失败"})
		return
	}

	// 合并排重
	for _, receiptInfo := range respData.Receipt.InApp {
		if CheckTransInfo(respData, receiptInfo.TransactionID, receiptInfo.ProductID) == false {
			respData.LatestReceiptInfo = append(respData.LatestReceiptInfo, receiptInfo)
		}
	}
	if CheckTransInfo(respData, orderNum, productId) == false {
		logOrder.Warnf("[Confirm] appstore confirm failed, cusid: %d, trans_id: %s, proid: %s, receipt: %s", customer.ID, orderNum, productId, receipt)
		c.JSON(http.StatusOK, Response{FAILURE, "订单验证失败"})
		return
	}

	// 校验成功
	if order.ID == 0 {
		order.Sandbox = sandbox
		order.Status = 1
		if err = order.Create(receipt); err != nil {
			logOrder.Errorf("[Mysql] create order: %s failed: %s", orderNum, err)
		}
	} else {
		if err = order.Update(sandbox, 1); err != nil {
			logOrder.Errorf("[Mysql] create order: %s failed: %s", orderNum, err)
		}
	}

	// 更新用户信息
	var event int
	diamond, cardTimes := product.Diamond, product.CardTimes
	switch product.ProductType {
	case 1:
		event = models.EVENT_PAYMENT_CARD
		// 新用户首次付费赠送10钻石和1次重置机会
		if customer.NewUser && !customer.Paid {
			diamond += 10
			cardTimes += 1
		}
	case 2:
		event = models.EVENT_CARD_SPEED
	case 3:
		event = models.EVENT_RECHARGE_DIAMOND
	}
	if err = customer.PayEvent(event, diamond, cardTimes); err != nil {
		logApi.Errorf("[Mysql] trans_id: %s update customer: %d diamond + %d remain_times + %d failed: %s", orderNum, customer.ID, product.Diamond, product.CardTimes, err)
	}

	c.JSON(http.StatusOK, Response{SUCCESS, ""})
}

// 验证交易信息
func CheckTransInfo(data *lib.AppStoreData, orderNum, productId string) bool {
	for i := len(data.LatestReceiptInfo) - 1; i >= 0; i-- {
		receipt := data.LatestReceiptInfo[i]
		if receipt.TransactionID == orderNum && receipt.ProductID == productId {
			return true
		}
	}
	return false
}
