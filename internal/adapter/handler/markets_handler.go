package handler

import (
	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
	"github.com/labstack/echo/v4"
)

// MarketsHandler is a thin, public (unauthenticated) proxy over Panta's
// catalog endpoints (PRD "Market feed and detail") — the API key stays
// server-side either way.
type MarketsHandler struct {
	panta ports.PantaClient
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
	return respond.OK(c, res)
}

func (h *MarketsHandler) Get(c echo.Context) error {
	res, err := h.panta.GetMarket(c.Request().Context(), c.Param("id"))
	if err != nil {
		return respond.FromError(c, err)
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
