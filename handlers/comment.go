package handlers

import (
	"html/template"
	"net/http"
	"xdsec-join-2026/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateCommentRequest 创建评论请求
type CreateCommentRequest struct {
	IntervieweeID string `json:"intervieweeId" binding:"required"`
	Content      string `json:"content" binding:"required,max=500"`
}

// CreateComment 创建评论（面试官）
func CreateComment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateCommentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 检查是否为面试官
		if GetCurrentUserRole(c) != "interviewer" {
			c.JSON(http.StatusForbidden, gin.H{"ok": false, "message": "只有面试官可以评论"})
			return
		}

		// 获取当前用户
		interviewerUUID, ok := GetCurrentUserUUID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "未登录"})
			return
		}

		// 解析面试者UUID
		intervieweeUUID, err := uuid.Parse(req.IntervieweeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 检查面试者是否存在
		var interviewee models.User
		if err := db.Where("uuid = ?", intervieweeUUID).First(&interviewee).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "面试者不存在"})
			return
		}

		// 检查是否为面试者
		if interviewee.Role != "interviewee" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "目标用户不是面试者"})
			return
		}

		// 创建评论
		commentUUID, _ := uuid.NewUUID()
		comment := models.Comment{
			UUID:          commentUUID,
			Content:       req.Content,
			IntervieweeID:  intervieweeUUID,
			InterviewerID: interviewerUUID,
		}

		if err := db.Create(&comment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// GetComments 获取评论列表
func GetComments(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		intervieweeID := c.Param("intervieweeId")
		if intervieweeID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 解析UUID
		intervieweeUUID, err := uuid.Parse(intervieweeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 查询评论
		var comments []models.Comment
		if err := db.Where("interviewee_id = ?", intervieweeUUID).Order("created_at DESC").Find(&comments).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		// 收集面试官ID
		interviewerIds := make([]uuid.UUID, 0, len(comments))
		for _, comment := range comments {
			interviewerIds = append(interviewerIds, comment.InterviewerID)
		}

		// 批量查询面试官昵称
		interviewerNames := make(map[string]string)
		if len(interviewerIds) > 0 {
			var users []models.User
			if err := db.Select("uuid", "nickname", "email").Where("uuid IN ?", interviewerIds).Find(&users).Error; err == nil {
				for _, user := range users {
					name := user.Email
					if user.Nickname != nil && *user.Nickname != "" {
						name = *user.Nickname
					}
					interviewerNames[user.UUID.String()] = name
				}
			}
		}

		// 构建返回数据
		items := make([]gin.H, 0, len(comments))
		for _, comment := range comments {
			interviewerName := interviewerNames[comment.InterviewerID.String()]
			if interviewerName == "" {
				interviewerName = comment.InterviewerID.String()
			}

			items = append(items, gin.H{
				"id":             comment.UUID.String(),
				"content":        template.HTMLEscapeString(comment.Content),
				"intervieweeId":   comment.IntervieweeID.String(),
				"interviewerId":  comment.InterviewerID.String(),
				"interviewerName": template.HTMLEscapeString(interviewerName),
				"createdAt":      comment.CreatedAt,
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

// UpdateCommentRequest 更新评论请求
type UpdateCommentRequest struct {
	Content string `json:"content" binding:"required,max=500"`
}

// UpdateComment 更新评论
func UpdateComment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		commentID := c.Param("id")
		if commentID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		var req UpdateCommentRequest
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

		// 检查是否为面试官
		if GetCurrentUserRole(c) != "interviewer" {
			c.JSON(http.StatusForbidden, gin.H{"ok": false, "message": "只有面试官可以修改评论"})
			return
		}

		// 解析评论UUID
		commentUUID, err := uuid.Parse(commentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 查找评论
		var comment models.Comment
		if err := db.Where("uuid = ?", commentUUID).First(&comment).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "评论不存在"})
			return
		}

		// 检查是否为评论创建者
		if comment.InterviewerID != userUUID {
			c.JSON(http.StatusForbidden, gin.H{"ok": false, "message": "只能修改自己的评论"})
			return
		}

		// 更新评论
		if err := db.Model(&comment).Update("content", req.Content).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// DeleteComment 删除评论
func DeleteComment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		commentID := c.Param("id")
		if commentID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 获取当前用户
		userUUID, ok := GetCurrentUserUUID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "未登录"})
			return
		}

		// 检查是否为面试官
		if GetCurrentUserRole(c) != "interviewer" {
			c.JSON(http.StatusForbidden, gin.H{"ok": false, "message": "只有面试官可以删除评论"})
			return
		}

		// 解析评论UUID
		commentUUID, err := uuid.Parse(commentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 查找评论
		var comment models.Comment
		if err := db.Where("uuid = ?", commentUUID).First(&comment).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "评论不存在"})
			return
		}

		// 检查是否为评论创建者
		if comment.InterviewerID != userUUID {
			c.JSON(http.StatusForbidden, gin.H{"ok": false, "message": "只能删除自己的评论"})
			return
		}

		// 删除评论
		if err := db.Delete(&comment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
