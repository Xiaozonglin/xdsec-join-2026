package smtp

import (
	"fmt"
	"net/smtp"
	"os"
)

// SendEmailCode 发送邮箱验证码
func SendEmailCode(to, code, purpose string) error {
	smtpHost := os.Getenv("smtpHost")
	smtpPort := os.Getenv("smtpPort")
	smtpUser := os.Getenv("smtpUser")
	smtpPassword := os.Getenv("smtpPassword")

	from := os.Getenv("smtpUser")

	// 根据不同用途生成不同的邮件内容
	var subject, body string
	switch purpose {
	case "register":
		subject = "[XDSec Recruitment System] 注册验证码"
		body = fmt.Sprintf("您的注册验证码是：%s\n\n该验证码5分钟内有效，请勿泄露给他人。\n\n如果这不是您本人的操作，请忽略此邮件。", code)
	case "reset":
		subject = "[XDSec Recruitment System] 密码重置验证码"
		body = fmt.Sprintf("您的密码重置验证码是：%s\n\n该验证码5分钟内有效，请勿泄露给他人。\n\n如果这不是您本人的操作，请忽略此邮件。", code)
	case "profile":
		subject = "[XDSec Recruitment System] 个人信息修改验证码"
		body = fmt.Sprintf("您的个人信息修改验证码是：%s\n\n该验证码5分钟内有效，请勿泄露给他人。\n\n如果这不是您本人的操作，请忽略此邮件。", code)
	default:
		return fmt.Errorf("invalid email purpose: %s", purpose)
	}

	message := []byte("Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		body + "\r\n")

	auth := smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)

	err := smtp.SendMail(
		smtpHost+":"+smtpPort,
		auth,
		from,
		[]string{to},
		message,
	)

	return err
}
