package mailer

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/eric/callit-backend/internal/core/ports"
)

//go:embed templates/*.html
var templatesFS embed.FS

// Real sender via Resend (https://resend.com) — picked at boot only when
// RESEND_API_KEY is set (see cmd/api/main.go), same swap pattern as
// mollah-backend's Brevo/mock split.
type resendEmailService struct {
	apiKey      string
	senderEmail string
	senderName  string
	logger      *slog.Logger
}

func NewResendEmailService(apiKey, senderEmail, senderName string, logger *slog.Logger) ports.EmailService {
	return &resendEmailService{apiKey: apiKey, senderEmail: senderEmail, senderName: senderName, logger: logger}
}

type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Html    string   `json:"html"`
}

func renderTemplate(filename string, vars map[string]string) (string, error) {
	b, err := templatesFS.ReadFile("templates/" + filename)
	if err != nil {
		return "", err
	}
	content := string(b)
	for k, v := range vars {
		content = strings.ReplaceAll(content, "{{"+k+"}}", v)
	}
	return content, nil
}

func (r *resendEmailService) send(toEmail, subject, html string) error {
	payload := resendPayload{
		From:    fmt.Sprintf("%s <%s>", r.senderName, r.senderEmail),
		To:      []string{toEmail},
		Subject: subject,
		Html:    html,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(res.Body)
		r.logger.Error("resend send failed", "status", res.StatusCode, "body", string(respBytes))
		return fmt.Errorf("resend API error (%d): %s", res.StatusCode, string(respBytes))
	}
	return nil
}

func (r *resendEmailService) SendVerificationEmail(toEmail, code string) error {
	html, err := renderTemplate("login.html", map[string]string{
		"verification_code": code,
		"support_email":     r.senderEmail,
	})
	if err != nil {
		r.logger.Error("failed to render login.html", "error", err)
		html = fmt.Sprintf(`<p>Your Callit verification code is <strong>%s</strong>. It expires in 10 minutes.</p>`, code)
	}
	return r.send(toEmail, fmt.Sprintf("%s is your Callit verification code", code), html)
}

func (r *resendEmailService) SendWelcomeEmail(toEmail, name string) error {
	firstName := strings.TrimSpace(name)
	if parts := strings.Fields(firstName); len(parts) > 0 {
		firstName = parts[0]
	}
	if firstName == "" {
		firstName = "there"
	}
	html, err := renderTemplate("welcome.html", map[string]string{
		"first_name":       firstName,
		"app_link":         "https://callit.my",
		"support_email":    r.senderEmail,
		"unsubscribe_link": "https://callit.my/unsubscribe",
	})
	if err != nil {
		r.logger.Error("failed to render welcome.html", "error", err)
		html = fmt.Sprintf(`<p>Welcome to Callit, %s! Your account is ready.</p>`, firstName)
	}
	return r.send(toEmail, "Welcome to Callit", html)
}

func (r *resendEmailService) SendGenericEmail(toEmail, subject, heading, body, buttonLabel, buttonUrl string) error {
	html, err := renderTemplate("layout.html", map[string]string{
		"preheader":        heading,
		"heading":          heading,
		"body":             body,
		"button_label":     buttonLabel,
		"button_url":       buttonUrl,
		"app_link":         buttonUrl,
		"support_email":    r.senderEmail,
		"unsubscribe_link": "https://callit.my/unsubscribe",
	})
	if err != nil {
		r.logger.Error("failed to render layout.html", "error", err)
		html = fmt.Sprintf(`<h2>%s</h2><p>%s</p>`, heading, body)
	}
	return r.send(toEmail, subject, html)
}
