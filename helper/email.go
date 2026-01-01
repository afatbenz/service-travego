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
	"service-travego/model"
	"strconv"
	"time"
)

type EmailTemplateData struct {
	Username         string
	OTP              string
	ExpiryMinutes    int
	Year             int
	ResetLink        string // For reset password email
	RequesterName    string // For join organization approval email
	OrganizationName string // For join organization approval email
}

type OrderSuccessEmailData struct {
	CustomerName     string
	OrderID          string
	FleetName        string
	Duration         string
	Facilities       string
	PickupLocation   string
	Destination      string
	TotalPrice       string
	Year             int
	OrganizationLogo string
	BrandName        string
	CompanyName      string
	ContactList      []model.ContentListItem
}

// GetOTPLength returns the OTP length from environment variable or default to 8
func GetOTPLength() int {
	if envLength := os.Getenv("OTP_LENGTH"); envLength != "" {
		if length, err := strconv.Atoi(envLength); err == nil && length > 0 {
			return length
		}
	}
	return 8 // Default to 8 digits
}

// GenerateOTP generates a random OTP with configurable length
// If length is 0 or not provided, it uses GetOTPLength() to get from env or default to 8
func GenerateOTP(length int) string {
	if length <= 0 {
		length = GetOTPLength()
	}

	rand.Seed(time.Now().UnixNano())
	max := 1
	for i := 0; i < length; i++ {
		max *= 10
	}
	return fmt.Sprintf("%0*d", length, rand.Intn(max))
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

func renderEmailTemplate(templatePath string, data interface{}) (string, error) {
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

// SendResetPasswordEmail sends a reset password email with link and token
func SendResetPasswordEmail(cfg *configs.EmailConfig, to, username, resetLink string, expiryMinutes int) error {
	data := EmailTemplateData{
		Username:      username,
		ResetLink:     resetLink,
		ExpiryMinutes: expiryMinutes,
		Year:          time.Now().Year(),
	}

	htmlBody, err := renderEmailTemplate("reset_password.html", data)
	if err != nil {
		return err
	}

	subject := "Reset Your Password - TraveGO"
	return sendHTMLEmail(cfg, to, subject, htmlBody)
}

// SendJoinOrganizationApprovalEmail sends an email to organization members for approval
func SendJoinOrganizationApprovalEmail(cfg *configs.EmailConfig, to, username, requesterUsername, organizationName string) error {
	data := EmailTemplateData{
		Username:         username,
		Year:             time.Now().Year(),
		RequesterName:    requesterUsername,
		OrganizationName: organizationName,
	}

	htmlBody, err := renderEmailTemplate("join_organization_approval.html", data)
	if err != nil {
		return err
	}

	subject := "New Member Request - TraveGO"
	return sendHTMLEmail(cfg, to, subject, htmlBody)
}

func SendOrderSuccessEmail(cfg *configs.EmailConfig, to string, data OrderSuccessEmailData) error {
	data.Year = time.Now().Year()
	htmlBody, err := renderEmailTemplate("order_success.html", data)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("Order Confirmation - %s", data.OrderID)
	return sendHTMLEmail(cfg, to, subject, htmlBody)
}
