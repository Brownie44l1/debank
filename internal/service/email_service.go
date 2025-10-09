package service

import (
	"fmt"

	"github.com/Brownie44l1/debank/internal/models"
)

// ==============================================
// EMAIL SERVICE
// ==============================================

type EmailService struct {
	// Add your email provider config here (SMTP, SendGrid, etc.)
	// smtpHost     string
	// smtpPort     int
	// smtpUsername string
	// smtpPassword string
	// fromEmail    string
}

func NewEmailService() *EmailService {
	return &EmailService{
		// Initialize with config from environment
	}
}

// ==============================================
// SEND OTP
// ==============================================

// SendOTP sends an OTP code via email
func (s *EmailService) SendOTP(email, code, purpose string) error {
	subject, body := s.getOTPEmailContent(code, purpose)
	
	// TODO: Implement actual email sending
	// For now, just log it (you can use SMTP, SendGrid, AWS SES, etc.)
	fmt.Printf("📧 Sending OTP to %s\n", email)
	fmt.Printf("Subject: %s\n", subject)
	fmt.Printf("Code: %s\n", code)
	fmt.Printf("Body: %s\n", body)
	
	// Example using net/smtp:
	// return s.sendViaSMTP(email, subject, body)
	
	// Example using SendGrid:
	// return s.sendViaSendGrid(email, subject, body)
	
	return nil
}

// ==============================================
// EMAIL TEMPLATES
// ==============================================

func (s *EmailService) getOTPEmailContent(code, purpose string) (subject string, body string) {
	switch purpose {
	case models.OTPPurposeEmailVerify:
		subject = "Verify Your Email - DeBank"
		body = fmt.Sprintf(`
Hello,

Thank you for signing up with DeBank!

Your email verification code is: %s

This code will expire in 10 minutes.

If you didn't request this code, please ignore this email.

Best regards,
DeBank Team
		`, code)

	case models.OTPPurposePasswordReset:
		subject = "Reset Your Password - DeBank"
		body = fmt.Sprintf(`
Hello,

We received a request to reset your password.

Your password reset code is: %s

This code will expire in 10 minutes.

If you didn't request this, please ignore this email and your password will remain unchanged.

Best regards,
DeBank Team
		`, code)

	case models.OTPPurposeTransactionAuth:
		subject = "Authorize Transaction - DeBank"
		body = fmt.Sprintf(`
Hello,

Please use this code to authorize your transaction:

Authorization code: %s

This code will expire in 10 minutes.

If you didn't initiate this transaction, please contact support immediately.

Best regards,
DeBank Team
		`, code)

	default:
		subject = "Your Verification Code - DeBank"
		body = fmt.Sprintf(`
Hello,

Your verification code is: %s

This code will expire in 10 minutes.

Best regards,
DeBank Team
		`, code)
	}

	return subject, body
}

// ==============================================
// EMAIL SENDING IMPLEMENTATIONS
// ==============================================

// SendViaSMTP sends email using SMTP
// func (s *EmailService) sendViaSMTP(to, subject, body string) error {
// 	auth := smtp.PlainAuth("", s.smtpUsername, s.smtpPassword, s.smtpHost)
// 	
// 	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to, subject, body))
// 	
// 	addr := fmt.Sprintf("%s:%d", s.smtpHost, s.smtpPort)
// 	return smtp.SendMail(addr, auth, s.fromEmail, []string{to}, msg)
// }

// SendViaSendGrid sends email using SendGrid API
// func (s *EmailService) sendViaSendGrid(to, subject, body string) error {
// 	// Implement SendGrid integration
// 	return nil
// }

// SendWelcomeEmail sends a welcome email to new users
func (s *EmailService) SendWelcomeEmail(email, name string) error {
	subject := "Welcome to DeBank!"
	body := fmt.Sprintf(`
Hello %s,

Welcome to DeBank! Your account has been successfully created.

You can now:
- Send and receive money instantly
- Check your balance anytime
- View your transaction history

Thank you for choosing DeBank!

Best regards,
DeBank Team
	`, name)

	fmt.Printf("📧 Sending welcome email to %s\n", email)
	fmt.Printf("Subject: %s\n", subject)
	
	// TODO: Implement actual email sending
	return nil
}

// SendTransactionNotification sends transaction notification
func (s *EmailService) SendTransactionNotification(email, transactionType string, amount int64) error {
	subject := fmt.Sprintf("Transaction %s - DeBank", transactionType)
	amountNGN := float64(amount) / 100.0
	
	body := fmt.Sprintf(`
Hello,

A %s transaction of ₦%.2f has been processed on your account.

Transaction type: %s
Amount: ₦%.2f

If you didn't authorize this transaction, please contact support immediately.

Best regards,
DeBank Team
	`, transactionType, amountNGN, transactionType, amountNGN)

	fmt.Printf("📧 Sending transaction notification to %s\n", email)
	
	// TODO: Implement actual email sending
	return nil
}