package email

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

type EmailService struct {
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	fromAddress  string
	fromName     string
}

type Email struct {
	To      []string
	Subject string
	Body    string
	IsHTML  bool
}

func NewEmailService() *EmailService {
	return &EmailService{
		smtpHost:     getEnv("SMTP_HOST", "localhost"),
		smtpPort:     getEnv("SMTP_PORT", "587"),
		smtpUsername: getEnv("SMTP_USERNAME", ""),
		smtpPassword: getEnv("SMTP_PASSWORD", ""),
		fromAddress:  getEnv("SMTP_FROM_ADDRESS", "noreply@example.com"),
		fromName:     getEnv("SMTP_FROM_NAME", "100y SaaS"),
	}
}

func (e *EmailService) Send(email *Email) error {
	if e.smtpHost == "localhost" || e.smtpUsername == "" {
		// In development or if SMTP not configured, just log
		fmt.Printf("EMAIL (would send): To=%v Subject=%s\n", email.To, email.Subject)
		return nil
	}

	auth := smtp.PlainAuth("", e.smtpUsername, e.smtpPassword, e.smtpHost)
	
	msg := e.buildMessage(email)
	addr := e.smtpHost + ":" + e.smtpPort
	
	return smtp.SendMail(addr, auth, e.fromAddress, email.To, []byte(msg))
}

func (e *EmailService) buildMessage(email *Email) string {
	var msg strings.Builder
	
	msg.WriteString("From: " + e.fromName + " <" + e.fromAddress + ">\r\n")
	msg.WriteString("To: " + strings.Join(email.To, ", ") + "\r\n")
	msg.WriteString("Subject: " + email.Subject + "\r\n")
	
	if email.IsHTML {
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	} else {
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}
	
	msg.WriteString("\r\n")
	msg.WriteString(email.Body)
	
	return msg.String()
}

// Common email templates
func (e *EmailService) SendWelcomeEmail(userEmail, userName string) error {
	email := &Email{
		To:      []string{userEmail},
		Subject: "Welcome to 100y SaaS",
		Body: fmt.Sprintf(`Hello %s,

Welcome to 100y SaaS! Your account has been created successfully.

You can now start using the application to manage your items.

Best regards,
100y SaaS Team`, userName),
		IsHTML: false,
	}
	
	return e.Send(email)
}

func (e *EmailService) SendPasswordResetEmail(userEmail, resetToken string) error {
	resetURL := getEnv("BASE_URL", "http://localhost:8080") + "/reset-password?token=" + resetToken
	
	email := &Email{
		To:      []string{userEmail},
		Subject: "Password Reset Request",
		Body: fmt.Sprintf(`A password reset was requested for your account.

Click the following link to reset your password:
%s

This link will expire in 1 hour.

If you did not request this reset, please ignore this email.

Best regards,
100y SaaS Team`, resetURL),
		IsHTML: false,
	}
	
	return e.Send(email)
}

func (e *EmailService) SendSubscriptionLimitEmail(userEmail string, tenantName string, limitType string) error {
	email := &Email{
		To:      []string{userEmail},
		Subject: fmt.Sprintf("Subscription Limit Reached - %s", tenantName),
		Body: fmt.Sprintf(`Your workspace "%s" has reached its %s limit.

To continue using all features, please consider upgrading your subscription.

You can manage your subscription in your account settings.

Best regards,
100y SaaS Team`, tenantName, limitType),
		IsHTML: false,
	}
	
	return e.Send(email)
}

func (e *EmailService) SendUsageSummaryEmail(userEmail string, tenantName string, summary map[string]interface{}) error {
	body := fmt.Sprintf(`Weekly usage summary for "%s":

• Total events: %v
• Active users: %v
• Total items: %v

Thank you for using 100y SaaS!

Best regards,
100y SaaS Team`, 
		tenantName,
		summary["total_events"],
		summary["active_users_24h"],
		summary["total_items"])

	email := &Email{
		To:      []string{userEmail},
		Subject: fmt.Sprintf("Weekly Summary - %s", tenantName),
		Body:    body,
		IsHTML:  false,
	}
	
	return e.Send(email)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
