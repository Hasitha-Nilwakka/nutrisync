// This file defines the email verification sender abstraction and its Gmail SMTP implementation
// used during the user registration workflow to deliver account verification links.
package auth

import (
	"errors"
	"fmt"
	"html"
	"net/smtp"
	"net/url"
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

// BuildRegistrationVerificationTargetURL constructs the absolute frontend /verify-email?token=... URL.
func BuildRegistrationVerificationTargetURL(applicationBaseURL string, verificationToken string) string {
	normalizedBaseURL := strings.TrimRight(strings.TrimSpace(applicationBaseURL), "/")
	if normalizedBaseURL == "" {
		normalizedBaseURL = defaultApplicationBaseURL
	}

	return fmt.Sprintf("%s/verify-email?token=%s", normalizedBaseURL, url.QueryEscape(strings.TrimSpace(verificationToken)))
}

// buildRegistrationVerificationPlainTextBody formats the plain-text fallback body for verification emails.
func buildRegistrationVerificationPlainTextBody(verificationURL string) string {
	return strings.Join([]string{
		"NutriSync — Email Verification",
		"",
		"Thank you for signing up for NutriSync! We received a registration request for your account.",
		"Use the link below to verify your email address and activate your account. For your security, this link expires in 24 hours and can only be used once.",
		"",
		verificationURL,
		"",
		"If you did not create a NutriSync account, you can safely ignore this email—no account will be activated.",
		"",
		"— The NutriSync Team",
	}, "\r\n")
}

// buildRegistrationVerificationHTMLBody formats the responsive inline-CSS HTML body for verification emails.
func buildRegistrationVerificationHTMLBody(verificationURL string) string {
	escapedVerificationURL := html.EscapeString(verificationURL)

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Verify your NutriSync email</title>
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

              <h1 style="margin:0 0 12px 0;font-size:22px;line-height:1.3;font-weight:700;color:#0F172A;">Verify your email address</h1>
              <p style="margin:0 0 16px 0;font-size:15px;line-height:1.6;color:#334155;">
                Thank you for signing up for NutriSync! Click the button below to verify your email address and activate your account.
              </p>

              <table role="presentation" cellspacing="0" cellpadding="0" border="0" style="margin:24px 0;">
                <tr>
                  <td align="center" style="border-radius:10px;background-color:#0B74B8;">
                    <a href="%s" target="_blank" rel="noopener noreferrer" style="display:inline-block;padding:13px 26px;font-size:15px;font-weight:600;color:#FFFFFF;text-decoration:none;border-radius:10px;">
                      Verify Email
                    </a>
                  </td>
                </tr>
              </table>

              <p style="margin:0 0 16px 0;font-size:13px;line-height:1.6;color:#475569;background-color:#F1F5F9;border-radius:8px;padding:12px 14px;">
                <strong>Security notice:</strong> This verification link expires in <strong>24 hours</strong> and can only be used once. If you did not create a NutriSync account, you can safely ignore this email—no account will be activated.
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
</html>`, escapedVerificationURL, escapedVerificationURL, escapedVerificationURL)
}

// BuildRegistrationVerificationEmailMessage constructs a multipart/alternative MIME email containing both plain-text
// and branded HTML verification instructions pointing to ${APP_URL}/verify-email?token=${verificationToken}.
func BuildRegistrationVerificationEmailMessage(senderEmailAddress string, recipientEmailAddress string, applicationBaseURL string, verificationToken string) []byte {
	safeSender := sanitizeEmailHeaderValue(senderEmailAddress)
	safeRecipient := sanitizeEmailHeaderValue(recipientEmailAddress)
	verificationURL := BuildRegistrationVerificationTargetURL(applicationBaseURL, verificationToken)
	boundary := generateMIMEBoundary()

	plainTextBody := buildRegistrationVerificationPlainTextBody(verificationURL)
	htmlBody := buildRegistrationVerificationHTMLBody(verificationURL)

	messageParts := []string{
		fmt.Sprintf("From: NutriSync <%s>", safeSender),
		fmt.Sprintf("To: %s", safeRecipient),
		"Subject: Verify your NutriSync email",
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
