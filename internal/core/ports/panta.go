package ports

import (
	"context"

	"github.com/eric/callit-backend/internal/core/domain"
)

// PantaClient is implemented twice: internal/adapter/panta/real.go (live
// HTTP calls) and internal/adapter/panta/mock.go (in-memory fixtures,
// selected when PANTA_MOCK=true). Every method maps 1:1 to a confirmed
// endpoint from docs.panta.market — see docs/NOTES.md for which were seen
// as a literal example vs inferred from the doc slug.
type PantaClient interface {
	ListMarkets(ctx context.Context, q domain.MarketListQuery) (*domain.MarketListResponse, error)
	GetMarket(ctx context.Context, marketID string) (*domain.MarketDetail, error)
	GetMarketTrades(ctx context.Context, marketID string) (any, error)
	GetCategories(ctx context.Context) ([]domain.Category, error)
	GetWalletTrades(ctx context.Context, wallet string) (any, error)

	ImageUploadSign(ctx context.Context) (*domain.ImageUploadSignResponse, error)
	CreateQuote(ctx context.Context, req domain.CreateQuoteRequest) (*domain.CreateQuoteResponse, error)
	CreateBuild(ctx context.Context, req domain.CreateBuildRequest) (*domain.CreateBuildResponse, error)
	RegisterMarket(ctx context.Context, req domain.RegisterMarketRequest) (*domain.RegisterMarketResponse, error)

	OrderQuote(ctx context.Context, req domain.OrderQuoteRequest) (*domain.OrderQuoteResponse, error)
	OrderBuild(ctx context.Context, req domain.OrderBuildRequest) (*domain.OrderBuildResponse, error)
	OrderSubmit(ctx context.Context, req domain.OrderSubmitRequest) (*domain.OrderSubmitResponse, error)
	OrderVerify(ctx context.Context, req domain.OrderVerifyRequest) (*domain.OrderVerifyResponse, error)

	ListPositions(ctx context.Context, wallet string) (*domain.PositionsResponse, error)
	ClaimWin(ctx context.Context, req domain.ClaimRequest) (*domain.ClaimBuildResponse, error)
	ClaimCreatorFees(ctx context.Context, req domain.ClaimRequest) (*domain.ClaimCreatorFeesResponse, error)

	ReportTrade(ctx context.Context, req domain.ReportTradeRequest) (*domain.ReportTradeResponse, error)
	TradeStatus(ctx context.Context, signature, userID string) (*domain.TradeStatusResponse, error)
}
