package mailer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/eric/callit-backend/internal/core/ports"
)

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

func (r *resendEmailService) SendVerificationEmail(toEmail, code string) error {
	payload := resendPayload{
		From:    fmt.Sprintf("%s <%s>", r.senderName, r.senderEmail),
		To:      []string{toEmail},
		Subject: fmt.Sprintf("%s is your Callit code", code),
		Html:    fmt.Sprintf(`<p>Your Callit verification code is <strong>%s</strong>. It expires in 10 minutes.</p>`, code),
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
		r.logger.Error("resend send failed", "status", res.StatusCode)
		return fmt.Errorf("resend API error: status %d", res.StatusCode)
	}
	return nil
}
