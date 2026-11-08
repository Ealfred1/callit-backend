package ports

import (
	"context"

	"github.com/eric/callit-backend/internal/core/domain"
)

type NearIntentsClient interface {
	GetTokens(ctx context.Context) ([]domain.NearIntentToken, error)
	CreateQuote(ctx context.Context, req domain.NearIntentQuoteReq) (*domain.NearIntentQuoteRes, error)
	GetStatus(ctx context.Context, depositAddress string) (*domain.NearIntentStatusRes, error)
}
