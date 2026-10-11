package handler

import (
	"io"
	"net/http"

	"github.com/eric/callit-backend/internal/adapter/handler/middleware"
	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/service"
	"github.com/labstack/echo/v4"
)

type FundingHandler struct {
	svc *service.FundingService
}

func NewFundingHandler(svc *service.FundingService) *FundingHandler {
	return &FundingHandler{svc: svc}
}

func (h *FundingHandler) CreateCheckout(c echo.Context) error {
	var req domain.CreateFundingRequest
	if err := c.Bind(&req); err != nil {
		return respond.Err(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request payload", nil)
	}

	userID := middleware.UserID(c)
	res, err := h.svc.CreateCheckout(c.Request().Context(), userID, req)
	if err != nil {
		return respond.Err(c, http.StatusBadRequest, "FUNDING_ERROR", err.Error(), nil)
	}

	return respond.Created(c, res)
}

func (h *FundingHandler) GetCheckoutStatus(c echo.Context) error {
	ref := c.Param("reference")
	if ref == "" {
		return respond.Err(c, http.StatusBadRequest, "INVALID_REFERENCE", "reference is required", nil)
	}

	res, err := h.svc.GetCheckoutStatus(c.Request().Context(), ref)
	if err != nil {
		return respond.Err(c, http.StatusNotFound, "TRANSACTION_NOT_FOUND", "funding transaction not found", nil)
	}

	return respond.OK(c, res)
}

func (h *FundingHandler) HandleWebhook(c echo.Context) error {
	sig := c.Request().Header.Get("x-fossapay-signature")

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusBadRequest, "failed to read body")
	}

	if err := h.svc.HandleWebhook(c.Request().Context(), body, sig); err != nil {
		return c.String(http.StatusUnauthorized, "invalid webhook signature or payload")
	}

	return c.String(http.StatusOK, "OK")
}
