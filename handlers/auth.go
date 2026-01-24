package handlers

import (
	"net/http"
	"strings"

	"xdsec-join-2026/auth"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从Header获取Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"ok":      false,
				"message": "未登录",
			})
			c.Abort()
			return
		}

		// 检查Bearer格式
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"ok":      false,
				"message": "未登录",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 解析并验证Token
		claims, err := auth.ParseToken(tokenString)
		if err != nil {
			if err == auth.ErrExpiredToken {
				c.JSON(http.StatusUnauthorized, gin.H{
					"ok":      false,
					"message": "token已过期",
				})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{
					"ok":      false,
					"message": "token无效",
				})
			}
			c.Abort()
			return
		}

		// 将用户信息存入Context
		c.Set("user_uuid", claims.UserUUID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}
