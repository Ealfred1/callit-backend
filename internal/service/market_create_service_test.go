package service

import (
	"context"
	"testing"

	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
	"github.com/google/uuid"
)

type testPantaClient struct {
	ports.PantaClient
	lastReq domain.CreateQuoteRequest
}

func (m *testPantaClient) CreateQuote(ctx context.Context, req domain.CreateQuoteRequest) (*domain.CreateQuoteResponse, error) {
	m.lastReq = req
	return &domain.CreateQuoteResponse{
		CreateID: "test-create-id",
	}, nil
}

type testCreateSessionRepo struct {
	ports.CreateSessionRepository
	saved *domain.CreateSession
}

func (r *testCreateSessionRepo) Create(ctx context.Context, session *domain.CreateSession) error {
	r.saved = session
	return nil
}

func TestMarketCreateService_Quote_DefaultImageFallback(t *testing.T) {
	pantaMock := &testPantaClient{}
	sessionRepo := &testCreateSessionRepo{}

	svc := NewMarketCreateService(pantaMock, nil, sessionRepo, nil, nil)
	userID := uuid.New().String()

	tests := []struct {
		name        string
		category    domain.Category
		inputImage  string
		expectedImg string
	}{
		{
			name:        "Empty image with crypto category uses crypto fallback",
			category:    domain.CategoryCrypto,
			inputImage:  "",
			expectedImg: defaultCategoryImages[domain.CategoryCrypto],
		},
		{
			name:        "Empty image with sports category uses sports fallback",
			category:    domain.CategorySports,
			inputImage:  "   ",
			expectedImg: defaultCategoryImages[domain.CategorySports],
		},
		{
			name:        "Custom image is preserved",
			category:    domain.CategoryCrypto,
			inputImage:  "https://custom.image/market.png",
			expectedImg: "https://custom.image/market.png",
		},
		{
			name:        "Unknown category defaults to crypto fallback",
			category:    domain.Category("unknown"),
			inputImage:  "",
			expectedImg: defaultCategoryImages[domain.CategoryCrypto],
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := domain.CreateQuoteRequest{
				Wallet:   "sol_wallet_123",
				Category: tc.category,
				ImageURL: tc.inputImage,
			}
			res, err := svc.Quote(context.Background(), userID, req, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res == nil || res.CreateID != "test-create-id" {
				t.Fatalf("unexpected quote response: %+v", res)
			}
			if pantaMock.lastReq.ImageURL != tc.expectedImg {
				t.Errorf("expected ImageURL %q, got %q", tc.expectedImg, pantaMock.lastReq.ImageURL)
			}
		})
	}
}
