// This file contains unit tests for GmailSMTPVerificationEmailSender and BuildRegistrationVerificationEmailMessage.
package auth

import (
	"errors"
	"net/smtp"
	"strings"
	"testing"
)

// TestBuildRegistrationVerificationEmailMessageIncludesExpectedVerificationLink verifies that
// BuildRegistrationVerificationEmailMessage formats the HTML anchor tag with APP_URL and token.
func TestBuildRegistrationVerificationEmailMessageIncludesExpectedVerificationLink(testingInstance *testing.T) {
	messageBytes := BuildRegistrationVerificationEmailMessage(
		"sender@gmail.com",
		"recipient@example.com",
		"",
		"token-12345",
	)

	messageString := string(messageBytes)
	expectedURL := "http://localhost:3000/verify-email?token=token-12345"
	if !strings.Contains(messageString, expectedURL) {
		testingInstance.Errorf("expected email message to contain %q, got %q", expectedURL, messageString)
	}
	if !strings.Contains(messageString, `Content-Type: multipart/alternative`) {
		testingInstance.Errorf("expected multipart/alternative content type, got %q", messageString)
	}
	if !strings.Contains(messageString, "24 hours") {
		testingInstance.Errorf("expected 24 hours expiration notice, got %q", messageString)
	}
}

// TestGmailSMTPVerificationEmailSenderSendsFormattedEmailViaSMTP verifies that
// SendRegistrationVerificationEmail invokes smtp.SendMail against smtp.gmail.com:587 with the expected HTML body.
func TestGmailSMTPVerificationEmailSenderSendsFormattedEmailViaSMTP(testingInstance *testing.T) {
	var capturedAddress string
	var capturedFrom string
	var capturedRecipients []string
	var capturedMessage string

	emailSender := &GmailSMTPVerificationEmailSender{
		emailUser:          "noreply@gmail.com",
		emailAppPassword:   "app-password-secret",
		applicationBaseURL: "http://localhost:3000",
		sendMail: func(address string, authentication smtp.Auth, fromAddress string, recipientAddresses []string, message []byte) error {
			capturedAddress = address
			capturedFrom = fromAddress
			capturedRecipients = recipientAddresses
			capturedMessage = string(message)
			return nil
		},
	}

	sendError := emailSender.SendRegistrationVerificationEmail("newuser@example.com", "abc123token")
	if sendError != nil {
		testingInstance.Fatalf("expected SendRegistrationVerificationEmail to succeed, got error: %v", sendError)
	}

	if capturedAddress != "smtp.gmail.com:587" {
		testingInstance.Errorf("expected SMTP address %q, got %q", "smtp.gmail.com:587", capturedAddress)
	}

	if capturedFrom != "noreply@gmail.com" {
		testingInstance.Errorf("expected sender %q, got %q", "noreply@gmail.com", capturedFrom)
	}

	if len(capturedRecipients) != 1 || capturedRecipients[0] != "newuser@example.com" {
		testingInstance.Errorf("expected recipient [newuser@example.com], got %v", capturedRecipients)
	}

	expectedLink := "http://localhost:3000/verify-email?token=abc123token"
	if !strings.Contains(capturedMessage, expectedLink) {
		testingInstance.Errorf("expected email body to contain %q, got %q", expectedLink, capturedMessage)
	}
}

// TestGmailSMTPVerificationEmailSenderValidationAndErrorPropagation verifies input validation
// and SMTP error propagation on GmailSMTPVerificationEmailSender.
func TestGmailSMTPVerificationEmailSenderValidationAndErrorPropagation(testingInstance *testing.T) {
	unconfiguredSender := NewGmailSMTPVerificationEmailSender()
	if err := unconfiguredSender.SendRegistrationVerificationEmail("", "token"); err == nil {
		testingInstance.Fatal("expected error when recipientEmailAddress is empty")
	}

	if err := unconfiguredSender.SendRegistrationVerificationEmail("user@example.com", ""); err == nil {
		testingInstance.Fatal("expected error when verificationToken is empty")
	}

	failingSender := &GmailSMTPVerificationEmailSender{
		emailUser:          "noreply@gmail.com",
		emailAppPassword:   "app-password-secret",
		applicationBaseURL: "http://localhost:3000",
		sendMail: func(address string, authentication smtp.Auth, fromAddress string, recipientAddresses []string, message []byte) error {
			return errors.New("simulated SMTP failure")
		},
	}

	if err := failingSender.SendRegistrationVerificationEmail("user@example.com", "token"); err == nil {
		testingInstance.Fatal("expected error when underlying SMTP send fails")
	}
}
