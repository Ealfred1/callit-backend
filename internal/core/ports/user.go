package ports

import (
	"context"

	"github.com/eric/callit-backend/internal/core/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	SetStatus(ctx context.Context, userID string, status domain.UserStatus) error

	UpsertVerificationCode(ctx context.Context, cv *domain.VerificationCode) error
	GetVerificationCode(ctx context.Context, email string) (*domain.VerificationCode, error)
	DeleteVerificationCode(ctx context.Context, email string) error
}

type EmailService interface {
	SendVerificationEmail(toEmail, code string) error
}
