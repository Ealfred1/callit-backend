package handler

import (
	"time"

	"github.com/eric/callit-backend/internal/adapter/handler/middleware"
	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/eric/callit-backend/internal/service"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	svc       *service.AuthService
	jwtSecret string
}

func NewAuthHandler(svc *service.AuthService, jwtSecret string) *AuthHandler {
	return &AuthHandler{svc: svc, jwtSecret: jwtSecret}
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return respond.Err(c, 400, "INVALID_MARKET_PARAMS", "invalid request body", nil)
	}
	if req.Email == "" || len(req.Password) < 8 || req.Name == "" {
		return respond.Err(c, 400, "INVALID_MARKET_PARAMS", "name, email and a password of at least 8 characters are required", nil)
	}

	if err := h.svc.Register(c.Request().Context(), req.Email, req.Password, req.Name); err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, map[string]string{"message": "Verification code sent"})
}

type verifyRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func (h *AuthHandler) Verify(c echo.Context) error {
	var req verifyRequest
	if err := c.Bind(&req); err != nil {
		return respond.Err(c, 400, "INVALID_MARKET_PARAMS", "invalid request body", nil)
	}

	user, err := h.svc.Verify(c.Request().Context(), req.Email, req.Code)
	if err != nil {
		return respond.FromError(c, err)
	}

	token, err := h.generateToken(user.ID.String())
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, map[string]any{"user": user, "token": token})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return respond.Err(c, 400, "INVALID_MARKET_PARAMS", "invalid request body", nil)
	}

	user, err := h.svc.Login(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return respond.FromError(c, err)
	}

	token, err := h.generateToken(user.ID.String())
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, map[string]any{"user": user, "token": token})
}

func (h *AuthHandler) Me(c echo.Context) error {
	user, err := h.svc.GetByID(c.Request().Context(), middleware.UserID(c))
	if err != nil {
		return respond.FromError(c, err)
	}
	return respond.OK(c, user)
}

func (h *AuthHandler) generateToken(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(72 * time.Hour).Unix(),
	})
	return token.SignedString([]byte(h.jwtSecret))
}
