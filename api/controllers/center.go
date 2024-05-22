package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"camera/lib"
	"camera/models"

	"github.com/gin-gonic/gin"
)

// 用户进度
func GetUserProgress(c *gin.Context) {
	customer, err := GetUser(c)
	if err != nil {
		logApi.Errorf("get user failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "进度获取失败"})
		return
	}

	result := make(map[string]any)
	result["step"] = customer.Step
	// 查看队列排名
	if customer.Step == models.CARD_STEP_MAKING {
		rank := lib.GetSDTaskRank(customer.TempCardTaskId)
		result["rank"] = rank
		if rank > 0 {
			result["wait_time"] = (rank + 1) * 10
		} else {
			task := &models.UserCardTask{ID: customer.TempCardTaskId}
			if err := task.GetByID(); err != nil {
				logApi.Errorf("[Mysql] get card task: %d failed: %s", task.ID, err)
				result["wait_time"] = 10
			} else {
				minute := int(time.Now().Sub(task.StartTime).Minutes())
				if minute < 10 {
					result["wait_time"] = 10 - minute
				} else {
					result["wait_time"] = 1
				}
			}
		}
	}

	c.JSON(http.StatusOK, Response{SUCCESS, result})
}

// 用户注销
func Logout(c *gin.Context) {
	//校验用户
	customer, err := GetUser(c)
	if err != nil {
		if err.Error() == models.NoRowError {
			c.JSON(http.StatusOK, Response{FAILURE, "用户不存在"})
			return
		}
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}

	// 注销
	if err = customer.Delete(); err != nil {
		logApi.Errorf("[Mysql] delete user failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "注销失败"})
		return
	}
	lib.RDB.HDel(ctx, lib.RedisUserToken, fmt.Sprintf("%d", customer.ID))

	// 删除队列中的任务
	switch customer.Step {
	case models.CARD_STEP_MAKING:
	}

	c.JSON(http.StatusOK, Response{SUCCESS, ""})
}

// 系统消息
func SystemMessage(c *gin.Context) {
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}

	page, _ := strconv.Atoi(c.Query("page"))
	message := &models.SysMessage{}
	list, err := message.List(page)
	if err != nil {
		logApi.Errorf("[Mysql] get message list error: %s", err.Error())
	}
	if len(list) > 0 && list[0].ID > customer.MessageId {
		customer.MessageId = list[0].ID
		customer.UpdateMessageId()
	}
	c.JSON(http.StatusOK, Response{SUCCESS, list})
}

// 个人中心
func PersonalCenter(c *gin.Context) {
	customer, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误"})
		return
	}

	// 更新登录时间,后期可以设定一个专用API
	ip := c.ClientIP()
	if err = customer.UpdateLoginTime(ip); err != nil {
		logApi.Errorf("[Mysql] update login time failed: %s, IP: %s", err, ip)
	}

	data := make(map[string]any)
	data["headicon"] = customer.Avatar
	data["diamond"] = customer.Diamond
	data["remain_times"] = customer.RemainTimes
	data["give_times"] = false
	if customer.NewUser && !customer.Paid {
		data["give_times"] = true
	}

	// 分身创建时间
	if customer.AvatarId > 0 {
		card := &models.UserCardImage{ID: customer.AvatarId}
		err = card.GetByID()
		if err == nil {
			task := &models.UserCardTask{ID: card.TaskId}
			if err = task.GetByID(); err == nil {
				data["created_at"] = task.CompleteTime.Unix()
			}
		}
	}

	//未读消息数量
	message := &models.SysMessage{}
	msgcount, err := message.UnReadCount(customer.MessageId)
	if err != nil {
		logApi.Errorf("[Mysql] get message unread count error: %s", err.Error())
	}
	data["msgcount"] = msgcount
	c.JSON(http.StatusOK, Response{SUCCESS, data})
}

// 钻石变动记录
func DiamondChangeRecord(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	cusId := GetUserID(c)

	list, err := models.GetDiamondChangeList(cusId, page)
	if err != nil {
		logApi.Errorf("[Mysql] get diamond change list error: %s", err.Error())
		c.JSON(http.StatusOK, Response{FAILURE, "获取失败"})
		return
	}
	c.JSON(http.StatusOK, Response{SUCCESS, list})
}
