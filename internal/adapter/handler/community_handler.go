package handler

import (
	"github.com/eric/callit-backend/internal/adapter/handler/middleware"
	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/eric/callit-backend/internal/service"
	"github.com/labstack/echo/v4"
)

type CommunityHandler struct {
	svc *service.CommunityService
}

func NewCommunityHandler(svc *service.CommunityService) *CommunityHandler {
	return &CommunityHandler{svc: svc}
}

func (h *CommunityHandler) Create(c echo.Context) error {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.Bind(&req); err != nil {
		return respond.Err(c, 400, "INVALID_MARKET_PARAMS", "invalid request body", nil)
	}
	community, err := h.svc.Create(c.Request().Context(), middleware.UserID(c), req.Name)
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.Created(c, community)
}

func (h *CommunityHandler) List(c echo.Context) error {
	items, err := h.svc.List(c.Request().Context())
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, map[string]any{"items": items})
}

func (h *CommunityHandler) Get(c echo.Context) error {
	community, markets, err := h.svc.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, map[string]any{"community": community, "markets": markets})
}
