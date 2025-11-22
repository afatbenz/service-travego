package helper

import (
	"bytes"
	"fmt"
	"html/template"
	"math/rand"
	"net/smtp"
	"os"
	"path/filepath"
	"service-travego/configs"
	"time"
)

type EmailTemplateData struct {
	Username      string
	OTP           string
	ExpiryMinutes int
	Year          int
}

func GenerateOTP() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%08d", rand.Intn(100000000))
}

func getTemplatePath(filename string) (string, error) {
	possiblePaths := []string{
		filepath.Join("templates", "email", filename),
		filepath.Join("..", "templates", "email", filename),
		filepath.Join(".", "templates", "email", filename),
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("template file not found: %s", filename)
}

func renderEmailTemplate(templatePath string, data EmailTemplateData) (string, error) {
	path, err := getTemplatePath(templatePath)
	if err != nil {
		return "", err
	}

	tmpl, err := template.ParseFiles(path)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

func sendHTMLEmail(cfg *configs.EmailConfig, to, subject, htmlBody string) error {
	from := cfg.From
	password := cfg.Password
	smtpHost := cfg.SMTPHost
	smtpPort := cfg.SMTPPort

	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + htmlBody

	auth := smtp.PlainAuth("", from, password, smtpHost)
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func SendOTPEmail(cfg *configs.EmailConfig, to, username, otp string) error {
	data := EmailTemplateData{
		Username:      username,
		OTP:           otp,
		ExpiryMinutes: 5,
		Year:          time.Now().Year(),
	}

	htmlBody, err := renderEmailTemplate("otp_register.html", data)
	if err != nil {
		return err
	}

	subject := "Verify Your Email - TraveGO"
	return sendHTMLEmail(cfg, to, subject, htmlBody)
}

func SendRegisterSuccessEmail(cfg *configs.EmailConfig, to, username string) error {
	data := EmailTemplateData{
		Username: username,
		Year:     time.Now().Year(),
	}

	htmlBody, err := renderEmailTemplate("register_success.html", data)
	if err != nil {
		return err
	}

	subject := "Welcome to TraveGO - Registration Successful"
	return sendHTMLEmail(cfg, to, subject, htmlBody)
}

func SendResetPasswordOTPEmail(cfg *configs.EmailConfig, to, username, otp string) error {
	data := EmailTemplateData{
		Username:      username,
		OTP:           otp,
		ExpiryMinutes: 5,
		Year:          time.Now().Year(),
	}

	htmlBody, err := renderEmailTemplate("otp_reset_password.html", data)
	if err != nil {
		return err
	}

	subject := "Reset Your Password - TraveGO"
	return sendHTMLEmail(cfg, to, subject, htmlBody)
}
