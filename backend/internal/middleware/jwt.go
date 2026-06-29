package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/diyorend/budget-tracker/internal/service"
)

type contextKey string

const UserClaimsKey contextKey = "user_claims"

func JWT(authSvc *service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing token"})
			}

			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims, err := authSvc.ValidateToken(tokenStr)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			}
			
			c.Set(string(UserClaimsKey), claims)
			return next(c)
		}
	}
}

// GetClaims extracts claims from context - call this in handlers
func GetClaims(c echo.Context) *service.Claims {
	v := c.Get(string(UserClaimsKey))
	if v == nil {
		return nil
	}
	claims, _ := v.(*service.Claims)
	return claims
}

// RequireAuth is a helper used inside handlers to get userID or abort
func RequireAuth(c echo.Context) (string, error) {
	claims := GetClaims(c)
	if claims == nil {
		return "", c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	return claims.UserID, nil
}
