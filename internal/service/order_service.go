package service

import (
	"context"

	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
)

// OrderService is the primary-buy pipeline (PRD "Trader flow"):
// quote -> build -> submit -> verify. communityID (optional) resolves
// server-side to the community's attribution_id and is sent to Panta as the
// buy-flow userId (PRD "Attribution").
type OrderService struct {
	panta       ports.PantaClient
	communities *CommunityService
}

func NewOrderService(panta ports.PantaClient, communities *CommunityService) *OrderService {
	return &OrderService{panta: panta, communities: communities}
}

func (s *OrderService) Quote(ctx context.Context, wallet, marketID string, side domain.Side, amountUsdc, communityID string) (*domain.OrderQuoteResponse, error) {
	if wallet == "" || marketID == "" || amountUsdc == "" {
		return nil, respond.BadRequest("INVALID_MARKET_PARAMS", "wallet, marketId and amountUsdc are required")
	}
	userID := s.communities.ResolveAttributionID(ctx, communityID)
	return s.panta.OrderQuote(ctx, domain.OrderQuoteRequest{Wallet: wallet, MarketID: marketID, Side: side, AmountUsdc: amountUsdc, UserID: userID})
}

func (s *OrderService) Build(ctx context.Context, quoteID, wallet, communityID string, maxSlippageBps int) (*domain.OrderBuildResponse, error) {
	if quoteID == "" || wallet == "" {
		return nil, respond.BadRequest("INVALID_MARKET_PARAMS", "quoteId and wallet are required")
	}
	userID := s.communities.ResolveAttributionID(ctx, communityID)
	return s.panta.OrderBuild(ctx, domain.OrderBuildRequest{QuoteID: quoteID, Wallet: wallet, UserID: userID, MaxSlippageBps: maxSlippageBps})
}

func (s *OrderService) Submit(ctx context.Context, req domain.OrderSubmitRequest) (*domain.OrderSubmitResponse, error) {
	if req.OrderID == "" || req.Signature == "" {
		return nil, respond.BadRequest("INVALID_MARKET_PARAMS", "orderId and signature are required")
	}
	return s.panta.OrderSubmit(ctx, req)
}

func (s *OrderService) Verify(ctx context.Context, req domain.OrderVerifyRequest) (*domain.OrderVerifyResponse, error) {
	if req.OrderID == "" {
		return nil, respond.BadRequest("INVALID_MARKET_PARAMS", "orderId is required")
	}
	return s.panta.OrderVerify(ctx, req)
}
