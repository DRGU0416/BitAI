package controllers

import (
	"net/http"
	"strconv"

	"camera/models"

	"github.com/gin-gonic/gin"
)

// 写真模板
func MaterialTemplate(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	photo := &models.UserPhotoTemplate{}
	temps, err := photo.List(page)
	if err != nil {
		logApi.Errorf("[Mysql] get photo template failed: %s", err)
	}
	c.JSON(http.StatusOK, Response{SUCCESS, temps})
}

// 写真造型
func MaterialPose(c *gin.Context) {
	tempId, _ := strconv.Atoi(c.Query("id"))
	pose := &models.UserPhotoPose{}
	poses, err := pose.List(tempId)
	if err != nil {
		logApi.Warnf("[Mysql] get pose failed: %s", err)
	}
	template := &models.UserPhotoTemplate{ID: tempId}
	if err = template.GetByID(); err != nil {
		logApi.Warnf("[Mysql] get template: %d failed: %s", template.ID, err)
	}

	result := struct {
		Template *models.UserPhotoTemplate `json:"template"`
		Poses    []models.UserPhotoPose    `json:"poses"`
	}{
		Template: template,
		Poses:    poses,
	}
	c.JSON(http.StatusOK, Response{SUCCESS, result})
}
