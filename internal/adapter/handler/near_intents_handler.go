package handler

import (
	"net/http"

	"github.com/eric/callit-backend/internal/adapter/handler/middleware"
	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/service"
	"github.com/labstack/echo/v4"
)

type NearIntentsHandler struct {
	svc *service.NearIntentsService
}

func NewNearIntentsHandler(svc *service.NearIntentsService) *NearIntentsHandler {
	return &NearIntentsHandler{svc: svc}
}

// Tokens returns the cross-chain tokens supported for 1-click funding via NEAR Intents.
func (h *NearIntentsHandler) Tokens(c echo.Context) error {
	tokens, err := h.svc.ListTokens(c.Request().Context())
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, tokens)
}

// Quote calculates a guaranteed cross-chain swap quote and provides the single-use
// origin chain deposit address for funding Callit balances.
func (h *NearIntentsHandler) Quote(c echo.Context) error {
	var req domain.NearIntentQuoteReq
	if err := c.Bind(&req); err != nil {
		return respond.Err(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid quote request", nil)
	}

	userID := middleware.UserID(c)
	res, err := h.svc.CreateDepositQuote(c.Request().Context(), userID, req)
	if err != nil {
		return respond.Err(c, http.StatusBadRequest, "NEAR_INTENTS_ERROR", err.Error(), nil)
	}

	return respond.OK(c, res)
}

// Status checks the execution status of a deposit address.
func (h *NearIntentsHandler) Status(c echo.Context) error {
	depositAddress := c.QueryParam("depositAddress")
	if depositAddress == "" {
		return respond.Err(c, http.StatusBadRequest, "INVALID_PARAMS", "depositAddress query parameter is required", nil)
	}

	res, err := h.svc.GetDepositStatus(c.Request().Context(), depositAddress)
	if err != nil {
		return respond.Err(c, http.StatusBadRequest, "NEAR_INTENTS_ERROR", err.Error(), nil)
	}

	return respond.OK(c, res)
}
