package middleware

import (
	"net/http"
	"strings"

	"github.com/diyorend/budget-tracker/internal/service"
	"github.com/labstack/echo/v4"
)

type contextKey string

const UserClaimsKey contextKey = "user_claims"

// JWT validates the token from either the Authorization header (normal REST
// calls) or the "token" query param (WebSocket upgrade requests — the
// browser's WebSocket API cannot set custom headers, so the frontend has no
// other way to authenticate the upgrade request).
func JWT(authSvc *service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenStr := extractToken(c)
			if tokenStr == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing token"})
			}

			claims, err := authSvc.ValidateToken(tokenStr)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			}

			c.Set(string(UserClaimsKey), claims)
			return next(c)
		}
	}
}

func extractToken(c echo.Context) string {
	header := c.Request().Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	// Fallback for WebSocket upgrade requests: /ws?token=<jwt>
	return c.QueryParam("token")
}

// GetClaims extracts claims from context — call this in handlers.
func GetClaims(c echo.Context) *service.Claims {
	v := c.Get(string(UserClaimsKey))
	if v == nil {
		return nil
	}
	claims, _ := v.(*service.Claims)
	return claims
}

// RequireAuth is a helper used inside handlers to get userID or abort.
func RequireAuth(c echo.Context) (string, error) {
	claims := GetClaims(c)
	if claims == nil {
		return "", c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	return claims.UserID, nil
}
