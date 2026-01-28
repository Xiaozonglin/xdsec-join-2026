package handlers

import (
	"encoding/json"
	"net/http"
	"xdsec-join-2026/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tealeg/xlsx/v3"
	"gorm.io/gorm"
)

// ExportApplications 导出申请信息为Excel（面试官）
func ExportApplications(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取所有面试者和他们的申请信息
		var users []models.User
		if err := db.Where("role = ?", "interviewee").Preload("Application").Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "服务器错误"})
			return
		}

		// 创建Excel文件
		file := xlsx.NewFile()
		sheet, err := file.AddSheet("申请者信息")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "创建Excel文件失败"})
			return
		}

		// 设置表头
		headers := []string{
			"用户ID", "邮箱", "昵称", "签名", "面试状态",
			"真实姓名", "手机号", "性别", "学院", "专业", "学号",
			"申请方向", "简历", "通过方向", "通过面试官", "创建时间", "更新时间",
		}
		headerRow := sheet.AddRow()
		for _, header := range headers {
			headerRow.AddCell().Value = header
		}

		// 填充数据
		for _, user := range users {
			row := sheet.AddRow()

			// 用户基本信息
			row.AddCell().Value = user.UUID.String()
			row.AddCell().Value = user.Email
			if user.Nickname != nil {
				row.AddCell().Value = *user.Nickname
			} else {
				row.AddCell().Value = ""
			}
			row.AddCell().Value = user.Signature
			row.AddCell().Value = user.Status

			// 申请信息
			if user.Application != nil {
				app := user.Application

				row.AddCell().Value = app.RealName
				row.AddCell().Value = app.Phone
				row.AddCell().Value = app.Gender
				row.AddCell().Value = app.Department
				row.AddCell().Value = app.Major
				row.AddCell().Value = app.StudentId

				// 解析申请方向
				if app.Directions != "" {
					var directions []string
					json.Unmarshal([]byte(app.Directions), &directions)
					directionsStr := ""
					for i, dir := range directions {
						if i > 0 {
							directionsStr += ", "
						}
						directionsStr += dir
					}
					row.AddCell().Value = directionsStr
				} else {
					row.AddCell().Value = ""
				}

				row.AddCell().Value = app.Resume
			} else {
				// 无申请信息时填充空值
				row.AddCell().Value = ""
				row.AddCell().Value = ""
				row.AddCell().Value = ""
				row.AddCell().Value = ""
				row.AddCell().Value = ""
				row.AddCell().Value = ""
				row.AddCell().Value = ""
				row.AddCell().Value = ""
			}

			// 解析通过方向
			if user.PassedDirections != "" {
				var passedDirections []string
				json.Unmarshal([]byte(user.PassedDirections), &passedDirections)
				passedStr := ""
				for i, dir := range passedDirections {
					if i > 0 {
						passedStr += "&"
					}
					passedStr += dir
				}
				row.AddCell().Value = passedStr
			} else {
				row.AddCell().Value = ""
			}

			// 解析通过面试官
			if user.PassedDirectionsBy != "" {
				var passedByList []string
				json.Unmarshal([]byte(user.PassedDirectionsBy), &passedByList)
				passedByStr := ""
				for i, name := range passedByList {
					if i > 0 {
						passedByStr += ", "
					}
					passedByStr += name
				}
				row.AddCell().Value = passedByStr
			} else {
				row.AddCell().Value = ""
			}

			row.AddCell().Value = user.CreatedAt.Format("2006-01-02 15:04:05")
			row.AddCell().Value = user.UpdatedAt.Format("2006-01-02 15:04:05")
		}

		// 生成文件名
		filename := "applications_" + uuid.New().String()[:8] + ".xlsx"

		// 设置响应头
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", "attachment; filename="+filename)

		// 返回文件
		if err := file.Write(c.Writer); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "生成Excel文件失败"})
			return
		}

		c.Status(http.StatusOK)
	}
}
