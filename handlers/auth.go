package handlers

import (
	"net/http"

	"xdsec-join-2026/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthMiddleware Session认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从Cookie获取session_id
		sessionID, err := c.Cookie("session_id")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "未登录"})
			c.Abort()
			return
		}

		if sessionID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "未登录"})
			c.Abort()
			return
		}

		// 验证Token（session_id实际上是JWT token）
		claims, err := auth.ParseToken(sessionID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "会话无效"})
			c.Abort()
			return
		}

		// 验证CSRF Token（对于非GET请求）
		if c.Request.Method != "GET" && c.Request.Method != "HEAD" && c.Request.Method != "OPTIONS" {
			csrfToken := c.GetHeader("X-CSRF-Token")
			if csrfToken == "" {
				c.JSON(http.StatusForbidden, gin.H{"ok": false, "message": "缺少CSRF Token"})
				c.Abort()
				return
			}

			// 从Cookie获取CSRF Token进行验证
			cookieCSRF, err := c.Cookie("csrf_token")
			if err != nil || cookieCSRF != csrfToken {
				c.JSON(http.StatusForbidden, gin.H{"ok": false, "message": "CSRF Token无效"})
				c.Abort()
				return
			}
		}

		// 将用户信息存入Context
		c.Set("user_uuid", claims.UserUUID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

// RequireInterviewer 要求面试官权限的中间件
func RequireInterviewer() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists || userRole != "interviewer" {
			c.JSON(http.StatusForbidden, gin.H{"ok": false, "message": "无权限"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// GetCurrentUserUUID 获取当前用户UUID
func GetCurrentUserUUID(c *gin.Context) (uuid.UUID, bool) {
	userUUID, exists := c.Get("user_uuid")
	if !exists {
		return uuid.Nil, false
	}
	parsed, err := uuid.Parse(userUUID.(string))
	if err != nil {
		return uuid.Nil, false
	}
	return parsed, true
}

// GetCurrentUserRole 获取当前用户角色
func GetCurrentUserRole(c *gin.Context) string {
	userRole, exists := c.Get("user_role")
	if !exists {
		return ""
	}
	return userRole.(string)
}
