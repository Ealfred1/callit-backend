package panta

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
)

// In-memory stand-in for the live Panta API (PANTA_MOCK=true). Lets the
// whole create -> quote -> build -> register -> buy -> verify -> claim
// pipeline be exercised end to end (Flutter app included) before a real
// pk_test_ key exists. Never enable in production.
type mockClient struct {
	mu       sync.Mutex
	seq      atomic.Int64
	sessions map[string]*mockCreateSession
	quotes   map[string]*mockQuote
	orders   map[string]*mockOrder
	markets  map[string]*domain.MarketDetail
}

type mockCreateSession struct {
	req    domain.CreateQuoteRequest
	status string
}

type mockQuote struct {
	req      domain.OrderQuoteRequest
	avgPrice string
	shares   string
	feeUsdc  string
}

type mockOrder struct {
	marketID   string
	side       domain.Side
	amountUsdc string
	status     string
	signature  string
}

func NewMockClient() ports.PantaClient {
	return &mockClient{
		sessions: map[string]*mockCreateSession{},
		quotes:   map[string]*mockQuote{},
		orders:   map[string]*mockOrder{},
		markets:  map[string]*domain.MarketDetail{},
	}
}

func (c *mockClient) nextID(prefix string) string {
	n := c.seq.Add(1)
	return fmt.Sprintf("%s_mock%06d", prefix, n)
}

func notFound(code, msg string) error {
	return &domain.PantaError{HTTPStatus: 404, Code: code, Message: msg}
}

func badRequest(code, msg string) error {
	return &domain.PantaError{HTTPStatus: 400, Code: code, Message: msg}
}

func (c *mockClient) seedMarketLocked(marketID string) *domain.MarketDetail {
	if m, ok := c.markets[marketID]; ok {
		return m
	}
	now := time.Now().Unix()
	yes, no := "0.50", "0.50"
	m := &domain.MarketDetail{
		MarketListItem: domain.MarketListItem{
			MarketID: marketID, Category: domain.CategoryOther, Title: "Mock market",
			Images: []string{"https://res.cloudinary.com/demo/image/upload/mock.png"},
			Phase: "primary", MarketType: "standard",
			StartTime: now, EndTime: now + 86400, ResolutionTime: now + 90000,
			Region: "Global", VolumeUsdc: "0.00",
			YesPrice: &yes, NoPrice: &no, PrimaryYesPrice: &yes, PrimaryNoPrice: &no,
		},
		Status: "open", CreatedByPartner: true,
	}
	c.markets[marketID] = m
	return m
}

func (c *mockClient) ListMarkets(ctx context.Context, q domain.MarketListQuery) (*domain.MarketListResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	items := make([]domain.MarketListItem, 0, len(c.markets))
	for _, m := range c.markets {
		items = append(items, m.MarketListItem)
	}
	return &domain.MarketListResponse{Items: items}, nil
}

func (c *mockClient) GetMarket(ctx context.Context, marketID string) (*domain.MarketDetail, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	m := c.seedMarketLocked(marketID)
	cp := *m
	return &cp, nil
}

func (c *mockClient) GetMarketTrades(ctx context.Context, marketID string) (any, error) {
	return []any{}, nil
}

func (c *mockClient) GetCategories(ctx context.Context) ([]domain.Category, error) {
	return []domain.Category{
		domain.CategorySports, domain.CategoryCrypto, domain.CategoryPolitics, domain.CategoryEntertainment,
		domain.CategoryFinance, domain.CategoryScience, domain.CategoryWorld, domain.CategoryOther,
	}, nil
}

func (c *mockClient) GetWalletTrades(ctx context.Context, wallet string) (any, error) {
	return []any{}, nil
}

func (c *mockClient) ImageUploadSign(ctx context.Context) (*domain.ImageUploadSignResponse, error) {
	return &domain.ImageUploadSignResponse{
		UploadURL: "https://api.cloudinary.com/v1_1/mock-cloud/image/upload",
		PublicID:  c.nextID("pub"),
		ExpiresAt: time.Now().Add(5 * time.Minute).Format(time.RFC3339),
		Fields:    map[string]string{"api_key": "mock", "timestamp": fmt.Sprint(time.Now().Unix()), "signature": "mock-signature"},
	}, nil
}

func (c *mockClient) CreateQuote(ctx context.Context, req domain.CreateQuoteRequest) (*domain.CreateQuoteResponse, error) {
	if req.Question == "" || len(req.Question) > 512 {
		return nil, badRequest("INVALID_MARKET_PARAMS", "question is required and must be <= 512 chars")
	}
	if !(req.StartTime < req.EndTime && req.EndTime <= req.ResolutionTime) {
		return nil, badRequest("INVALID_MARKET_PARAMS", "startTime < endTime <= resolutionTime required")
	}

	c.mu.Lock()
	createID := c.nextID("cr")
	c.sessions[createID] = &mockCreateSession{req: req, status: "quoted"}
	c.mu.Unlock()

	marketType := req.MarketType
	if marketType == "" {
		marketType = "standard"
	}
	return &domain.CreateQuoteResponse{
		CreateID: createID, ExpectedEventPda: c.nextID("evt"),
		PaymentUsdc: "50000000", LiquidityInjectionUsdc: "10000000", PlatformRevenueUsdc: "40000000",
		MarketType: marketType, ExpiresAt: time.Now().Add(5 * time.Minute).Format(time.RFC3339), BlockhashExpiryHintSec: 60,
	}, nil
}

func (c *mockClient) CreateBuild(ctx context.Context, req domain.CreateBuildRequest) (*domain.CreateBuildResponse, error) {
	c.mu.Lock()
	session, ok := c.sessions[req.CreateID]
	if ok {
		session.status = "built"
	}
	c.mu.Unlock()
	if !ok {
		return nil, notFound("CREATE_EXPIRED", "Unknown or expired createId")
	}

	marketType := session.req.MarketType
	if marketType == "" {
		marketType = "standard"
	}
	return &domain.CreateBuildResponse{
		CreateID: req.CreateID, ExpectedEventPda: c.nextID("evt"),
		Transaction: base64.StdEncoding.EncodeToString([]byte("mock-versioned-transaction")),
		RecentBlockhash: c.nextID("bh"), LastValidBlockHeight: 123456789, BlockhashExpiryHintSec: 60,
		BuildFingerprint: c.nextID("fp"),
		PaymentUsdc: "50000000", LiquidityInjectionUsdc: "10000000", PlatformRevenueUsdc: "40000000",
		MarketType: marketType,
		Derived:    domain.CreateBuildDerived{Event: c.nextID("evt"), VaultAuthority: c.nextID("vault"), MarketConfig: c.nextID("cfg")},
		ExpiresAt:  time.Now().Add(60 * time.Second).Format(time.RFC3339),
	}, nil
}

func (c *mockClient) RegisterMarket(ctx context.Context, req domain.RegisterMarketRequest) (*domain.RegisterMarketResponse, error) {
	if req.Signature == "" {
		return nil, badRequest("TX_NOT_FOUND", "signature is required")
	}

	c.mu.Lock()
	session, ok := c.sessions[req.CreateID]
	var marketID string
	if ok {
		session.status = "registered"
		marketID = c.nextID("mkt")
		title := session.req.Title
		if title == "" {
			title = session.req.Question
		}
		yes, no := "0.50", "0.50"
		c.markets[marketID] = &domain.MarketDetail{
			MarketListItem: domain.MarketListItem{
				MarketID: marketID, Category: session.req.Category, Title: title,
				Images: []string{session.req.ImageURL}, Phase: "primary", MarketType: session.req.MarketType,
				StartTime: session.req.StartTime, EndTime: session.req.EndTime, ResolutionTime: session.req.ResolutionTime,
				Region: session.req.Region, VolumeUsdc: "0.00",
				YesPrice: &yes, NoPrice: &no, PrimaryYesPrice: &yes, PrimaryNoPrice: &no,
			},
			Status: "open", CreatedByPartner: true,
		}
	}
	c.mu.Unlock()

	if !ok {
		return nil, notFound("CREATE_EXPIRED", "Unknown or expired createId")
	}

	title := session.req.Title
	if title == "" {
		title = session.req.Question
	}
	return &domain.RegisterMarketResponse{
		CreateID: req.CreateID, MarketID: marketID, Status: "confirmed", Signature: req.Signature,
		Category: string(session.req.Category), Title: title, Images: []string{session.req.ImageURL},
	}, nil
}

func (c *mockClient) OrderQuote(ctx context.Context, req domain.OrderQuoteRequest) (*domain.OrderQuoteResponse, error) {
	amount := parseFloatSafe(req.AmountUsdc)
	if amount <= 0 {
		return nil, badRequest("AMOUNT_TOO_SMALL", "amountUsdc must be > 0")
	}

	c.mu.Lock()
	c.seedMarketLocked(req.MarketID)
	avgPrice := "0.520800"
	shares := fmt.Sprintf("%.2f", amount/parseFloatSafe(avgPrice))
	feeUsdc := fmt.Sprintf("%.2f", amount*0.02)
	quoteID := c.nextID("qt")
	c.quotes[quoteID] = &mockQuote{req: req, avgPrice: avgPrice, shares: shares, feeUsdc: feeUsdc}
	c.mu.Unlock()

	return &domain.OrderQuoteResponse{
		QuoteID: quoteID, MarketID: req.MarketID, Side: req.Side, AmountUsdc: req.AmountUsdc,
		Shares: shares, AvgPrice: avgPrice, FeeUsdc: feeUsdc,
		ExpiresAt: time.Now().Add(90 * time.Second).Format(time.RFC3339), BlockhashExpiryHintSec: 60,
	}, nil
}

func (c *mockClient) OrderBuild(ctx context.Context, req domain.OrderBuildRequest) (*domain.OrderBuildResponse, error) {
	c.mu.Lock()
	quote, ok := c.quotes[req.QuoteID]
	var orderID string
	if ok {
		orderID = c.nextID("ord")
		c.orders[orderID] = &mockOrder{marketID: quote.req.MarketID, side: quote.req.Side, amountUsdc: quote.req.AmountUsdc, status: "built"}
	}
	c.mu.Unlock()

	if !ok {
		return nil, notFound("QUOTE_STALE", "Unknown or expired quoteId")
	}

	return &domain.OrderBuildResponse{
		OrderID: orderID, QuoteID: req.QuoteID, Wallet: req.Wallet, MarketID: quote.req.MarketID, Side: quote.req.Side,
		AmountUsdc: quote.req.AmountUsdc, ExpectedShares: quote.shares, FeeUsdc: quote.feeUsdc, Status: "built",
		Instructions: []domain.OrderInstruction{{ProgramID: c.nextID("prog"), Data: base64.StdEncoding.EncodeToString([]byte("mock-ix"))}},
		Derived:      domain.OrderBuildDerived{Event: c.nextID("evt"), VaultAuthority: c.nextID("vault")},
		RecentBlockhash: c.nextID("bh"), LastValidBlockHeight: 123456789,
		ExpiresAt: time.Now().Add(120 * time.Second).Format(time.RFC3339), BlockhashExpiryHintSec: 60,
	}, nil
}

func (c *mockClient) OrderSubmit(ctx context.Context, req domain.OrderSubmitRequest) (*domain.OrderSubmitResponse, error) {
	c.mu.Lock()
	order, ok := c.orders[req.OrderID]
	if ok {
		order.status = "submitted"
		order.signature = req.Signature
	}
	c.mu.Unlock()
	if !ok {
		return nil, notFound("MARKET_NOT_FOUND", "Unknown orderId")
	}
	return &domain.OrderSubmitResponse{OrderID: req.OrderID, Status: "submitted", Signature: req.Signature}, nil
}

func (c *mockClient) OrderVerify(ctx context.Context, req domain.OrderVerifyRequest) (*domain.OrderVerifyResponse, error) {
	c.mu.Lock()
	order, ok := c.orders[req.OrderID]
	if ok && order.status == "submitted" {
		// Mock auto-confirms once submitted, so the demo path doesn't hang.
		order.status = "confirmed"
	}
	c.mu.Unlock()
	if !ok {
		return nil, notFound("MARKET_NOT_FOUND", "Unknown orderId")
	}
	var sig *string
	if order.signature != "" {
		sig = &order.signature
	}
	return &domain.OrderVerifyResponse{
		OrderID: req.OrderID, Status: order.status, Signature: sig, MarketID: order.marketID, Side: order.side,
		AmountUsdc: parseFloatSafe(order.amountUsdc),
	}, nil
}

func (c *mockClient) ListPositions(ctx context.Context, wallet string) (*domain.PositionsResponse, error) {
	return &domain.PositionsResponse{Wallet: wallet, Positions: []domain.Position{}}, nil
}

func (c *mockClient) ClaimWin(ctx context.Context, req domain.ClaimRequest) (*domain.ClaimBuildResponse, error) {
	return &domain.ClaimBuildResponse{
		Outcome: domain.SideYes, WinningShares: "10.00",
		Instructions: []domain.OrderInstruction{{ProgramID: c.nextID("prog"), Data: base64.StdEncoding.EncodeToString([]byte("mock-claim-ix"))}},
		Derived:      domain.ClaimBuildDerived{WinClaim: c.nextID("wc"), PositionPda: c.nextID("pos"), VaultAuthority: c.nextID("vault")},
		RecentBlockhash: c.nextID("bh"), LastValidBlockHeight: 123456789,
	}, nil
}

func (c *mockClient) ClaimCreatorFees(ctx context.Context, req domain.ClaimRequest) (*domain.ClaimCreatorFeesResponse, error) {
	return &domain.ClaimCreatorFeesResponse{
		ClaimableFeesUsdc: "2500000",
		Instructions:      []domain.OrderInstruction{{ProgramID: c.nextID("prog"), Data: base64.StdEncoding.EncodeToString([]byte("mock-fee-ix"))}},
		Derived:           domain.ClaimCreatorFeesDerived{CreatorFeeVault: c.nextID("vault"), MarketConfig: c.nextID("cfg")},
		RecentBlockhash:   c.nextID("bh"), LastValidBlockHeight: 123456789,
	}, nil
}

func (c *mockClient) ReportTrade(ctx context.Context, req domain.ReportTradeRequest) (*domain.ReportTradeResponse, error) {
	return &domain.ReportTradeResponse{Signature: req.Signature, Status: "processed", MarketID: req.MarketID, Wallet: req.Wallet, Side: domain.SideYes, Kind: "buy"}, nil
}

func (c *mockClient) TradeStatus(ctx context.Context, signature, userID string) (*domain.TradeStatusResponse, error) {
	return &domain.TradeStatusResponse{Signature: signature, Status: "processed"}, nil
}

func parseFloatSafe(s string) float64 {
	var f float64
	_, _ = fmt.Sscanf(s, "%f", &f)
	return f
}
