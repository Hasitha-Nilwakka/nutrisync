// This file defines the email verification sender abstraction and its Gmail SMTP implementation
// used during the user registration workflow to deliver account verification links.
package auth

import (
	"errors"
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

const (
	defaultApplicationBaseURL = "http://localhost:3000"
	gmailSMTPHost             = "smtp.gmail.com"
	gmailSMTPAddress          = "smtp.gmail.com:587"
)

// VerificationEmailSender defines the contract for sending registration verification emails.
// Consumed by handlers.UserRegistrationHandler after creating an unverified user record.
type VerificationEmailSender interface {
	SendRegistrationVerificationEmail(recipientEmailAddress string, verificationToken string) error
}

// smtpSendMailFunction represents the signature of net/smtp.SendMail so tests can verify
// SMTP payload construction without opening external network connections.
type smtpSendMailFunction func(address string, authentication smtp.Auth, fromAddress string, recipientAddresses []string, message []byte) error

// GmailSMTPVerificationEmailSender implements VerificationEmailSender using Gmail's SMTP server (smtp.gmail.com:587).
type GmailSMTPVerificationEmailSender struct {
	emailUser          string
	emailAppPassword   string
	applicationBaseURL string
	sendMail           smtpSendMailFunction
}

// NewGmailSMTPVerificationEmailSender creates a GmailSMTPVerificationEmailSender initialized from
// EMAIL_USER, EMAIL_APP_PASSWORD, and APP_URL environment variables (defaulting APP_URL to http://localhost:3000).
func NewGmailSMTPVerificationEmailSender() *GmailSMTPVerificationEmailSender {
	applicationBaseURL := strings.TrimSpace(os.Getenv("APP_URL"))
	if applicationBaseURL == "" {
		applicationBaseURL = defaultApplicationBaseURL
	}

	return &GmailSMTPVerificationEmailSender{
		emailUser:          strings.TrimSpace(os.Getenv("EMAIL_USER")),
		emailAppPassword:   strings.TrimSpace(os.Getenv("EMAIL_APP_PASSWORD")),
		applicationBaseURL: strings.TrimRight(applicationBaseURL, "/"),
		sendMail:           smtp.SendMail,
	}
}

// BuildRegistrationVerificationEmailMessage formats the MIME headers and HTML body containing
// the verification link <a href="${APP_URL}/api/auth/verify?token=${verificationToken}">Verify Email</a>.
func BuildRegistrationVerificationEmailMessage(senderEmailAddress string, recipientEmailAddress string, applicationBaseURL string, verificationToken string) []byte {
	normalizedBaseURL := strings.TrimRight(strings.TrimSpace(applicationBaseURL), "/")
	if normalizedBaseURL == "" {
		normalizedBaseURL = defaultApplicationBaseURL
	}

	verificationLink := fmt.Sprintf(`<a href="%s/api/auth/verify?token=%s">Verify Email</a>`, normalizedBaseURL, verificationToken)
	emailMessage := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: Verify your email\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n<p>Please verify your email by clicking the link below:</p><p>%s</p>",
		senderEmailAddress,
		recipientEmailAddress,
		verificationLink,
	)

	return []byte(emailMessage)
}

// SendRegistrationVerificationEmail sends an HTML verification email via smtp.gmail.com:587 to recipientEmailAddress.
// Reads EMAIL_USER, EMAIL_APP_PASSWORD, and APP_URL (defaulting to http://localhost:3000) and returns an error if inputs or SMTP transmission fail.
func (emailSender *GmailSMTPVerificationEmailSender) SendRegistrationVerificationEmail(recipientEmailAddress string, verificationToken string) error {
	if strings.TrimSpace(recipientEmailAddress) == "" {
		return errors.New("recipient email address must not be empty")
	}

	if strings.TrimSpace(verificationToken) == "" {
		return errors.New("verification token must not be empty")
	}

	senderEmailUser := ""
	senderEmailPassword := ""
	applicationBaseURL := defaultApplicationBaseURL
	sendMailFunction := smtp.SendMail

	if emailSender != nil {
		senderEmailUser = strings.TrimSpace(emailSender.emailUser)
		senderEmailPassword = strings.TrimSpace(emailSender.emailAppPassword)
		if strings.TrimSpace(emailSender.applicationBaseURL) != "" {
			applicationBaseURL = strings.TrimRight(strings.TrimSpace(emailSender.applicationBaseURL), "/")
		}
		if emailSender.sendMail != nil {
			sendMailFunction = emailSender.sendMail
		}
	}

	if senderEmailUser == "" {
		senderEmailUser = strings.TrimSpace(os.Getenv("EMAIL_USER"))
	}
	if senderEmailPassword == "" {
		senderEmailPassword = strings.TrimSpace(os.Getenv("EMAIL_APP_PASSWORD"))
	}
	if applicationBaseURL == defaultApplicationBaseURL {
		if environmentAppURL := strings.TrimSpace(os.Getenv("APP_URL")); environmentAppURL != "" {
			applicationBaseURL = strings.TrimRight(environmentAppURL, "/")
		}
	}

	if senderEmailUser == "" || senderEmailPassword == "" {
		return errors.New("EMAIL_USER and EMAIL_APP_PASSWORD must be configured")
	}

	smtpAuthentication := smtp.PlainAuth("", senderEmailUser, senderEmailPassword, gmailSMTPHost)
	emailMessageBytes := BuildRegistrationVerificationEmailMessage(senderEmailUser, recipientEmailAddress, applicationBaseURL, verificationToken)

	return sendMailFunction(
		gmailSMTPAddress,
		smtpAuthentication,
		senderEmailUser,
		[]string{recipientEmailAddress},
		emailMessageBytes,
	)
}
