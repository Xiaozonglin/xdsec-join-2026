package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"xdsec-join-2026/auth"
	"xdsec-join-2026/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateAnnouncementRequest 创建公告请求
type CreateAnnouncementRequest struct {
	Title           string   `json:"title" binding:"required,max=20"`
	Content         string   `json:"content" binding:"required,max=10000"`
	Visibility      string   `json:"visibility" binding:"required"`
	AllowedStatuses []string `json:"allowedStatuses"`
}

// CreateAnnouncement 创建公告（面试官）
func CreateAnnouncement(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateAnnouncementRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}
		if !validateAnnouncementVisibility(req.Visibility, req.AllowedStatuses) {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "可见范围参数校验失败"})
			return
		}

		// 获取当前用户
		userUUID, ok := GetCurrentUserUUID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "未登录"})
			return
		}

		allowedStatuses := req.AllowedStatuses
		if req.Visibility != "status" {
			allowedStatuses = []string{}
		}
		allowedStatusesJSON, _ := json.Marshal(allowedStatuses)
		// 创建公告
		announcementUUID, _ := uuid.NewUUID()
		announcement := models.Announcement{
			UUID:            announcementUUID,
			Title:           req.Title,
			Content:         req.Content,
			Pinned:          false,
			AuthorId:        userUUID,
			Visibility:      req.Visibility,
			AllowedStatuses: string(allowedStatusesJSON),
		}

		if err := db.Create(&announcement).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// UpdateAnnouncementRequest 更新公告请求
type UpdateAnnouncementRequest struct {
	Title           string   `json:"title" binding:"required,max=20"`
	Content         string   `json:"content" binding:"required,max=10000"`
	Visibility      string   `json:"visibility" binding:"required"`
	AllowedStatuses []string `json:"allowedStatuses"`
}

// UpdateAnnouncement 更新公告（面试官）
func UpdateAnnouncement(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		announcementID := c.Param("id")
		if announcementID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		var req UpdateAnnouncementRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}
		if !validateAnnouncementVisibility(req.Visibility, req.AllowedStatuses) {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "可见范围参数校验失败"})
			return
		}

		// 解析UUID
		announcementUUID, err := uuid.Parse(announcementID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 查找公告
		var announcement models.Announcement
		if err := db.Where("uuid = ?", announcementUUID).First(&announcement).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "公告不存在"})
			return
		}

		allowedStatuses := req.AllowedStatuses
		if req.Visibility != "status" {
			allowedStatuses = []string{}
		}
		allowedStatusesJSON, _ := json.Marshal(allowedStatuses)
		// 更新公告
		updates := map[string]interface{}{
			"title":            req.Title,
			"content":          req.Content,
			"visibility":       req.Visibility,
			"allowed_statuses": string(allowedStatusesJSON),
		}

		if err := db.Model(&announcement).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// PinAnnouncementRequest 置顶公告请求
type PinAnnouncementRequest struct {
	Pinned bool `json:"pinned"`
}

// PinAnnouncement 置顶或取消置顶公告（面试官）
func PinAnnouncement(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		announcementID := c.Param("id")
		if announcementID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		var req PinAnnouncementRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败" + err.Error()})
			return
		}

		// 查找公告
		var announcement models.Announcement
		if err := db.Where("uuid = ?", announcementID).First(&announcement).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "公告不存在"})
			return
		}

		// 更新置顶状态
		if err := db.Model(&announcement).Update("pinned", req.Pinned).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// GetAnnouncements 获取公告列表
func GetAnnouncements(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var announcements []models.Announcement
		query := db.Model(&models.Announcement{})

		// 尝试解析登录信息（可选）
		sessionID, _ := c.Cookie("session_id")
		role := ""
		status := ""
		if sessionID != "" {
			if claims, err := auth.ParseToken(sessionID); err == nil {
				role = claims.Role
				if role == "interviewee" {
					var user models.User
					if err := db.Select("status").Where("uuid = ?", claims.UserUUID).First(&user).Error; err == nil {
						status = user.Status
					}
				}
			}
		}

		if role == "interviewer" {
			// 面试官可见全部
		} else if role == "interviewee" {
			statusJSON := fmt.Sprintf("\"%s\"", status)
			query = query.Where("(visibility IN ? OR visibility = '' OR visibility IS NULL) OR (visibility = 'status' AND JSON_CONTAINS(allowed_statuses, ?))",
				[]string{"public", "all"}, statusJSON)
		} else {
			query = query.Where("visibility = ? OR visibility = '' OR visibility IS NULL", "public")
		}

		// 按置顶和创建时间排序，并预加载作者信息
		if err := query.Order("pinned DESC, created_at DESC").Find(&announcements).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		authorIds := make([]uuid.UUID, 0, len(announcements))
		authorIdSet := make(map[uuid.UUID]struct{})
		for _, announcement := range announcements {
			if _, exists := authorIdSet[announcement.AuthorId]; !exists {
				authorIdSet[announcement.AuthorId] = struct{}{}
				authorIds = append(authorIds, announcement.AuthorId)
			}
		}

		authorNames := make(map[string]string)
		if len(authorIds) > 0 {
			var authors []models.User
			if err := db.Select("uuid", "nickname", "email").Where("uuid IN ?", authorIds).Find(&authors).Error; err == nil {
				for _, author := range authors {
					name := author.Email
					if author.Nickname != nil && *author.Nickname != "" {
						name = *author.Nickname
					}
					authorNames[author.UUID.String()] = name
				}
			}
		}

		items := make([]gin.H, 0, len(announcements))
		for _, a := range announcements {
			authorNickname := authorNames[a.AuthorId.String()]
			if authorNickname == "" {
				authorNickname = "未知"
			}

			items = append(items, gin.H{
				"id":              a.UUID.String(),
				"title":           template.HTMLEscapeString(a.Title),
				"content":         template.HTMLEscapeString(a.Content),
				"pinned":          a.Pinned,
				"authorNickname":  template.HTMLEscapeString(authorNickname),
				"visibility":      a.Visibility,
				"allowedStatuses": parseJSONList(a.AllowedStatuses),
				"createdAt":       a.CreatedAt,
				"updatedAt":       a.UpdatedAt,
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

func parseJSONList(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return []string{}
	}
	return values
}

func validateAnnouncementVisibility(visibility string, allowedStatuses []string) bool {
	valid := map[string]bool{
		"public":      true,
		"all":         true,
		"interviewer": true,
		"status":      true,
	}
	if !valid[visibility] {
		return false
	}
	if visibility == "status" {
		if len(allowedStatuses) == 0 {
			return false
		}
		for _, item := range allowedStatuses {
			if !auth.ValidateStatus(item) {
				return false
			}
		}
	}
	return true
}

// DeleteAnnouncement 删除公告（面试官）
func DeleteAnnouncement(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		announcementID := c.Param("id")
		if announcementID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 解析UUID
		announcementUUID, err := uuid.Parse(announcementID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 查找公告
		var announcement models.Announcement
		if err := db.Where("uuid = ?", announcementUUID).First(&announcement).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "公告不存在"})
			return
		}

		// 删除公告
		if err := db.Delete(&announcement).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
