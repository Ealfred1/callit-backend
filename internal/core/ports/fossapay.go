package ports

import (
	"context"

	"github.com/eric/callit-backend/internal/core/domain"
)

type FossapayClient interface {
	CreateCheckout(ctx context.Context, idempotencyKey string, req domain.FossapayCreateCheckoutReq) (*domain.FossapayCheckoutResponse, error)
	GetCheckout(ctx context.Context, checkoutID string) (*domain.FossapayCheckoutResponse, error)
	GetCheckoutByReference(ctx context.Context, reference string) (*domain.FossapayCheckoutResponse, error)
	VerifyWebhookSignature(body []byte, signature, secret string) bool
}
