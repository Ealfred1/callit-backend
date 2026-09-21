package mailer

import (
	"log/slog"

	"github.com/eric/callit-backend/internal/core/ports"
)

// Active whenever RESEND_API_KEY is unset — logs the code instead of
// sending it, so signup/verify can be exercised end to end with zero
// external keys (mirrors mollah-backend's NewMockEmailService).
type mockEmailService struct {
	logger *slog.Logger
}

func NewMockEmailService(logger *slog.Logger) ports.EmailService {
	return &mockEmailService{logger: logger}
}

func (m *mockEmailService) SendVerificationEmail(toEmail, code string) error {
	m.logger.Info("MOCK email — verification code", "to", toEmail, "code", code)
	return nil
}
