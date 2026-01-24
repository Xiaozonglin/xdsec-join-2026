package utils

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetCurrentUserUUID 获取当前登录用户的UUID
func GetCurrentUserUUID(c *gin.Context) (uuid.UUID, error) {
	userUUIDStr, exists := c.Get("user_uuid")
	if !exists {
		return uuid.Nil, errors.New("user not authenticated")
	}

	userUUID, err := uuid.Parse(userUUIDStr.(string))
	if err != nil {
		return uuid.Nil, err
	}

	return userUUID, nil
}

// GetCurrentUserRole 获取当前用户的角色
func GetCurrentUserRole(c *gin.Context) string {
	role, exists := c.Get("user_role")
	if !exists {
		return ""
	}
	return role.(string)
}

// IsInterviewer 检查当前用户是否是面试官
func IsInterviewer(c *gin.Context) bool {
	return GetCurrentUserRole(c) == "interviewer"
}

// IsInterviewee 检查当前用户是否是面试者
func IsInterviewee(c *gin.Context) bool {
	return GetCurrentUserRole(c) == "interviewee"
}
