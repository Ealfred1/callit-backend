package fossapay

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
	"github.com/google/uuid"
)

type mockClient struct {
	mu        sync.RWMutex
	checkouts map[string]*domain.FossapayCheckoutResponse
}

func NewMockClient() ports.FossapayClient {
	return &mockClient{
		checkouts: make(map[string]*domain.FossapayCheckoutResponse),
	}
}

func (m *mockClient) CreateCheckout(ctx context.Context, idempotencyKey string, req domain.FossapayCreateCheckoutReq) (*domain.FossapayCheckoutResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := uuid.New().String()
	exp := time.Now().Add(30 * time.Minute)
	now := time.Now()

	fee := 80.0
	amountPayable := float64(req.Amount) + fee
	if req.FeeBearer == "merchant" {
		amountPayable = float64(req.Amount)
	}

	resp := &domain.FossapayCheckoutResponse{
		Status:     "success",
		StatusCode: 201,
		Message:    "Mock checkout session created",
		Data: domain.FossapayCheckoutData{
			ID:               id,
			Reference:        req.Reference,
			Status:           "pending",
			Amount:           fmt.Sprintf("%.2f", float64(req.Amount)),
			Fee:              fmt.Sprintf("%.2f", fee),
			AmountPayable:    fmt.Sprintf("%.2f", amountPayable),
			SettlementAmount: fmt.Sprintf("%.2f", float64(req.Amount)),
			FeeBearer:        req.FeeBearer,
			Currency:         req.Currency,
			Account: domain.FossapayAccount{
				Type:          "temporary",
				BankName:      "Sterling Bank (FossaPay Demo)",
				AccountName:   "CALLIT / " + req.Reference,
				AccountNumber: "9541382701",
				ExpiresAt:     &exp,
			},
			TransactionID: uuid.New().String(),
			ExpiresAt:     &exp,
			CreatedAt:     &now,
		},
	}

	m.checkouts[req.Reference] = resp
	m.checkouts[id] = resp

	return resp, nil
}

func (m *mockClient) GetCheckout(ctx context.Context, checkoutID string) (*domain.FossapayCheckoutResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	c, ok := m.checkouts[checkoutID]
	if !ok {
		return nil, fmt.Errorf("checkout %s not found", checkoutID)
	}
	return c, nil
}

func (m *mockClient) GetCheckoutByReference(ctx context.Context, reference string) (*domain.FossapayCheckoutResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	c, ok := m.checkouts[reference]
	if !ok {
		return nil, fmt.Errorf("checkout reference %s not found", reference)
	}
	return c, nil
}

func (m *mockClient) VerifyWebhookSignature(body []byte, signature, secret string) bool {
	return true
}
