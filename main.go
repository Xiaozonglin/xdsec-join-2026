package main

import (
	"log"
	"net/http"
	"os"

	"xdsec-join-2026/handlers"
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
	db.AutoMigrate(&models.User{}, &models.Application{}, &models.Announcement{}, &models.Task{}, &models.EmailCode{})

	r := gin.Default()
	r.RedirectTrailingSlash = false

	// 跨域中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
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
		authRoute.POST("/email-code", handlers.SendEmailCode(db))
		authRoute.POST("/register", handlers.Register(db))
		authRoute.POST("/login", handlers.Login(db))
		authRoute.POST("/logout", handlers.AuthMiddleware(), handlers.Logout())
		authRoute.POST("/reset-password", handlers.ResetPassword(db))
		authRoute.POST("/change-password", handlers.AuthMiddleware(), handlers.ChangePassword(db))
		authRoute.GET("/me", handlers.AuthMiddleware(), handlers.GetCurrentUser(db))
	}

	// 用户与权限
	usersRoute := api.Group("/users")
	{
		usersRoute.GET("", handlers.AuthMiddleware(), handlers.GetUsers(db))
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
		announcementsRoute.GET("", handlers.GetAnnouncements(db))
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
