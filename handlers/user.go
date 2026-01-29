package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"xdsec-join-2026/auth"
	"xdsec-join-2026/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CommentData 评论数据结构
type CommentData struct {
	ID            string `json:"id"`
	Content       string `json:"content"`
	InterviewerId string `json:"interviewerId"`
	InterviewerName string `json:"interviewerName"`
	CreatedAt     string `json:"createdAt"`
}

// GetUsers 获取用户列表
func GetUsers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		currentRole := GetCurrentUserRole(c)

		// 获取查询参数
		role := c.Query("role")
		query := c.Query("q")

		// 构建查询
		tx := db.Model(&models.User{})
		if currentRole == "interviewer" {
			tx = tx.Preload("Application")
		}

		// 按角色过滤
		if role != "" {
			if !auth.ValidateRole(role) {
				c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
				return
			}
			tx = tx.Where("role = ?", role)
		}

		// 按关键词搜索（昵称或邮箱）
		if query != "" {
			tx = tx.Where("nickname LIKE ? OR email LIKE ?", "%"+query+"%", "%"+query+"%")
		}

		var users []models.User
		if err := tx.Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		// 面试官视角：预加载评论
		userCommentsMap := make(map[uuid.UUID][]CommentData)
		if currentRole == "interviewer" {
			// 收集所有面试者ID
			intervieweeIds := make([]uuid.UUID, 0)
			for _, user := range users {
				if user.Role == "interviewee" {
					intervieweeIds = append(intervieweeIds, user.UUID)
				}
			}

			// 批量查询评论
			if len(intervieweeIds) > 0 {
				var comments []models.Comment
				if err := db.Where("interviewee_id IN ?", intervieweeIds).Find(&comments).Error; err == nil {
					// 收集面试官ID
					interviewerIds := make([]uuid.UUID, 0)
					interviewerSet := make(map[uuid.UUID]struct{})
					for _, comment := range comments {
						if _, exists := interviewerSet[comment.InterviewerID]; !exists {
							interviewerSet[comment.InterviewerID] = struct{}{}
							interviewerIds = append(interviewerIds, comment.InterviewerID)
						}
					}

					// 批量查询面试官昵称
					interviewerNames := make(map[string]string)
					if len(interviewerIds) > 0 {
						var interviewers []models.User
						if err := db.Select("uuid", "nickname", "email").Where("uuid IN ?", interviewerIds).Find(&interviewers).Error; err == nil {
							for _, interviewer := range interviewers {
								name := interviewer.Email
								if interviewer.Nickname != nil && *interviewer.Nickname != "" {
									name = *interviewer.Nickname
								}
								interviewerNames[interviewer.UUID.String()] = name
							}
						}
					}

					// 按面试者分组评论
					for _, comment := range comments {
						interviewerName := interviewerNames[comment.InterviewerID.String()]
						if interviewerName == "" {
							interviewerName = comment.InterviewerID.String()
						}
						commentData := CommentData{
							ID:             comment.UUID.String(),
							Content:        comment.Content,
							InterviewerId:  comment.InterviewerID.String(),
							InterviewerName: interviewerName,
							CreatedAt:     comment.CreatedAt.Format("2006-01-02 15:04:05"),
						}
						userCommentsMap[comment.IntervieweeID] = append(userCommentsMap[comment.IntervieweeID], commentData)
					}
				}
			}
		}

		// 面试官视角：预加载任务
		userTasksMap := make(map[uuid.UUID]interface{})
		if currentRole == "interviewer" {
			// 收集所有面试者ID
			intervieweeIds := make([]uuid.UUID, 0)
			for _, user := range users {
				if user.Role == "interviewee" {
					intervieweeIds = append(intervieweeIds, user.UUID)
				}
			}

			// 批量查询任务
			if len(intervieweeIds) > 0 {
				var tasks []models.Task
				if err := db.Where("target_user_id IN ?", intervieweeIds).Find(&tasks).Error; err == nil {
					// 按面试者分组任务
					for _, task := range tasks {
						userTasksMap[task.TargetUserId] = gin.H{
							"id":          task.UUID.String(),
							"title":       task.Title,
							"description": task.Description,
							"report":      task.Report,
							"createdAt":   task.CreatedAt.Format("2006-01-02 15:04:05"),
							"updatedAt":   task.UpdatedAt.Format("2006-01-02 15:04:05"),
						}
					}
				}
			}
		}

		// 构建响应
		items := make([]gin.H, 0, len(users))
		for _, user := range users {
			nickname := ""
			if user.Nickname != nil {
				nickname = *user.Nickname
			}

			userData := gin.H{
				"id":        user.UUID.String(),
				"nickname":  template.HTMLEscapeString(nickname),
				"email":     user.Email,
				"signature": template.HTMLEscapeString(user.Signature),
				"role":      user.Role,
				"status":    user.Status,
			}

			// 解析Directions
			if user.Directions != "" {
				var directions []string
				json.Unmarshal([]byte(user.Directions), &directions)
				userData["directions"] = directions
			}

			// 解析PassedDirections
			if user.PassedDirections != "" {
				var passedDirections []string
				json.Unmarshal([]byte(user.PassedDirections), &passedDirections)
				userData["passedDirections"] = passedDirections
			}

			// 解析PassedDirectionsBy（数组）
			if user.PassedDirectionsBy != "" {
				var passedByList []string
				json.Unmarshal([]byte(user.PassedDirectionsBy), &passedByList)
				userData["passedDirectionsBy"] = passedByList
			}

			// 面试官视角可以看到更多信息
			if currentRole == "interviewer" {
				userData["email"] = user.Email
				if user.Application != nil {
					userData["application"] = user.Application
				}

				// 添加任务
				if task, exists := userTasksMap[user.UUID]; exists {
					userData["task"] = task
				}

				// 添加评论
				if comments, exists := userCommentsMap[user.UUID]; exists {
					userData["comments"] = comments
				} else {
					userData["comments"] = []CommentData{}
				}
			}

			items = append(items, userData)
		}

		c.JSON(http.StatusOK, gin.H{
			"ok": true,
			"data": gin.H{
				"items": items,
			},
		})
	}
}

// GetUserDetail 获取用户详情（面试官）
func GetUserDetail(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("id")
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 解析UUID
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 查询用户
		var user models.User
		if err := db.Preload("Application").Where("uuid = ?", userUUID).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "用户不存在"})
			return
		}

		userData := gin.H{
			"id":        user.UUID.String(),
			"email":     user.Email,
			"nickname":  user.Nickname,
			"signature": user.Signature,
			"role":      user.Role,
			"status":    user.Status,
		}

		// 解析Directions
		if user.Directions != "" {
			var directions []string
			json.Unmarshal([]byte(user.Directions), &directions)
			userData["directions"] = directions
		}

		// 解析PassedDirections
		if user.PassedDirections != "" {
			var passedDirections []string
			json.Unmarshal([]byte(user.PassedDirections), &passedDirections)
			userData["passedDirections"] = passedDirections
		}

		// 解析PassedDirectionsBy（数组）
		if user.PassedDirectionsBy != "" {
			var passedByList []string
			json.Unmarshal([]byte(user.PassedDirectionsBy), &passedByList)
			userData["passedDirectionsBy"] = passedByList
		}

		// 包含申请信息
		if user.Application != nil {
			app := user.Application
			appData := gin.H{
				"realName":   app.RealName,
				"phone":      app.Phone,
				"gender":     app.Gender,
				"department": app.Department,
				"major":      app.Major,
				"studentId":  app.StudentId,
				"resume":     app.Resume,
			}

			if app.Directions != "" {
				var directions []string
				json.Unmarshal([]byte(app.Directions), &directions)
				appData["directions"] = directions
			}

			userData["application"] = appData
		}

		c.JSON(http.StatusOK, gin.H{
			"ok": true,
			"data": gin.H{
				"user": userData,
			},
		})
	}
}

// UpdateProfileRequest 更新个人资料请求
type UpdateProfileRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Nickname  string `json:"nickname" binding:"required"`
	Signature string `json:"signature" binding:"required"`
	EmailCode string `json:"emailCode" binding:"required"`
}

// UpdateProfile 更新个人资料
func UpdateProfile(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req UpdateProfileRequest
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

		// 验证昵称
		if !auth.ValidateNickname(req.Nickname) {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "昵称格式不正确"})
			return
		}

		// 验证邮箱验证码
		if !ValidateEmailCode(db, req.Email, req.EmailCode, "profile") {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "验证码无效或已过期"})
			return
		}

		// 查找用户
		var user models.User
		if err := db.Where("uuid = ?", userUUID).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "用户不存在"})
			return
		}

		// 检查邮箱是否被其他用户使用
		if user.Email != req.Email {
			var existingUser models.User
			if err := db.Where("email = ? AND uuid != ?", req.Email, userUUID).First(&existingUser).Error; err == nil {
				c.JSON(http.StatusConflict, gin.H{"ok": false, "message": "邮箱已被使用"})
				return
			}
		}

		// 检查昵称是否被其他用户使用
		if user.Nickname != nil && *user.Nickname != req.Nickname {
			var existingUser models.User
			if err := db.Where("nickname = ? AND uuid != ?", req.Nickname, userUUID).First(&existingUser).Error; err == nil {
				c.JSON(http.StatusConflict, gin.H{"ok": false, "message": "昵称已被使用"})
				return
			}
		}

		// 更新用户信息
		updates := map[string]interface{}{
			"email":     req.Email,
			"nickname":  &req.Nickname,
			"signature": req.Signature,
		}

		if err := db.Model(&user).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// SetRoleRequest 设置角色请求
type SetRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// SetUserRole 设置用户角色（面试官）
func SetUserRole(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("id")
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		var req SetRoleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 验证角色
		if !auth.ValidateRole(req.Role) {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 解析UUID
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 查找用户
		var user models.User
		if err := db.Where("uuid = ?", userUUID).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "用户不存在"})
			return
		}

		// 更新角色
		if err := db.Model(&user).Update("role", req.Role).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// SetPassedDirectionsRequest 设置通过方向请求
type SetPassedDirectionsRequest struct {
	Directions []string `json:"directions" binding:"required"`
}

// SetPassedDirections 设置通过方向（面试官）
func SetPassedDirections(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("id")
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		var req SetPassedDirectionsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 验证方向
		if !auth.ValidateDirections(req.Directions) {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 解析UUID
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 查找用户
		var user models.User
		if err := db.Where("uuid = ?", userUUID).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "用户不存在"})
			return
		}

		// 获取当前用户（面试官）
		currentUUID, ok := GetCurrentUserUUID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "未登录"})
			return
		}

		var currentUser models.User
		if err := db.Where("uuid = ?", currentUUID).First(&currentUser).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "用户不存在"})
			return
		}

		// 序列化方向
		directionsJSON, _ := json.Marshal(req.Directions)

		// 解析现有的 passedDirectionsBy 数组
		var passedByList []string
		if user.PassedDirectionsBy != "" {
			json.Unmarshal([]byte(user.PassedDirectionsBy), &passedByList)
		}

		// 获取当前面试官昵称
		currentNickname := ""
		if currentUser.Nickname != nil {
			currentNickname = *currentUser.Nickname
		}

		// 检查当前面试官是否已在列表中
		found := false
		for _, nickname := range passedByList {
			if nickname == currentNickname {
				found = true
				break
			}
		}

		// 如果不在列表中，添加当前面试官昵称
		if !found {
			passedByList = append(passedByList, currentNickname)
		}

		// 序列化数组
		passedByJSON, _ := json.Marshal(passedByList)

		// 更新
		updates := map[string]interface{}{
			"passed_directions":    string(directionsJSON),
			"passed_directions_by": string(passedByJSON),
		}

		if err := db.Model(&user).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// DeleteUser 删除用户（面试官）
func DeleteUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("id")
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 解析UUID
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 查找用户
		var user models.User
		if err := db.Where("uuid = ?", userUUID).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "用户不存在"})
			return
		}

		// 删除用户（会级联删除关联的申请）
		if err := db.Delete(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// DeleteSelf 删除自己的账户
func DeleteSelf(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取当前用户
		userUUID, ok := GetCurrentUserUUID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "未登录"})
			return
		}

		// 查找用户
		var user models.User
		if err := db.Where("uuid = ?", userUUID).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "用户不存在"})
			return
		}

		// 删除用户（会级联删除关联的申请）
		if err := db.Delete(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
