package nearintents

import (
	"context"
	"time"

	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
)

type mockClient struct{}

func NewMockClient() ports.NearIntentsClient {
	return &mockClient{}
}

func (m *mockClient) GetTokens(ctx context.Context) ([]domain.NearIntentToken, error) {
	return []domain.NearIntentToken{
		{AssetID: AssetIdNearWNear, Symbol: "wNEAR", Blockchain: "near", Decimals: 24, PriceUSD: 3.65},
		{AssetID: AssetIdSolSOL, Symbol: "SOL", Blockchain: "sol", Decimals: 9, PriceUSD: 168.50},
		{AssetID: AssetIdEthUSDC, Symbol: "USDC", Blockchain: "eth", Decimals: 6, PriceUSD: 1.00},
		{AssetID: AssetIdBaseETH, Symbol: "ETH", Blockchain: "base", Decimals: 18, PriceUSD: 2450.00},
	}, nil
}

func (m *mockClient) CreateQuote(ctx context.Context, req domain.NearIntentQuoteReq) (*domain.NearIntentQuoteRes, error) {
	return &domain.NearIntentQuoteRes{
		DepositAddress:     "6ba5ad624ff72e290dbdccf83f90d8ce44c7f4f931a3a5589ed3684027a78bd6",
		OriginAsset:        req.OriginAsset,
		DestinationAsset:   req.DestinationAsset,
		AmountIn:           req.Amount,
		AmountInFormatted:  "1.0",
		AmountInUSD:        "3.65",
		AmountOut:          "41100000",
		AmountOutFormatted: "0.0411",
		AmountOutUSD:       "3.64",
		TimeEstimateSec:    19,
		Deadline:           time.Now().Add(24 * time.Hour),
		Status:             "PENDING_DEPOSIT",
		CorrelationID:      "mock-correlation-12345",
	}, nil
}

func (m *mockClient) GetStatus(ctx context.Context, depositAddress string) (*domain.NearIntentStatusRes, error) {
	return &domain.NearIntentStatusRes{
		Status:         "PENDING_DEPOSIT",
		DepositAddress: depositAddress,
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
	}, nil
}
