package handler

import (
	"net/http"

	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/eric/callit-backend/internal/service"
	"github.com/labstack/echo/v4"
)

type TruthSynthesisHandler struct {
	svc *service.TruthSynthesisService
}

func NewTruthSynthesisHandler(svc *service.TruthSynthesisService) *TruthSynthesisHandler {
	return &TruthSynthesisHandler{svc: svc}
}

// GetMarketTruthConsensus returns the dynamic AI-synthesized encyclopedia entry
// for a market, derived from real-time trading odds and verified resolution sources.
func (h *TruthSynthesisHandler) GetMarketTruthConsensus(c echo.Context) error {
	marketID := c.Param("id")
	if marketID == "" {
		return respond.Err(c, http.StatusBadRequest, "INVALID_PARAMS", "market id is required", nil)
	}

	consensus, err := h.svc.SynthesizeTruthConsensus(c.Request().Context(), marketID)
	if err != nil {
		return respond.FromError(c, err)
	}

	return respond.OK(c, consensus)
}
