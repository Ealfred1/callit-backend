package ports

import (
	"context"
	"time"

	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/google/uuid"
)

type FundingRepository interface {
	Create(ctx context.Context, tx *domain.FundingTransaction) error
	GetByReference(ctx context.Context, reference string) (*domain.FundingTransaction, error)
	GetByCheckoutID(ctx context.Context, checkoutID string) (*domain.FundingTransaction, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.FundingTransaction, error)
	UpdateStatus(ctx context.Context, reference string, status domain.FundingStatus, completedAt *time.Time) error
}
