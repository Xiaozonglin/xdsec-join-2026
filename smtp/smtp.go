package smtp

import (
	"fmt"
	"net/smtp"
	"os"
)

func main() {
	// 配置SMTP服务器信息
	smtpHost := os.Getenv("smtpHost")
	smtpPort := os.Getenv("smtpPort")
	smtpUser := os.Getenv("smtpUser")
	smtpPassword := os.Getenv("smtpPassword")

	// 发件人和收件人
	from := "noreply@xdsec.club"
	to := []string{""}

	// 邮件内容
	subject := "西电信安协会招新系统测试邮件\r\n"
	body := "这是一封测试邮件，由林林通过 Golang 调用 smtp 发出。\r\n"
	message := []byte(subject + "\r\n" + body)

	// 认证信息
	auth := smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)

	// 发送邮件
	err := smtp.SendMail(
		smtpHost+":"+smtpPort,
		auth,
		from,
		to,
		message,
	)

	if err != nil {
		fmt.Println("发送失败:", err)
		return
	}
	fmt.Println("邮件发送成功!")
}
