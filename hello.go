package main

import (
	"net/http"
	"xdsec-join-2026/models"

	"github.com/gin-gonic/gin"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	dsn := "root:root@tcp(127.0.0.1:3306)/xdjoin?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("数据库连接失败: " + err.Error())
	}

	db.AutoMigrate(&models.Application{}, &models.User{})

	r := gin.Default()

	// 定义路由组

	base := r.Group("/api/v1")

	users := base.Group("/users")

	users.GET("/", func(c *gin.Context) {
		var users []models.User
		result := db.Find(&users)

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
