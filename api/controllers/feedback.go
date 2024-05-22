package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"camera/models"

	"github.com/gin-gonic/gin"
)

// 创建反馈
func CreateFeedback(c *gin.Context) {
	customre, err := GetUser(c)
	if err != nil {
		c.JSON(http.StatusOK, Response{FAILURE, "操作失败，请重试"})
		return
	}

	feedback := &models.UserFeedback{
		CusId:     customre.ID,
		Platform:  strings.ToLower(c.GetHeader("platform")),
		Contact:   c.PostForm("contact"),
		Content:   c.PostForm("content"),
		CreatedAt: models.JsonDate(time.Now()),
	}
	if feedback.Content == "" {
		c.JSON(http.StatusOK, Response{INVALID_PARAM, "参数错误2"})
		return
	}

	if err := feedback.Create(); err != nil {
		logApi.Errorf("[Mysql] create feedback failed: %s", err)
		c.JSON(http.StatusOK, Response{FAILURE, "操作失败，请重试"})
		return
	}

	c.JSON(http.StatusOK, Response{SUCCESS, ""})
}

// 反馈列表
func GetFeedbackList(c *gin.Context) {
	cusId := GetUserID(c)
	page, _ := strconv.Atoi(c.Query("page"))

	feedback := &models.UserFeedback{CusId: cusId}
	list, err := feedback.List(page)
	if err != nil {
		logApi.Errorf("[Mysql] get feedback failed: %s, cusid: %d", err, cusId)
		c.JSON(http.StatusOK, Response{FAILURE, "操作失败，请重试"})
		return
	}

	c.JSON(http.StatusOK, Response{SUCCESS, list})
}
