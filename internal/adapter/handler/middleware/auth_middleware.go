package middleware

import (
	"strings"

	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

const UserIDContextKey = "user_id"

// JWTMiddleware validates the Bearer token and injects user_id into the
// context (mirrors mollah-backend's auth_middleware.go).
func JWTMiddleware(jwtSecret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return respond.Unauthorized("missing authorization header")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				return respond.Unauthorized("invalid authorization format")
			}

			token, err := jwt.Parse(parts[1], func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				return respond.Unauthorized("invalid or expired token")
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return respond.Unauthorized("invalid token claims")
			}
			userID, ok := claims["user_id"].(string)
			if !ok || userID == "" {
				return respond.Unauthorized("user_id missing from token")
			}

			c.Set(UserIDContextKey, userID)
			return next(c)
		}
	}
}

func UserID(c echo.Context) string {
	id, _ := c.Get(UserIDContextKey).(string)
	return id
}
