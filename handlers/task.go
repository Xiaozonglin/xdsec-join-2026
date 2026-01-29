package handlers

import (
	"net/http"
	"xdsec-join-2026/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Title        string `json:"title" binding:"required"`
	Description  string `json:"description" binding:"required"`
	TargetUserId string `json:"targetUserId" binding:"required"`
}

// CreateTask 创建任务（面试官）
func CreateTask(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateTaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 获取当前用户
		userUUID, ok := GetCurrentUserUUID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "未登录"})
			return
		}

		// 解析目标用户UUID
		targetUUID, err := uuid.Parse(req.TargetUserId)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 检查目标用户是否存在
		var targetUser models.User
		if err := db.Where("uuid = ?", targetUUID).First(&targetUser).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "目标用户不存在"})
			return
		}

		// 创建任务
		taskUUID, _ := uuid.NewUUID()
		task := models.Task{
			UUID:         taskUUID,
			Title:        req.Title,
			Description:  req.Description,
			TargetUserId: targetUUID,
			AssignedBy:   userUUID,
			Report:       "",
		}

		if err := db.Create(&task).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// UpdateTaskRequest 更新任务请求
type UpdateTaskRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
}

// UpdateTask 更新任务（面试官）
func UpdateTask(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID := c.Param("id")
		if taskID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		var req UpdateTaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 获取当前用户
		userUUID, ok := GetCurrentUserUUID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "未登录"})
			return
		}

		// 解析UUID
		taskUUID, err := uuid.Parse(taskID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 查找任务
		var task models.Task
		if err := db.Where("uuid = ?", taskUUID).First(&task).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "任务不存在"})
			return
		}

		// 检查是否为任务创建者
		if task.AssignedBy != userUUID {
			c.JSON(http.StatusForbidden, gin.H{"ok": false, "message": "只能修改自己布置的任务"})
			return
		}

		// 更新任务
		updates := map[string]interface{}{
			"title":       req.Title,
			"description": req.Description,
		}

		if err := db.Model(&task).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// SubmitTaskReportRequest 提交任务报告请求
type SubmitTaskReportRequest struct {
	Report string `json:"report" binding:"required"`
}

// SubmitTaskReport 提交任务报告（面试者）
func SubmitTaskReport(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID := c.Param("id")
		if taskID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		var req SubmitTaskReportRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 获取当前用户
		userUUID, ok := GetCurrentUserUUID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "未登录"})
			return
		}

		// 解析UUID
		taskUUID, err := uuid.Parse(taskID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 查找任务
		var task models.Task
		if err := db.Where("uuid = ?", taskUUID).First(&task).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "任务不存在"})
			return
		}

		// 检查是否为目标用户
		if task.TargetUserId != userUUID {
			c.JSON(http.StatusForbidden, gin.H{"ok": false, "message": "无权限"})
			return
		}

		// 更新报告
		if err := db.Model(&task).Update("report", req.Report).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// GetTasks 获取任务列表
func GetTasks(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取查询参数
		scope := c.Query("scope")
		if scope == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		if scope != "mine" && scope != "all" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "scope 参数校验失败"})
			return
		}

		// 获取当前用户
		userUUID, ok := GetCurrentUserUUID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "未登录"})
			return
		}

		var tasks []models.Task
		tx := db.Model(&models.Task{})

		// 根据scope过滤
		if scope == "mine" {
			tx = tx.Where("target_user_id = ?", userUUID)
		}

		// 如果不是面试官且scope为all，只能看到自己的任务
		if scope == "all" && GetCurrentUserRole(c) != "interviewer" {
			c.JSON(http.StatusForbidden, gin.H{"ok": false, "message": "无权限"})
			return
		}

		if err := tx.Order("created_at DESC").Find(&tasks).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		// 收集所有相关的用户ID（布置者和目标用户）
		userIds := make([]uuid.UUID, 0, len(tasks)*2)
		userSet := make(map[uuid.UUID]struct{})
		for _, task := range tasks {
			if _, exists := userSet[task.AssignedBy]; !exists {
				userSet[task.AssignedBy] = struct{}{}
				userIds = append(userIds, task.AssignedBy)
			}
			if _, exists := userSet[task.TargetUserId]; !exists {
				userSet[task.TargetUserId] = struct{}{}
				userIds = append(userIds, task.TargetUserId)
			}
		}

		// 批量查询用户信息
		userNames := make(map[string]string)
		if len(userIds) > 0 {
			var users []models.User
			if err := db.Select("uuid", "nickname", "email").Where("uuid IN ?", userIds).Find(&users).Error; err == nil {
				for _, user := range users {
					name := user.Email
					if user.Nickname != nil && *user.Nickname != "" {
						name = *user.Nickname
					}
					userNames[user.UUID.String()] = name
				}
			}
		}

		items := make([]gin.H, 0, len(tasks))
		for _, t := range tasks {
			assignedBy := userNames[t.AssignedBy.String()]
			if assignedBy == "" {
				assignedBy = t.AssignedBy.String()
			}

			targetUserName := userNames[t.TargetUserId.String()]
			if targetUserName == "" {
				targetUserName = t.TargetUserId.String()
			}

			items = append(items, gin.H{
				"id":             t.UUID.String(),
				"title":          t.Title,
				"description":    t.Description,
				"targetUserId":   t.TargetUserId.String(),
				"targetUserName": targetUserName,
				"assignedBy":     assignedBy,
				"report":         t.Report,
				"createdAt":      t.CreatedAt,
				"updatedAt":      t.UpdatedAt,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"ok": true,
			"data": gin.H{
				"items": items,
			},
		})
	}
}

// DeleteTask 删除任务（面试官）
func DeleteTask(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID := c.Param("id")
		if taskID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 获取当前用户
		userUUID, ok := GetCurrentUserUUID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "未登录"})
			return
		}

		// 解析UUID
		taskUUID, err := uuid.Parse(taskID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 查找任务
		var task models.Task
		if err := db.Where("uuid = ?", taskUUID).First(&task).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "任务不存在"})
			return
		}

		// 检查是否为任务创建者
		if task.AssignedBy != userUUID {
			c.JSON(http.StatusForbidden, gin.H{"ok": false, "message": "只能删除自己布置的任务"})
			return
		}

		// 删除任务
		if err := db.Delete(&task).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
