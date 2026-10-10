// Package panta implements ports.PantaClient against the live API
// (real.go) and against in-memory fixtures (mock.go, PANTA_MOCK=true).
// Endpoint paths/fields are transcribed from https://docs.panta.market
// (fetched 2026-09-21) — see docs/NOTES.md for which were confirmed via a
// literal curl example vs inferred from the doc's slug/pattern.
package panta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
)

type realClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewRealClient(baseURL, apiKey string) ports.PantaClient {
	return &realClient{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, http: http.DefaultClient}
}

type callOpts struct {
	query  url.Values
	userID string
}

// Every Panta path takes a trailing slash — confirmed repeatedly across
// docs.panta.market ("the slash comes before the query"). Centralized here
// so no call site can forget it.
func (c *realClient) call(ctx context.Context, method, path string, body any, out any, opts *callOpts) error {
	p := "/" + strings.Trim(path, "/") + "/"
	u := c.baseURL + p
	if opts != nil && len(opts.query) > 0 {
		u += "?" + opts.query.Encode()
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	if opts != nil && opts.userID != "" {
		req.Header.Set("X-User-Id", opts.userID)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("panta request failed: %w", err)
	}
	defer res.Body.Close()

	respBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode >= 300 {
		var pe domain.PantaError
		if len(respBytes) > 0 {
			_ = json.Unmarshal(respBytes, &pe)
		}
		pe.HTTPStatus = res.StatusCode
		if pe.Code == "" {
			pe.Code = "PANTA_ERROR"
		}
		if pe.Message == "" {
			pe.Message = res.Status
		}
		return &pe
	}

	if out != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, out); err != nil {
			return fmt.Errorf("decode panta response: %w", err)
		}
	}
	return nil
}

func (c *realClient) ListMarkets(ctx context.Context, q domain.MarketListQuery) (*domain.MarketListResponse, error) {
	query := url.Values{}
	if q.Category != "" {
		query.Set("category", q.Category)
	}
	if q.Status != "" {
		query.Set("status", q.Status)
	}
	if q.CreatedBy != "" {
		query.Set("createdBy", q.CreatedBy)
	}
	if q.Cursor != "" {
		query.Set("cursor", q.Cursor)
	}
	if q.Limit != "" {
		query.Set("limit", q.Limit)
	}
	var out domain.MarketListResponse
	if err := c.call(ctx, http.MethodGet, "markets", nil, &out, &callOpts{query: query}); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *realClient) GetMarket(ctx context.Context, marketID string) (*domain.MarketDetail, error) {
	var out domain.MarketDetail
	if err := c.call(ctx, http.MethodGet, "markets/"+marketID, nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *realClient) GetMarketTrades(ctx context.Context, marketID string) (any, error) {
	var out any
	if err := c.call(ctx, http.MethodGet, "markets/"+marketID+"/trades", nil, &out, nil); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *realClient) GetCategories(ctx context.Context) ([]domain.Category, error) {
	var resp struct {
		Categories []domain.Category `json:"categories"`
	}
	if err := c.call(ctx, http.MethodGet, "categories", nil, &resp, nil); err != nil {
		return nil, err
	}
	return resp.Categories, nil
}

func (c *realClient) GetWalletTrades(ctx context.Context, wallet string) (any, error) {
	var out any
	if err := c.call(ctx, http.MethodGet, "wallets/"+wallet+"/trades", nil, &out, nil); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *realClient) ImageUploadSign(ctx context.Context) (*domain.ImageUploadSignResponse, error) {
	var out domain.ImageUploadSignResponse
	if err := c.call(ctx, http.MethodPost, "markets/create/image-upload", struct{}{}, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *realClient) CreateQuote(ctx context.Context, req domain.CreateQuoteRequest) (*domain.CreateQuoteResponse, error) {
	var out domain.CreateQuoteResponse
	if err := c.call(ctx, http.MethodPost, "markets/create/quote", req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *realClient) CreateBuild(ctx context.Context, req domain.CreateBuildRequest) (*domain.CreateBuildResponse, error) {
	var out domain.CreateBuildResponse
	if err := c.call(ctx, http.MethodPost, "markets/create/build", req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *realClient) RegisterMarket(ctx context.Context, req domain.RegisterMarketRequest) (*domain.RegisterMarketResponse, error) {
	var out domain.RegisterMarketResponse
	// NOTE: docs.panta.market/api-reference/markets/register.md shows the
	// path as /markets/register/, not /markets/create/register/ despite
	// living alongside the other create/* endpoints — confirmed via that
	// page's own request example.
	if err := c.call(ctx, http.MethodPost, "markets/register", req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *realClient) OrderQuote(ctx context.Context, req domain.OrderQuoteRequest) (*domain.OrderQuoteResponse, error) {
	var out domain.OrderQuoteResponse
	if err := c.call(ctx, http.MethodPost, "primaryorderquote", req, &out, &callOpts{userID: req.UserID}); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *realClient) OrderBuild(ctx context.Context, req domain.OrderBuildRequest) (*domain.OrderBuildResponse, error) {
	var out domain.OrderBuildResponse
	if err := c.call(ctx, http.MethodPost, "primaryorderbuild", req, &out, &callOpts{userID: req.UserID}); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *realClient) OrderSubmit(ctx context.Context, req domain.OrderSubmitRequest) (*domain.OrderSubmitResponse, error) {
	var out domain.OrderSubmitResponse
	if err := c.call(ctx, http.MethodPost, "primaryordersubmit", req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *realClient) OrderVerify(ctx context.Context, req domain.OrderVerifyRequest) (*domain.OrderVerifyResponse, error) {
	var out domain.OrderVerifyResponse
	if err := c.call(ctx, http.MethodPost, "primaryorderverify", req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *realClient) ListPositions(ctx context.Context, wallet string) (*domain.PositionsResponse, error) {
	var out domain.PositionsResponse
	query := url.Values{"wallet": []string{wallet}}
	if err := c.call(ctx, http.MethodGet, "positions", nil, &out, &callOpts{query: query}); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *realClient) ClaimWin(ctx context.Context, req domain.ClaimRequest) (*domain.ClaimBuildResponse, error) {
	var out domain.ClaimBuildResponse
	// Path inferred from the doc slug /api-reference/claims/build (no
	// literal curl example shown) — verify on first live call.
	if err := c.call(ctx, http.MethodPost, "claims/build", req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *realClient) ClaimCreatorFees(ctx context.Context, req domain.ClaimRequest) (*domain.ClaimCreatorFeesResponse, error) {
	var out domain.ClaimCreatorFeesResponse
	// Path inferred from the doc slug /api-reference/claims/creator-fees —
	// same caveat as ClaimWin above.
	if err := c.call(ctx, http.MethodPost, "claims/creator-fees", req, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *realClient) ReportTrade(ctx context.Context, req domain.ReportTradeRequest) (*domain.ReportTradeResponse, error) {
	var out domain.ReportTradeResponse
	if err := c.call(ctx, http.MethodPost, "trades/report", req, &out, &callOpts{userID: req.UserID}); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *realClient) TradeStatus(ctx context.Context, signature, userID string) (*domain.TradeStatusResponse, error) {
	var out domain.TradeStatusResponse
	query := url.Values{}
	if userID != "" {
		query.Set("userId", userID)
	}
	if err := c.call(ctx, http.MethodGet, "trades/"+signature, nil, &out, &callOpts{query: query}); err != nil {
		return nil, err
	}
	return &out, nil
}
