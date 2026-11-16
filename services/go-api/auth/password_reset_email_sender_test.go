package auth

import (
	"net/smtp"
	"strings"
	"testing"
)

func TestBuildPasswordResetEmailMessage_ConstructsMultipartAndSanitizesHeaders(t *testing.T) {
	messageBytes := BuildPasswordResetEmailMessage(
		"noreply@nutrisync.app\r\nBcc: evil@attacker.test",
		"user@example.com\r\nCc: leak@attacker.test",
		"https://nutrisync.app/",
		"abc123token",
	)
	messageText := string(messageBytes)

	if strings.Contains(messageText, "\r\nBcc:") || strings.Contains(messageText, "\r\nCc:") {
		t.Fatalf("expected CRLF header injection to be stripped, got:\n%s", messageText)
	}

	requiredSnippets := []string{
		"Subject: Reset your NutriSync password",
		`Content-Type: multipart/alternative; boundary="nutrisync-reset-`,
		`Content-Type: text/plain; charset="UTF-8"`,
		`Content-Type: text/html; charset="UTF-8"`,
		"https://nutrisync.app/new-password?token=abc123token",
		"1 hour",
		"Reset Password",
		"safely ignore this email",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(messageText, snippet) {
			t.Fatalf("expected email message to contain %q", snippet)
		}
	}
}

func TestGmailSMTPPasswordResetEmailSender_SendPasswordResetEmail(t *testing.T) {
	var capturedAddress string
	var capturedFrom string
	var capturedRecipients []string
	var capturedBody string

	sender := &GmailSMTPPasswordResetEmailSender{
		emailUser:          "sender@nutrisync.app",
		emailAppPassword:   "app-password",
		applicationBaseURL: "http://localhost:3000",
		sendMail: func(address string, authentication smtp.Auth, fromAddress string, recipientAddresses []string, message []byte) error {
			capturedAddress = address
			capturedFrom = fromAddress
			capturedRecipients = recipientAddresses
			capturedBody = string(message)
			return nil
		},
	}

	if err := sender.SendPasswordResetEmail("member@example.com", "raw-reset-token-123"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if capturedAddress != gmailSMTPAddress {
		t.Fatalf("expected SMTP address %s, got %s", gmailSMTPAddress, capturedAddress)
	}

	if capturedFrom != "sender@nutrisync.app" {
		t.Fatalf("expected from address sender@nutrisync.app, got %s", capturedFrom)
	}

	if len(capturedRecipients) != 1 || capturedRecipients[0] != "member@example.com" {
		t.Fatalf("expected single recipient member@example.com, got %v", capturedRecipients)
	}

	if !strings.Contains(capturedBody, "http://localhost:3000/new-password?token=raw-reset-token-123") {
		t.Fatalf("expected reset URL in email body, got:\n%s", capturedBody)
	}
}

func TestGmailSMTPPasswordResetEmailSender_ValidationErrors(t *testing.T) {
	sender := &GmailSMTPPasswordResetEmailSender{
		emailUser:        "sender@nutrisync.app",
		emailAppPassword: "app-password",
	}

	if err := sender.SendPasswordResetEmail("", "token"); err == nil {
		t.Fatalf("expected error for empty recipient")
	}

	if err := sender.SendPasswordResetEmail("member@example.com", " "); err == nil {
		t.Fatalf("expected error for empty token")
	}
}
