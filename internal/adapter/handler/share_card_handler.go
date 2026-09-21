package handler

import (
	"fmt"
	"net/http"

	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/eric/callit-backend/internal/service"
	"github.com/labstack/echo/v4"
)

// ShareCardHandler serves PRD "Share card and deep link" — a public HTML
// page with OG tags, for link unfurling in WhatsApp/X/Telegram. The image
// itself is served as a static file from /static/share-cards/.
type ShareCardHandler struct {
	svc *service.ShareCardService
}

func NewShareCardHandler(svc *service.ShareCardService) *ShareCardHandler {
	return &ShareCardHandler{svc: svc}
}

// Render pre-warms the card right after a market registers (called by the
// authenticated create flow), so the very first share is instant.
func (h *ShareCardHandler) Render(c echo.Context) error {
	var req struct {
		MarketID string `json:"marketId"`
	}
	if err := c.Bind(&req); err != nil || req.MarketID == "" {
		return respond.Err(c, 400, "INVALID_MARKET_PARAMS", "marketId is required", nil)
	}
	result, err := h.svc.RenderAndStore(c.Request().Context(), req.MarketID)
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, map[string]string{"imageUrl": result.ImageURL, "deepLink": "callit://market/" + req.MarketID})
}

// Unfurl is public — link-preview bots (WhatsApp, Telegram, X) don't send
// auth headers.
func (h *ShareCardHandler) Unfurl(c echo.Context) error {
	marketID := c.Param("marketId")
	result, err := h.svc.GetOrRender(c.Request().Context(), marketID)
	if err != nil {
		return respond.FromError(c, err)
	}

	deepLink := "callit://market/" + marketID
	html := fmt.Sprintf(`<!doctype html>
<html>
<head>
<meta charset="utf-8" />
<title>%s — Callit</title>
<meta property="og:title" content="%s" />
<meta property="og:description" content="Turn any take into a market. Category: %s. Powered by Panta." />
<meta property="og:image" content="%s" />
<meta name="twitter:card" content="summary_large_image" />
<meta http-equiv="refresh" content="0; url=%s" />
</head>
<body style="font-family: sans-serif; text-align:center; padding-top: 40px;">
  <img src="%s" alt="%s" style="max-width: 90vw; border-radius: 12px;" />
  <p><a href="%s">Open in Callit</a></p>
</body>
</html>`, result.Title, result.Title, result.Category, result.ImageURL, deepLink, result.ImageURL, result.Title, deepLink)

	return c.HTML(http.StatusOK, html)
}
