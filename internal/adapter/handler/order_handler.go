package handler

import (
	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/service"
	"github.com/labstack/echo/v4"
)

type OrderHandler struct {
	svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

func (h *OrderHandler) Quote(c echo.Context) error {
	var req struct {
		Wallet      string      `json:"wallet"`
		MarketID    string      `json:"marketId"`
		Side        domain.Side `json:"side"`
		AmountUsdc  string      `json:"amountUsdc"`
		CommunityID string      `json:"communityId"`
	}
	if err := c.Bind(&req); err != nil {
		return respond.Err(c, 400, "INVALID_MARKET_PARAMS", "invalid request body", nil)
	}
	res, err := h.svc.Quote(c.Request().Context(), req.Wallet, req.MarketID, req.Side, req.AmountUsdc, req.CommunityID)
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, res)
}

func (h *OrderHandler) Build(c echo.Context) error {
	var req struct {
		QuoteID        string `json:"quoteId"`
		Wallet         string `json:"wallet"`
		CommunityID    string `json:"communityId"`
		MaxSlippageBps int    `json:"maxSlippageBps"`
	}
	if err := c.Bind(&req); err != nil {
		return respond.Err(c, 400, "INVALID_MARKET_PARAMS", "invalid request body", nil)
	}
	res, err := h.svc.Build(c.Request().Context(), req.QuoteID, req.Wallet, req.CommunityID, req.MaxSlippageBps)
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, res)
}

func (h *OrderHandler) Submit(c echo.Context) error {
	var req domain.OrderSubmitRequest
	if err := c.Bind(&req); err != nil {
		return respond.Err(c, 400, "INVALID_MARKET_PARAMS", "invalid request body", nil)
	}
	res, err := h.svc.Submit(c.Request().Context(), req)
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, res)
}

func (h *OrderHandler) Verify(c echo.Context) error {
	var req domain.OrderVerifyRequest
	if err := c.Bind(&req); err != nil {
		return respond.Err(c, 400, "INVALID_MARKET_PARAMS", "invalid request body", nil)
	}
	res, err := h.svc.Verify(c.Request().Context(), req)
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, res)
}
