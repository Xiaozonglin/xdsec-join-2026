package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"xdsec-join-2026/auth"
	"xdsec-join-2026/models"
	"xdsec-join-2026/smtp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EmailCodeRequest 邮箱验证码请求
type EmailCodeRequest struct {
	Email   string `json:"email" binding:"required,email"`
	Purpose string `json:"purpose" binding:"required"`
}

// SendEmailCode 发送邮箱验证码
func SendEmailCode(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req EmailCodeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 验证purpose
		if !auth.ValidateEmailCodePurpose(req.Purpose) {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "purpose 参数校验失败"})
			return
		}

		// 根据purpose验证邮箱状态
		var user models.User
		switch req.Purpose {
		case "register":
			// 注册验证：检查邮箱是否已注册
			if err := db.Where("email = ?", req.Email).First(&user).Error; err == nil {
				c.JSON(http.StatusConflict, gin.H{"ok": false, "message": "该邮箱已被注册，请直接登录"})
				return
			}
		case "reset":
			// 重置密码验证：检查邮箱是否存在用户
			if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "该邮箱未注册账号"})
				return
			}
		}

		// 检查发送频率限制（每分钟一次）
		var rateLimit models.EmailRateLimit
		oneMinuteAgo := time.Now().Add(-1 * time.Minute)
		if err := db.Where("email = ? AND last_sent > ?", req.Email, oneMinuteAgo).First(&rateLimit).Error; err == nil {
			c.JSON(http.StatusTooManyRequests, gin.H{"ok": false, "message": "发送过于频繁，请1分钟后再试"})
			return
		}

		// 生成验证码
		code, err := auth.GenerateEmailCode()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "生成验证码失败"})
			return
		}

		// 清理该邮箱的旧验证码（包括已使用和过期的）
		db.Where("email = ? AND purpose = ?", req.Email, req.Purpose).Delete(&models.EmailCode{})

		// 创建新的验证码记录
		emailCodeUUID, _ := uuid.NewUUID()
		emailCode := models.EmailCode{
			UUID:      emailCodeUUID,
			Email:     req.Email,
			Code:      code,
			Purpose:   req.Purpose,
			ExpiresAt: time.Now().Add(5 * time.Minute),
			Used:      false,
		}

		if err := db.Create(&emailCode).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		// 更新或创建频率限制记录
		rateLimitUUID, _ := uuid.NewUUID()
		if err := db.Where("email = ?", req.Email).Assign(models.EmailRateLimit{
			Email:    req.Email,
			LastSent: time.Now(),
		}).FirstOrCreate(&models.EmailRateLimit{
			UUID:     rateLimitUUID,
			Email:    req.Email,
			LastSent: time.Now(),
		}).Error; err != nil {
			// 记录失败不影响发送，仅记录日志
			c.JSON(http.StatusOK, gin.H{"ok": true, "message": "sent"})
			return
		}

		// 发送邮件
		go func() {
			smtp.SendEmailCode(req.Email, code, req.Purpose)
		}()

		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "sent"})
	}
}

// ValidateEmailCode 验证邮箱验证码（不标记为已使用）
func ValidateEmailCode(db *gorm.DB, email, code, purpose string) bool {
	var emailCode models.EmailCode
	result := db.Where("email = ? AND code = ? AND purpose = ? AND used = ? AND expires_at > ?",
		email, code, purpose, false, time.Now()).First(&emailCode)

	if result.Error != nil {
		return false
	}

	return true
}

// MarkEmailCodeUsed 标记邮箱验证码为已使用
func MarkEmailCodeUsed(db *gorm.DB, email, code, purpose string) bool {
	var emailCode models.EmailCode
	result := db.Where("email = ? AND code = ? AND purpose = ? AND used = ? AND expires_at > ?",
		email, code, purpose, false, time.Now()).First(&emailCode)

	if result.Error != nil {
		return false
	}

	// 标记为已使用
	db.Model(&emailCode).Update("used", true)
	return true
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Password  string `json:"password" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	Nickname  string `json:"nickname" binding:"required"`
	Signature string `json:"signature" binding:"required"`
	EmailCode string `json:"emailCode" binding:"required"`
}

// Register 用户注册
func Register(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

			// 验证邮箱验证码（不标记为已使用）
		if !ValidateEmailCode(db, req.Email, req.EmailCode, "register") {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "验证码无效或已过期"})
			return
		}

		// 验证昵称
		if !auth.ValidateNickname(req.Nickname) {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "昵称格式不正确"})
			return
		}

		// 验证密码
		if len(req.Password) < 8 {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "密码长度不能少于8位"})
			return
		}

		// 检查邮箱是否已存在
		var existingUser models.User
		if err := db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"ok": false, "message": "邮箱已被注册"})
			return
		}

		// 检查昵称是否已存在
		if err := db.Where("nickname = ?", req.Nickname).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"ok": false, "message": "昵称已被使用"})
			return
		}

		// 所有参数校验通过后，标记验证码为已使用
		if !MarkEmailCodeUsed(db, req.Email, req.EmailCode, "register") {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "验证码验证失败"})
			return
		}

		// 生成UUID
		userUUID, err := uuid.NewUUID()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "生成用户ID失败"})
			return
		}

		// 加密密码
		hashedPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		// 创建用户
		user := models.User{
			UUID:               userUUID,
			Email:              req.Email,
			Nickname:           &req.Nickname,
			Signature:          req.Signature,
			Role:               "interviewee",
			Status:             "r1_pending",
			Directions:         "[]",
			PassedDirections:   "[]",
			PassedDirectionsBy: "[]",
			PassWord:           hashedPassword,
		}

		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ok": true,
			"data": gin.H{
				"userId": userUUID.String(),
			},
		})
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	ID       string `json:"id" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 用户登录
func Login(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 查找用户（邮箱或昵称）
		var user models.User
		if err := db.Where("email = ? OR nickname = ?", req.ID, req.ID).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "邮箱或密码错误"})
			return
		}

		// 验证密码
		if err := auth.CheckPassword(req.Password, user.PassWord); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "邮箱或密码错误"})
			return
		}

		// 生成Token
		token, err := auth.GenerateToken(user.UUID.String(), user.Email, user.Role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		// 生成CSRF Token
		csrfToken := auth.GenerateCSRFToken()

		// 设置Cookie
		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie("session_id", token, 7*24*3600, "/", "", false, true)
		c.SetCookie("csrf_token", csrfToken, 7*24*3600, "/", "", false, false)

		// 解析Directions JSON
		var directions []string
		if user.Directions != "" {
			json.Unmarshal([]byte(user.Directions), &directions)
		}

		c.JSON(http.StatusOK, gin.H{
			"ok": true,
			"data": gin.H{
				"user": gin.H{
					"id":         user.UUID.String(),
					"role":       user.Role,
					"nickname":   user.Nickname,
					"email":      user.Email,
					"signature":  user.Signature,
					"directions": directions,
					"status":     user.Status,
				},
				"csrfToken": csrfToken,
			},
		})
	}
}

// Logout 用户登出
func Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.SetCookie("session_id", "", -1, "/", "", false, true)
		c.SetCookie("csrf_token", "", -1, "/", "", false, false)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

// ChangePassword 修改密码
func ChangePassword(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ChangePasswordRequest
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

		var user models.User
		if err := db.Where("uuid = ?", userUUID).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "用户不存在"})
			return
		}

		// 验证旧密码
		if err := auth.CheckPassword(req.OldPassword, user.PassWord); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "旧密码错误"})
			return
		}

		// 验证新密码
		if len(req.NewPassword) < 8 {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "新密码长度不能少于8位"})
			return
		}

		// 加密新密码
		hashedPassword, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		// 更新密码
		if err := db.Model(&user).Update("password", hashedPassword).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	EmailCode   string `json:"emailCode" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

// ResetPassword 忘记密码重置
func ResetPassword(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ResetPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "参数校验失败"})
			return
		}

		// 验证邮箱验证码（不标记为已使用）
		if !ValidateEmailCode(db, req.Email, req.EmailCode, "reset") {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "验证码无效或已过期"})
			return
		}

		// 查找用户
		var user models.User
		if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "用户不存在"})
			return
		}

		// 验证新密码
		if len(req.NewPassword) < 8 {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "新密码长度不能少于8位"})
			return
		}

		// 所有参数校验通过后，标记验证码为已使用
		if !MarkEmailCodeUsed(db, req.Email, req.EmailCode, "reset") {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "验证码验证失败"})
			return
		}

		// 加密新密码
		hashedPassword, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		// 更新密码
		if err := db.Model(&user).Update("password", hashedPassword).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// GetCurrentUser 获取当前用户信息
func GetCurrentUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userUUID, ok := GetCurrentUserUUID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "未登录"})
			return
		}

		var user models.User
		if err := db.Where("uuid = ?", userUUID).First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "用户不存在"})
			return
		}

		// 解析Directions JSON
		var directions []string
		if user.Directions != "" {
			json.Unmarshal([]byte(user.Directions), &directions)
		}

		c.JSON(http.StatusOK, gin.H{
			"ok": true,
			"data": gin.H{
				"user": gin.H{
					"id":         user.UUID.String(),
					"role":       user.Role,
					"nickname":   user.Nickname,
					"email":      user.Email,
					"signature":  user.Signature,
					"directions": directions,
					"status":     user.Status,
				},
			},
		})
	}
}
