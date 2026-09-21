package handler

import (
	"github.com/eric/callit-backend/internal/adapter/handler/middleware"
	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
	"github.com/eric/callit-backend/internal/service"
	"github.com/labstack/echo/v4"
)

type MarketCreateHandler struct {
	svc   *service.MarketCreateService
	panta ports.PantaClient
}

func NewMarketCreateHandler(svc *service.MarketCreateService, panta ports.PantaClient) *MarketCreateHandler {
	return &MarketCreateHandler{svc: svc, panta: panta}
}

// AICompile is step 1 of the create flow (PRD "Creator flow" 1-2):
// paste headline -> compiled draft + computed timestamps, ready to review.
func (h *MarketCreateHandler) AICompile(c echo.Context) error {
	var req struct {
		Text string `json:"text"`
	}
	if err := c.Bind(&req); err != nil {
		return respond.Err(c, 400, "INVALID_MARKET_PARAMS", "invalid request body", nil)
	}
	result, err := h.svc.Compile(c.Request().Context(), req.Text)
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, result)
}

func (h *MarketCreateHandler) ImageUpload(c echo.Context) error {
	res, err := h.panta.ImageUploadSign(c.Request().Context())
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, res)
}

type quoteRequest struct {
	domain.CreateQuoteRequest
	CommunityID string `json:"communityId"`
}

func (h *MarketCreateHandler) Quote(c echo.Context) error {
	var req quoteRequest
	if err := c.Bind(&req); err != nil {
		return respond.Err(c, 400, "INVALID_MARKET_PARAMS", "invalid request body", nil)
	}
	res, err := h.svc.Quote(c.Request().Context(), middleware.UserID(c), req.CreateQuoteRequest, req.CommunityID)
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, res)
}

func (h *MarketCreateHandler) Build(c echo.Context) error {
	var req domain.CreateBuildRequest
	if err := c.Bind(&req); err != nil {
		return respond.Err(c, 400, "INVALID_MARKET_PARAMS", "invalid request body", nil)
	}
	res, err := h.svc.Build(c.Request().Context(), req)
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, res)
}

func (h *MarketCreateHandler) Register(c echo.Context) error {
	var req domain.RegisterMarketRequest
	if err := c.Bind(&req); err != nil {
		return respond.Err(c, 400, "INVALID_MARKET_PARAMS", "invalid request body", nil)
	}
	res, err := h.svc.Register(c.Request().Context(), middleware.UserID(c), req)
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, res)
}
