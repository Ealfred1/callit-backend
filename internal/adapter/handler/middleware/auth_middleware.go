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
			// NOTE: every branch below goes through respond.FromError, which
			// actually writes the JSON envelope — returning a bare *AppError
			// here instead (as this used to) would bubble up to Echo's
			// default error handler, which doesn't know our envelope shape
			// and turns it into a generic 500. Caught via manual testing;
			// scripts/smoke.sh now has an "unauthorized" case so this can't
			// regress silently again.
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return respond.FromError(c, respond.Unauthorized("missing authorization header"))
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				return respond.FromError(c, respond.Unauthorized("invalid authorization format"))
			}

			token, err := jwt.Parse(parts[1], func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				return respond.FromError(c, respond.Unauthorized("invalid or expired token"))
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return respond.FromError(c, respond.Unauthorized("invalid token claims"))
			}
			userID, ok := claims["user_id"].(string)
			if !ok || userID == "" {
				return respond.FromError(c, respond.Unauthorized("user_id missing from token"))
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
