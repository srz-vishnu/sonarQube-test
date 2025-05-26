package middleware

import (
	"context"
	"net/http"
	"sonartest_cart/pkg/api"
	"sonartest_cart/pkg/jwt"
	"strings"
)

type contextKey string

const (
	UserIDKey   contextKey = "userid"
	UsernameKey contextKey = "username"
	IsAdminKey  contextKey = "isadmin"
)

// JWTMiddleware defines the interface for middleware methods
//
//go:generate mockgen -destination=mock_jwtmiddleware.go -package=middleware . JWTMiddleware
type JWTMiddleware interface {
	JWTAuthMiddleware(next http.Handler) http.Handler
	AdminOnlyMiddleware(next http.Handler) http.Handler
}

// JWTMiddlewareImpl is the concrete implementation
// It holds a reference to a JWTService
type JWTMiddlewareImpl struct {
	jwtService jwt.JWTService
}

// NewJWTMiddleware creates a new JWTMiddleware with the given JWTService
func NewJWTMiddleware(jwtService jwt.JWTService) JWTMiddleware {
	return &JWTMiddlewareImpl{jwtService: jwtService}
}

// middleware for users routes
func (m *JWTMiddlewareImpl) JWTAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the token from the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			api.Fail(w, http.StatusUnauthorized, 401, "Authorization header is missing", "")
			return
		}

		// Extract the token part (removing 'Bearer ' prefix)
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			api.Fail(w, http.StatusUnauthorized, 401, "Invalid token format", "")
			return
		}

		// Validate the token using the interface
		claims, err := m.jwtService.ValidateToken(tokenString)
		if err != nil {
			if err == jwt.ErrExpiredToken {
				api.Fail(w, http.StatusUnauthorized, 401, "Token expired, please log in again", "")
				return
			}

			api.Fail(w, http.StatusUnauthorized, 401, "Invalid token", err.Error())
			return
		}

		// Store userid and username in context
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UsernameKey, claims.Username)
		ctx = context.WithValue(ctx, IsAdminKey, claims.IsAdmin)

		// Pass control to the next handler
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

// middleware for admin-only routes
func (m *JWTMiddlewareImpl) AdminOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAdmin, ok := r.Context().Value(IsAdminKey).(bool)
		if !ok || !isAdmin {
			api.Fail(w, http.StatusForbidden, 403, "Admin access required", "")
			return
		}
		next.ServeHTTP(w, r)
	})
}
