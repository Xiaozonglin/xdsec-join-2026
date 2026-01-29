package handlers

import (
	"encoding/json"
	"net/http"
	"xdsec-join-2026/auth"
	"xdsec-join-2026/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateApplicationRequest 创建申请请求
type CreateApplicationRequest struct {
	RealName   string   `json:"realName" binding:"required,max=10"`
	Phone      string   `json:"phone" binding:"required,max=15"`
	Gender     string   `json:"gender" binding:"required"`
	Department string   `json:"department" binding:"required,max=20"`
	Major      string   `json:"major" binding:"required,max=20"`
	StudentId  string   `json:"studentId" binding:"required,max=20"`
	Directions []string `json:"directions" binding:"required,max=30"`
	Resume     string   `json:"resume" binding:"required,max=10000"`
}

// CreateApplication 创建申请
func CreateApplication(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateApplicationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 兼容中文输入
		switch req.Gender {
		case "男":
			req.Gender = "male"
		case "女":
			req.Gender = "female"
		}

		// 验证性别
		if req.Gender != "male" && req.Gender != "female" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "后端不承认非二元性别"})
			return
		}

		// 验证方向
		if !auth.ValidateDirections(req.Directions) {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 获取当前用户
		userUUID, ok := GetCurrentUserUUID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "未登录"})
			return
		}

		// 序列化方向
		directionsJSON, _ := json.Marshal(req.Directions)

		// 检查是否已存在申请
		var existingApp models.Application
		if err := db.Where("user_id = ?", userUUID).First(&existingApp).Error; err == nil {
			// 申请已存在，检查是否有数据变化
			directionsMatch := existingApp.Directions == string(directionsJSON)
			dataChanged := existingApp.RealName != req.RealName ||
				existingApp.Phone != req.Phone ||
				existingApp.Gender != req.Gender ||
				existingApp.Department != req.Department ||
				existingApp.Major != req.Major ||
				existingApp.StudentId != req.StudentId ||
				!directionsMatch ||
				existingApp.Resume != req.Resume

			if !dataChanged {
				c.JSON(http.StatusTeapot, gin.H{"ok": false, "message": "申请数据未发生变化"})
				return
			}

			// 数据有变化，更新申请
			updates := map[string]interface{}{
				"real_name":  req.RealName,
				"phone":      req.Phone,
				"gender":     req.Gender,
				"department": req.Department,
				"major":      req.Major,
				"student_id": req.StudentId,
				"directions": string(directionsJSON),
				"resume":     req.Resume,
			}

			if err := db.Model(&existingApp).Updates(updates).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器更新申请时发生错误"})
				return
			}

			// 更新用户的方向信息
			if err := db.Model(&models.User{}).Where("uuid = ?", userUUID).Update("directions", string(directionsJSON)).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器更新用户方向时发生错误"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"ok": true, "message": "修改申请信息成功"})
			return
		}

		// 创建申请
		application := models.Application{
			RealName:   req.RealName,
			Phone:      req.Phone,
			Gender:     req.Gender,
			Department: req.Department,
			Major:      req.Major,
			StudentId:  req.StudentId,
			Directions: string(directionsJSON),
			Resume:     req.Resume,
			UserID:     userUUID,
		}

		if err := db.Create(&application).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器创建申请时发生错误"})
			return
		}

		// 更新用户的方向信息
		if err := db.Model(&models.User{}).Where("uuid = ?", userUUID).Update("directions", string(directionsJSON)).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器更新用户方向时发生错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// GetMyApplication 获取我的申请
func GetMyApplication(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取当前用户
		userUUID, ok := GetCurrentUserUUID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "未登录"})
			return
		}

		// 查找申请
		var application models.Application
		if err := db.Where("user_id = ?", userUUID).First(&application).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "你还没有提交申请吧"})
			return
		}

		appData := gin.H{
			"realName":   application.RealName,
			"phone":      application.Phone,
			"gender":     application.Gender,
			"department": application.Department,
			"major":      application.Major,
			"studentId":  application.StudentId,
			"resume":     application.Resume,
			"createdAt":  application.CreatedAt,
			"updatedAt":  application.UpdatedAt,
		}

		if application.Directions != "" {
			var directions []string
			json.Unmarshal([]byte(application.Directions), &directions)
			appData["directions"] = directions
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":   true,
			"data": appData,
		})
	}
}

// GetApplicationDetail 获取申请详情（面试官）
func GetApplicationDetail(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("userId")
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "你没有传 id 哦"})
			return
		}

		// 解析UUID
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "你传的 id 是 uuid 吗？"})
			return
		}

		// 查找申请
		var application models.Application
		if err := db.Where("user_id = ?", userUUID).First(&application).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "申请不存在"})
			return
		}

		appData := gin.H{
			"realName":   application.RealName,
			"phone":      application.Phone,
			"gender":     application.Gender,
			"department": application.Department,
			"major":      application.Major,
			"studentId":  application.StudentId,
			"resume":     application.Resume,
			"createdAt":  application.CreatedAt,
			"updatedAt":  application.UpdatedAt,
		}

		if application.Directions != "" {
			var directions []string
			json.Unmarshal([]byte(application.Directions), &directions)
			appData["directions"] = directions
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":   true,
			"data": appData,
		})
	}
}

// SetInterviewStatusRequest 设置面试状态请求
type SetInterviewStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// SetInterviewStatus 设置面试状态（面试官）
func SetInterviewStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("userId")
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "没传 id 属于是"})
			return
		}

		var req SetInterviewStatusRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "检查一下 request body 吧"})
			return
		}

		// 验证状态
		if !auth.ValidateStatus(req.Status) {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "要设置的面试状态不合法"})
			return
		}

		// 解析UUID
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "id 参数校验失败"})
			return
		}

		// 查找用户
		var user models.User
		if err := db.Where("uuid = ?", userUUID).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "用户不存在"})
			return
		}

		// 更新状态
		if err := db.Model(&user).Update("status", req.Status).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// DeleteApplication 删除申请（面试官）
func DeleteApplication(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("userId")
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "没传 id 属于是"})
			return
		}

		// 解析UUID
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "id 参数校验失败"})
			return
		}

		// 查找申请
		var application models.Application
		if err := db.Where("user_id = ?", userUUID).First(&application).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "申请不存在"})
			return
		}

		// 删除申请
		if err := db.Delete(&application).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// DeleteSelfApplication 删除自己的申请
func DeleteSelfApplication(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取当前用户
		userUUID, ok := GetCurrentUserUUID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "未登录"})
			return
		}

		// 查找申请
		var application models.Application
		if err := db.Where("user_id = ?", userUUID).First(&application).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "申请不存在"})
			return
		}

		// 删除申请
		if err := db.Delete(&application).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
