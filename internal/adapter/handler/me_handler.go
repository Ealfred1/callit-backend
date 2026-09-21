package handler

import (
	"github.com/eric/callit-backend/internal/adapter/handler/middleware"
	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/eric/callit-backend/internal/core/ports"
	"github.com/labstack/echo/v4"
)

// MeHandler backs PRD "Creator dashboard": the authenticated creator's own
// markets (local market_meta rows) plus live Panta price/volume for each.
// Fees-to-claim comes from a per-market POST /claims/creator-fees call from
// the client, using the marketIds this returns.
type MeHandler struct {
	marketMeta ports.MarketMetaRepository
	panta      ports.PantaClient
}

func NewMeHandler(marketMeta ports.MarketMetaRepository, panta ports.PantaClient) *MeHandler {
	return &MeHandler{marketMeta: marketMeta, panta: panta}
}

func (h *MeHandler) Markets(c echo.Context) error {
	ctx := c.Request().Context()
	rows, err := h.marketMeta.ListByCreator(ctx, middleware.UserID(c))
	if err != nil {
		return respond.FromError(c, err)
	}

	type item struct {
		Meta any `json:"meta"`
		Live any `json:"live"`
	}
	items := make([]item, 0, len(rows))
	for _, row := range rows {
		live, err := h.panta.GetMarket(ctx, row.MarketID)
		if err != nil {
			items = append(items, item{Meta: row, Live: nil})
			continue
		}
		items = append(items, item{Meta: row, Live: live})
	}
	return respond.OK(c, map[string]any{"items": items})
}
