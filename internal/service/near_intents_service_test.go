package service

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/eric/callit-backend/internal/adapter/nearintents"
	"github.com/eric/callit-backend/internal/core/domain"
)

func TestNearIntentsService_ListTokensAndQuote(t *testing.T) {
	mockClient := nearintents.NewMockClient()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewNearIntentsService(mockClient, nil, logger)

	ctx := context.Background()

	// 1. Tokens
	tokens, err := svc.ListTokens(ctx)
	if err != nil {
		t.Fatalf("ListTokens error: %v", err)
	}
	if len(tokens) == 0 {
		t.Fatalf("expected tokens from mock, got 0")
	}

	// 2. Quote
	quote, err := svc.CreateDepositQuote(ctx, "test-user-id", domain.NearIntentQuoteReq{
		OriginAsset:      "wNEAR",
		DestinationAsset: "SOL",
		Amount:           "1000000000000000000000000",
		Recipient:        "BuviaRJWHby737pShLbyd53UXqYmtWCWAsvJ7pDEsxEa",
	})
	if err != nil {
		t.Fatalf("CreateDepositQuote error: %v", err)
	}
	if quote.DepositAddress == "" {
		t.Fatalf("expected valid depositAddress, got empty")
	}
	if quote.TimeEstimateSec <= 0 {
		t.Fatalf("expected positive time estimate")
	}

	// 3. Status
	status, err := svc.GetDepositStatus(ctx, quote.DepositAddress)
	if err != nil {
		t.Fatalf("GetDepositStatus error: %v", err)
	}
	if status.Status != "PENDING_DEPOSIT" {
		t.Fatalf("expected PENDING_DEPOSIT, got %s", status.Status)
	}
}
