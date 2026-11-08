package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
)

type NearIntentsService struct {
	client ports.NearIntentsClient
	users  ports.UserRepository
	logger *slog.Logger
}

func NewNearIntentsService(
	client ports.NearIntentsClient,
	users ports.UserRepository,
	logger *slog.Logger,
) *NearIntentsService {
	return &NearIntentsService{
		client: client,
		users:  users,
		logger: logger,
	}
}

func (s *NearIntentsService) ListTokens(ctx context.Context) ([]domain.NearIntentToken, error) {
	return s.client.GetTokens(ctx)
}

func (s *NearIntentsService) CreateDepositQuote(
	ctx context.Context,
	userID string,
	req domain.NearIntentQuoteReq,
) (*domain.NearIntentQuoteRes, error) {
	if req.OriginAsset == "" {
		return nil, fmt.Errorf("originAsset is required (e.g. wNEAR, SOL, ETH, USDC)")
	}
	if req.Amount == "" && req.AmountFormatted == "" {
		return nil, fmt.Errorf("amount or amountFormatted is required")
	}

	// If recipient wasn't explicitly supplied, populate with user's Callit Solana wallet
	if req.Recipient == "" && userID != "" {
		user, err := s.users.GetByID(ctx, userID)
		if err == nil && user != nil && user.WalletAddress != "" {
			req.Recipient = user.WalletAddress
		}
	}

	if req.Recipient == "" {
		return nil, fmt.Errorf("recipient address is required")
	}

	res, err := s.client.CreateQuote(ctx, req)
	if err != nil {
		s.logger.Error("NearIntents CreateQuote failed", "err", err, "recipient", req.Recipient)
		return nil, err
	}

	return res, nil
}

func (s *NearIntentsService) GetDepositStatus(
	ctx context.Context,
	depositAddress string,
) (*domain.NearIntentStatusRes, error) {
	if depositAddress == "" {
		return nil, fmt.Errorf("depositAddress is required")
	}
	return s.client.GetStatus(ctx, depositAddress)
}
