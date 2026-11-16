// This file defines the password reset email sender interface and its Gmail SMTP implementation,
// constructing multipart/alternative (plain-text + branded HTML) password reset emails.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"net/smtp"
	"net/url"
	"os"
	"strings"
)

// PasswordResetEmailSender defines the contract for delivering password reset verification links.
type PasswordResetEmailSender interface {
	SendPasswordResetEmail(recipientEmailAddress string, rawResetToken string) error
}

// GmailSMTPPasswordResetEmailSender implements PasswordResetEmailSender using Gmail's SMTP server (smtp.gmail.com:587).
type GmailSMTPPasswordResetEmailSender struct {
	emailUser          string
	emailAppPassword   string
	applicationBaseURL string
	sendMail           smtpSendMailFunction
}

// NewGmailSMTPPasswordResetEmailSender creates a GmailSMTPPasswordResetEmailSender initialized from
// EMAIL_USER, EMAIL_APP_PASSWORD, and APP_URL environment variables (defaulting APP_URL to http://localhost:3000).
func NewGmailSMTPPasswordResetEmailSender() *GmailSMTPPasswordResetEmailSender {
	applicationBaseURL := strings.TrimSpace(os.Getenv("APP_URL"))
	if applicationBaseURL == "" {
		applicationBaseURL = defaultApplicationBaseURL
	}

	return &GmailSMTPPasswordResetEmailSender{
		emailUser:          strings.TrimSpace(os.Getenv("EMAIL_USER")),
		emailAppPassword:   strings.TrimSpace(os.Getenv("EMAIL_APP_PASSWORD")),
		applicationBaseURL: strings.TrimRight(applicationBaseURL, "/"),
		sendMail:           smtp.SendMail,
	}
}

// sanitizeEmailHeaderValue strips carriage returns and line feeds to prevent SMTP header injection.
func sanitizeEmailHeaderValue(rawHeaderValue string) string {
	replacer := strings.NewReplacer("\r", "", "\n", "")
	return strings.TrimSpace(replacer.Replace(rawHeaderValue))
}

// BuildPasswordResetTargetURL constructs the absolute frontend /new-password?token=... URL for password reset emails.
func BuildPasswordResetTargetURL(applicationBaseURL string, rawResetToken string) string {
	normalizedBaseURL := strings.TrimRight(strings.TrimSpace(applicationBaseURL), "/")
	if normalizedBaseURL == "" {
		normalizedBaseURL = defaultApplicationBaseURL
	}

	return fmt.Sprintf("%s/new-password?token=%s", normalizedBaseURL, url.QueryEscape(strings.TrimSpace(rawResetToken)))
}

// generateMIMEBoundary produces a random MIME multipart boundary string.
func generateMIMEBoundary() string {
	boundaryBytes := make([]byte, 12)
	if _, readError := rand.Read(boundaryBytes); readError != nil {
		return "nutrisync-reset-boundary-fallback"
	}

	return "nutrisync-reset-" + hex.EncodeToString(boundaryBytes)
}

// buildPasswordResetPlainTextBody formats the plain-text fallback body for password reset emails.
func buildPasswordResetPlainTextBody(resetURL string) string {
	return strings.Join([]string{
		"NutriSync — Password Reset Request",
		"",
		"We received a request to reset the password for your NutriSync account.",
		"Use the link below to choose a new password. For your security, this link expires in 1 hour and can only be used once.",
		"",
		resetURL,
		"",
		"If you did not request a password reset, you can safely ignore this email—your current password will remain unchanged.",
		"",
		"— The NutriSync Team",
	}, "\r\n")
}

// buildPasswordResetHTMLBody formats the responsive inline-CSS HTML body for password reset emails.
func buildPasswordResetHTMLBody(resetURL string) string {
	escapedResetURL := html.EscapeString(resetURL)

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Reset your NutriSync password</title>
</head>
<body style="margin:0;padding:0;background-color:#F8FAFC;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:#0F172A;">
  <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="background-color:#F8FAFC;padding:32px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="max-width:560px;background-color:#FFFFFF;border:1px solid #E2E8F0;border-radius:16px;overflow:hidden;">
          <tr>
            <td style="height:6px;background:linear-gradient(90deg,#0B74B8 0%%,#18A999 50%%,#43B047 100%%);background-color:#0B74B8;font-size:0;line-height:0;">&nbsp;</td>
          </tr>
          <tr>
            <td style="padding:32px 32px 24px 32px;">
              <p style="margin:0 0 4px 0;font-size:20px;font-weight:700;letter-spacing:-0.02em;color:#0F172A;">NutriSync</p>
              <p style="margin:0 0 24px 0;font-size:12px;font-weight:500;letter-spacing:0.04em;text-transform:uppercase;color:#18A999;">Personalized AI Nutrition</p>

              <h1 style="margin:0 0 12px 0;font-size:22px;line-height:1.3;font-weight:700;color:#0F172A;">Reset your password</h1>
              <p style="margin:0 0 16px 0;font-size:15px;line-height:1.6;color:#334155;">
                We received a request to reset the password for your NutriSync account. Click the button below to choose a new password.
              </p>

              <table role="presentation" cellspacing="0" cellpadding="0" border="0" style="margin:24px 0;">
                <tr>
                  <td align="center" style="border-radius:10px;background-color:#0B74B8;">
                    <a href="%s" target="_blank" rel="noopener noreferrer" style="display:inline-block;padding:13px 26px;font-size:15px;font-weight:600;color:#FFFFFF;text-decoration:none;border-radius:10px;">
                      Reset Password
                    </a>
                  </td>
                </tr>
              </table>

              <p style="margin:0 0 16px 0;font-size:13px;line-height:1.6;color:#475569;background-color:#F1F5F9;border-radius:8px;padding:12px 14px;">
                <strong>Security notice:</strong> This link expires in <strong>1 hour</strong> and can only be used once. If you did not request a password reset, you can safely ignore this email—your password will remain unchanged.
              </p>

              <p style="margin:0 0 6px 0;font-size:12px;line-height:1.5;color:#64748B;">
                If the button above does not work, copy and paste this link into your browser:
              </p>
              <p style="margin:0;font-size:12px;line-height:1.5;word-break:break-all;color:#0B74B8;">
                <a href="%s" target="_blank" rel="noopener noreferrer" style="color:#0B74B8;text-decoration:underline;">%s</a>
              </p>
            </td>
          </tr>
          <tr>
            <td style="padding:16px 32px 24px 32px;border-top:1px solid #E2E8F0;">
              <p style="margin:0;font-size:12px;line-height:1.5;color:#94A3B8;">
                Sent by NutriSync. Please do not reply directly to this automated message.
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, escapedResetURL, escapedResetURL, escapedResetURL)
}

// BuildPasswordResetEmailMessage constructs a multipart/alternative MIME email containing both plain-text
// and branded HTML password reset instructions with CRLF-sanitized headers.
func BuildPasswordResetEmailMessage(
	senderEmailAddress string,
	recipientEmailAddress string,
	applicationBaseURL string,
	rawResetToken string,
) []byte {
	safeSender := sanitizeEmailHeaderValue(senderEmailAddress)
	safeRecipient := sanitizeEmailHeaderValue(recipientEmailAddress)
	resetURL := BuildPasswordResetTargetURL(applicationBaseURL, rawResetToken)
	boundary := generateMIMEBoundary()

	plainTextBody := buildPasswordResetPlainTextBody(resetURL)
	htmlBody := buildPasswordResetHTMLBody(resetURL)

	messageParts := []string{
		fmt.Sprintf("From: NutriSync <%s>", safeSender),
		fmt.Sprintf("To: %s", safeRecipient),
		"Subject: Reset your NutriSync password",
		"MIME-Version: 1.0",
		fmt.Sprintf(`Content-Type: multipart/alternative; boundary="%s"`, boundary),
		"",
		fmt.Sprintf("--%s", boundary),
		`Content-Type: text/plain; charset="UTF-8"`,
		"Content-Transfer-Encoding: 8bit",
		"",
		plainTextBody,
		"",
		fmt.Sprintf("--%s", boundary),
		`Content-Type: text/html; charset="UTF-8"`,
		"Content-Transfer-Encoding: 8bit",
		"",
		htmlBody,
		"",
		fmt.Sprintf("--%s--", boundary),
		"",
	}

	return []byte(strings.Join(messageParts, "\r\n"))
}

// SendPasswordResetEmail sends a multipart/alternative password reset email via smtp.gmail.com:587.
func (emailSender *GmailSMTPPasswordResetEmailSender) SendPasswordResetEmail(
	recipientEmailAddress string,
	rawResetToken string,
) error {
	if strings.TrimSpace(recipientEmailAddress) == "" {
		return errors.New("recipient email address must not be empty")
	}

	if strings.TrimSpace(rawResetToken) == "" {
		return errors.New("password reset token must not be empty")
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
	emailMessageBytes := BuildPasswordResetEmailMessage(senderEmailUser, recipientEmailAddress, applicationBaseURL, rawResetToken)

	return sendMailFunction(
		gmailSMTPAddress,
		smtpAuthentication,
		senderEmailUser,
		[]string{sanitizeEmailHeaderValue(recipientEmailAddress)},
		emailMessageBytes,
	)
}
