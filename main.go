package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"
	"xdsec-join-2026/auth"
	"xdsec-join-2026/handlers"
	"xdsec-join-2026/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func isInterviewer(c *gin.Context) bool {
	result := false
	return result
}

func genToken()

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	dsn := os.Getenv("dsn")
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("数据库连接失败: " + err.Error())
	}

	db.AutoMigrate(&models.Application{}, &models.User{}, &models.Announcement{}, &models.Task{})

	r := gin.Default()
	r.RedirectTrailingSlash = false

	// 定义路由组

	base := r.Group("/api/v2")

	authRoute := base.Group("/auth")

	// 用户注册
	authRoute.POST("/register", func(c *gin.Context) {
		type register struct {
			Password  string `json:"password" binding:"required"`
			Email     string `json:"email" binding:"required,email"`
			Nickname  string `json:"nickname" binding:"required"`
			Signature string `json:"signature" binding:"required"`
		}

		var userData register

		// 使用 BindJSON 自动绑定并验证
		if err := c.ShouldBindJSON(&userData); err != nil {
			c.JSON(400, gin.H{
				"ok":      false,
				"message": "请求数据无效",
			})
			return
		}

		var existingUser models.User
		result := db.Where("email = ? OR nickname = ?", userData.Email, userData.Nickname).First(&existingUser)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				useruuid, err := uuid.NewUUID()
				if err != nil {
					c.JSON(500, gin.H{
						"ok":      false,
						"message": "生成UUID时出现问题",
					})
					return
				}
				nickname := userData.Nickname
				hashedPassword, err := auth.HashPassword(userData.Password)
				if err != nil {
					c.JSON(500, gin.H{
						"ok":      false,
						"message": "密码加密失败",
					})
					return
				}
				user := models.User{
					UUID:      useruuid,
					Email:     userData.Email,
					Nickname:  &nickname,
					Signature: userData.Signature,
					Role:      "interviewee", // 默认权限为interviewee
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					PassWord:  hashedPassword,
				}

				result := db.Create(&user)
				if result.Error == nil {
					token, err := auth.GenerateToken(user.UUID.String(), user.Email, user.Role)
					if err != nil {
						c.JSON(500, gin.H{
							"ok":      false,
							"message": "服务器生成token时发生错误",
						})
						return
					}
					c.JSON(200, gin.H{
						"ok": true,
						"data": gin.H{
							"userId": useruuid.String(),
							"token":  token,
						},
					})
				} else {
					c.JSON(500, gin.H{
						"ok":      false,
						"message": "数据库操作时出现错误",
					})
				}
			} else {
				c.JSON(500, gin.H{
					"ok":      false,
					"message": "数据库操作时出现错误",
				})
			}
		} else {
			if existingUser.Email == userData.Email {
				c.JSON(409, gin.H{"ok": false, "message": "邮箱已被注册"})
			} else if existingUser.Nickname != nil && *existingUser.Nickname == userData.Nickname {
				c.JSON(409, gin.H{"ok": false, "message": "昵称已被使用"})
			} else {
				c.JSON(409, gin.H{"ok": false, "message": "用户信息重复"})
			}
		}
	})

	// 用户登录
	authRoute.POST("/login", func(c *gin.Context) {
		type LoginRequest struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required"`
		}

		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"ok":      false,
				"message": "请求数据无效",
			})
			return
		}

		var user models.User
		result := db.Where("Email = ?", req.Email).First(&user)
		if result.Error != nil {
			c.JSON(404, gin.H{
				"ok":      false,
				"message": "未找到用户",
			})
			return
		}

		if err := auth.CheckPassword(req.Password, user.PassWord); err != nil {
			c.JSON(400, gin.H{
				"ok":      false,
				"message": "邮箱或密码错误",
			})
			return
		}

		token, err := auth.GenerateToken(user.UUID.String(), user.Email, user.Role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "生成token时发生错误"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ok": true,
			"data": gin.H{
				"token": token,
				"userInfo": gin.H{
					"uuid":     user.UUID,
					"email":    user.Email,
					"nickname": user.Nickname,
					"role":     user.Role,
				},
				"expires_in": 7 * 24 * 60 * 60, // 7天，单位秒
			}})
	})

	authRoute.POST("/change-password", handlers.AuthMiddleware(), func(c *gin.Context) {
		type ChangePasswordRequest struct {
			OldPassword string `json:"old_password" binding:"required"`
			NewPassword string `json:"new_password" binding:"required"`
		}
		var req ChangePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"ok":      false,
				"message": "请求数据无效",
			})
			return
		}

		userUUIDStr, exists := c.Get("user_uuid")
		if !exists {
			c.JSON(401, gin.H{"ok": false, "message": "未登录"})
			return
		}
		userUUID, _ := uuid.Parse(userUUIDStr.(string))

		// 获取当前用户
		var user models.User
		result := db.Where("id = ?", userUUID).First(&user)
		if result.Error != nil {
			c.JSON(404, gin.H{"ok": false, "message": "未找到用户"})
			return
		}

		// 验证旧密码
		if err := bcrypt.CompareHashAndPassword([]byte(user.PassWord), []byte(req.OldPassword)); err != nil {
			c.JSON(400, gin.H{"ok": false, "message": "密码校验失败"})
			return
		}

		// 哈希新密码
		newHashedPassword, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			c.JSON(500, gin.H{"ok": false, "message": "对新密码哈希时失败"})
			return
		}

		// 更新密码
		result = db.Model(&user).Update("password", newHashedPassword)
		if result.Error != nil {
			c.JSON(500, gin.H{"ok": false, "message": "更新密码时失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":      true,
			"message": "更新密码成功",
		})
	},
	)

	users := base.Group("/users")

	users.GET("/", func(c *gin.Context) {
		var users []models.User
		result := db.Preload("Application").Find(&users)

		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "查询用户失败",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true, "data": users})
	})

	r.Run(":8080")

}
