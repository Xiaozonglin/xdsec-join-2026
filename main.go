package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"xdsec-join-2026/handlers"
	"xdsec-join-2026/middleware"
	"xdsec-join-2026/models"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

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

	// 自动迁移
	db.AutoMigrate(&models.User{}, &models.Application{}, &models.Announcement{}, &models.Task{}, &models.EmailCode{}, &models.EmailRateLimit{})

	// 频率限制中间件（每分钟60次请求）
	rateLimiter := middleware.NewIPRateLimiter(1, 60)

	// 邮箱验证码专用频率限制器（每分钟1次）
	// rate=0.05（每秒恢复0.016个 = 每分钟恢复1个），burst=3（初始1个配额）
	emailCodeRateLimiter := middleware.NewIPRateLimiter(0.016, 1)

	// 启动定时清理任务，每小时清理一次过期的验证码
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			log.Println("开始清理过期的验证码...")
			if err := db.Where("expires_at < ?", time.Now()).Delete(&models.EmailCode{}).Error; err != nil {
				log.Printf("清理过期验证码失败: %v", err)
			} else {
				log.Println("过期验证码清理完成")
			}
		}
	}()

	r := gin.Default()
	r.RedirectTrailingSlash = false

	// 跨域中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// 基础路由组
	api := r.Group("/api/v2")

	// 认证与账号
	authRoute := api.Group("/auth")
	{
		authRoute.POST("/email-code", emailCodeRateLimiter.Middleware(), handlers.SendEmailCode(db))
		authRoute.POST("/register", rateLimiter.Middleware(), handlers.Register(db))
		authRoute.POST("/login", rateLimiter.Middleware(), handlers.Login(db))
		authRoute.POST("/logout", handlers.AuthMiddleware(), handlers.Logout())
		authRoute.POST("/reset-password", rateLimiter.Middleware(), handlers.ResetPassword(db))
		authRoute.POST("/change-password", handlers.AuthMiddleware(), handlers.ChangePassword(db))
		authRoute.GET("/me", handlers.AuthMiddleware(), handlers.GetCurrentUser(db))
	}

	// 用户与权限
	usersRoute := api.Group("/users")
	{
		usersRoute.GET("/", handlers.AuthMiddleware(), handlers.GetUsers(db))
		usersRoute.GET("/:id", handlers.AuthMiddleware(), handlers.RequireInterviewer(), handlers.GetUserDetail(db))
		usersRoute.PATCH("/me", handlers.AuthMiddleware(), handlers.UpdateProfile(db))
		usersRoute.POST("/:id/role", handlers.AuthMiddleware(), handlers.RequireInterviewer(), handlers.SetUserRole(db))
		usersRoute.POST("/:id/passed-directions", handlers.AuthMiddleware(), handlers.RequireInterviewer(), handlers.SetPassedDirections(db))
		usersRoute.DELETE("/:id", handlers.AuthMiddleware(), handlers.RequireInterviewer(), handlers.DeleteUser(db))
		usersRoute.DELETE("/me", handlers.AuthMiddleware(), handlers.DeleteSelf(db))
	}

	// 公告
	announcementsRoute := api.Group("/announcements")
	{
		announcementsRoute.GET("", rateLimiter.Middleware(), handlers.GetAnnouncements(db))
		announcementsRoute.POST("", handlers.AuthMiddleware(), handlers.RequireInterviewer(), handlers.CreateAnnouncement(db))
		announcementsRoute.PATCH("/:id", handlers.AuthMiddleware(), handlers.RequireInterviewer(), handlers.UpdateAnnouncement(db))
		announcementsRoute.POST("/:id/pin", handlers.AuthMiddleware(), handlers.RequireInterviewer(), handlers.PinAnnouncement(db))
		announcementsRoute.DELETE("/:id", handlers.AuthMiddleware(), handlers.RequireInterviewer(), handlers.DeleteAnnouncement(db))
	}

	// 面试申请
	applicationsRoute := api.Group("/applications")
	{
		applicationsRoute.POST("", handlers.AuthMiddleware(), handlers.CreateApplication(db))
		applicationsRoute.GET("/me", handlers.AuthMiddleware(), handlers.GetMyApplication(db))
		applicationsRoute.GET("/:userId", handlers.AuthMiddleware(), handlers.RequireInterviewer(), handlers.GetApplicationDetail(db))
		applicationsRoute.POST("/:userId/status", handlers.AuthMiddleware(), handlers.RequireInterviewer(), handlers.SetInterviewStatus(db))
		applicationsRoute.DELETE("/:userId", handlers.AuthMiddleware(), handlers.RequireInterviewer(), handlers.DeleteApplication(db))
		applicationsRoute.DELETE("/me", handlers.AuthMiddleware(), handlers.DeleteSelfApplication(db))
	}

	// 面试任务
	tasksRoute := api.Group("/tasks")
	{
		tasksRoute.GET("", handlers.AuthMiddleware(), handlers.GetTasks(db))
		tasksRoute.POST("", handlers.AuthMiddleware(), handlers.RequireInterviewer(), handlers.CreateTask(db))
		tasksRoute.PATCH("/:id", handlers.AuthMiddleware(), handlers.RequireInterviewer(), handlers.UpdateTask(db))
		tasksRoute.POST("/:id/report", handlers.AuthMiddleware(), handlers.SubmitTaskReport(db))
		tasksRoute.DELETE("/:id", handlers.AuthMiddleware(), handlers.RequireInterviewer(), handlers.DeleteTask(db))
	}

	// 数据导出
	exportRoute := api.Group("/export")
	{
		exportRoute.GET("/applications", handlers.AuthMiddleware(), handlers.RequireInterviewer(), handlers.ExportApplications(db))
	}

	r.Run(":8080")
}
