package handler

import (
	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
	"github.com/eric/callit-backend/internal/service"
	"github.com/labstack/echo/v4"
)

// PositionsHandler proxies GET /positions (PRD "Positions and win claim") —
// value is estimated client-side from shares x market price.
type PositionsHandler struct{ panta ports.PantaClient }

func NewPositionsHandler(panta ports.PantaClient) *PositionsHandler {
	return &PositionsHandler{panta: panta}
}

func (h *PositionsHandler) Get(c echo.Context) error {
	wallet := c.QueryParam("wallet")
	if wallet == "" {
		return respond.Err(c, 400, "INVALID_MARKET_PARAMS", "wallet query param is required", nil)
	}
	res, err := h.panta.ListPositions(c.Request().Context(), wallet)
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, res)
}

// ClaimsHandler proxies the win-claim and creator-fee-claim builders.
type ClaimsHandler struct{ panta ports.PantaClient }

func NewClaimsHandler(panta ports.PantaClient) *ClaimsHandler { return &ClaimsHandler{panta: panta} }

func (h *ClaimsHandler) Win(c echo.Context) error {
	var req domain.ClaimRequest
	if err := c.Bind(&req); err != nil {
		return respond.Err(c, 400, "INVALID_MARKET_PARAMS", "invalid request body", nil)
	}
	res, err := h.panta.ClaimWin(c.Request().Context(), req)
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, res)
}

func (h *ClaimsHandler) CreatorFees(c echo.Context) error {
	var req domain.ClaimRequest
	if err := c.Bind(&req); err != nil {
		return respond.Err(c, 400, "INVALID_MARKET_PARAMS", "invalid request body", nil)
	}
	res, err := h.panta.ClaimCreatorFees(c.Request().Context(), req)
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, res)
}

// TradesHandler reports/looks up attribution for a broadcast signature.
type TradesHandler struct {
	panta       ports.PantaClient
	communities *service.CommunityService
}

func NewTradesHandler(panta ports.PantaClient, communities *service.CommunityService) *TradesHandler {
	return &TradesHandler{panta: panta, communities: communities}
}

func (h *TradesHandler) Report(c echo.Context) error {
	var req struct {
		domain.ReportTradeRequest
		CommunityID string `json:"communityId"`
	}
	if err := c.Bind(&req); err != nil {
		return respond.Err(c, 400, "INVALID_MARKET_PARAMS", "invalid request body", nil)
	}
	req.UserID = h.communities.ResolveAttributionID(c.Request().Context(), req.CommunityID)

	res, err := h.panta.ReportTrade(c.Request().Context(), req.ReportTradeRequest)
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, res)
}

func (h *TradesHandler) Status(c echo.Context) error {
	res, err := h.panta.TradeStatus(c.Request().Context(), c.Param("signature"), c.QueryParam("userId"))
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, res)
}
