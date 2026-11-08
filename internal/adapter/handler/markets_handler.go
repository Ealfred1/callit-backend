package handler

import (
	"context"
	"sync"
	"time"

	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
	"github.com/labstack/echo/v4"
)

// MarketsHandler is a thin, public (unauthenticated) proxy over Panta's
// catalog endpoints (PRD "Market feed and detail") — the API key stays
// server-side either way.
type MarketsHandler struct {
	panta      ports.PantaClient
	titleCache sync.Map // marketID (string) -> title (string)
}

func NewMarketsHandler(panta ports.PantaClient) *MarketsHandler {
	return &MarketsHandler{panta: panta}
}

func (h *MarketsHandler) List(c echo.Context) error {
	q := domain.MarketListQuery{
		Category:  c.QueryParam("category"),
		Status:    c.QueryParam("status"),
		CreatedBy: c.QueryParam("createdBy"),
		Cursor:    c.QueryParam("cursor"),
		Limit:     c.QueryParam("limit"),
	}
	res, err := h.panta.ListMarkets(c.Request().Context(), q)
	if err != nil {
		return respond.FromError(c, err)
	}

	// Panta's catalog list often returns empty title and description, while
	// the detail endpoint contains the actual question. Enrich missing titles
	// concurrently with in-memory caching.
	var wg sync.WaitGroup
	for i := range res.Items {
		item := &res.Items[i]
		if item.Title != "" {
			h.titleCache.Store(item.MarketID, item.Title)
			continue
		}
		if cached, ok := h.titleCache.Load(item.MarketID); ok {
			item.Title = cached.(string)
			continue
		}

		wg.Add(1)
		go func(it *domain.MarketListItem) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			det, err := h.panta.GetMarket(ctx, it.MarketID)
			if err == nil && det != nil {
				title := det.Title
				if title == "" && det.Description != nil {
					title = *det.Description
				}
				if title != "" {
					it.Title = title
					h.titleCache.Store(it.MarketID, title)
				}
			}
		}(item)
	}
	wg.Wait()

	return respond.OK(c, res)
}

func (h *MarketsHandler) Get(c echo.Context) error {
	res, err := h.panta.GetMarket(c.Request().Context(), c.Param("id"))
	if err != nil {
		return respond.FromError(c, err)
	}
	if res != nil && res.Title != "" {
		h.titleCache.Store(res.MarketID, res.Title)
	}
	return respond.OK(c, res)
}

func (h *MarketsHandler) Trades(c echo.Context) error {
	res, err := h.panta.GetMarketTrades(c.Request().Context(), c.Param("id"))
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, res)
}

func (h *MarketsHandler) WalletTrades(c echo.Context) error {
	res, err := h.panta.GetWalletTrades(c.Request().Context(), c.Param("wallet"))
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, res)
}

func (h *MarketsHandler) Categories(c echo.Context) error {
	res, err := h.panta.GetCategories(c.Request().Context())
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, res)
}
