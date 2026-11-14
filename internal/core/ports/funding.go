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
	ListPending(ctx context.Context, newerThan time.Time, limit int) ([]*domain.FundingTransaction, error)
	ClaimPayout(ctx context.Context, reference string) (*PayoutJob, error)
	SetPayoutSending(ctx context.Context, reference, signature string, usdc float64) error
	SetPayoutResult(ctx context.Context, reference, status, errMsg string) error
	ListPayable(ctx context.Context, limit int) ([]string, error)
	ListStuckSending(ctx context.Context, olderThan time.Duration) ([]PayoutJob, error)
	GetPayoutInfo(ctx context.Context, reference string) (*domain.PayoutInfo, error)
	UpdateStatus(ctx context.Context, reference string, status domain.FundingStatus, completedAt *time.Time) error
}

// PayoutJob is a confirmed naira payment waiting for its USDC.
type PayoutJob struct {
	Reference     string
	Wallet        string
	AmountNGN     float64
	EstimatedUSDC float64
	Signature     string
}
