package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userContextKey = contextKey("userID")

// Claims represents the JWT payload.
type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

// generateToken creates a new JWT for a user.
func (s *Server) generateToken(userID int) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour * 7) // 7 days
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// requireAuth is a middleware that verifies the JWT token.
// Note: This wraps the entire mux or specific routes.
// For finer control, we use requireAuthHandler on specific routes.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for public routes (a bit hacky, cleaner to attach middleware per route)
		if strings.HasPrefix(r.URL.Path, "/api/auth/") {
			next.ServeHTTP(w, r)
			return
		}

		s.requireAuthHandler(next).ServeHTTP(w, r)
	})
}

// requireAuthHandler applies auth check to a specific handler.
func (s *Server) requireAuthHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Check Authorization header (Bearer <token>)
		var tokenString string
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// 2. Fallback to HttpOnly cookie
		if tokenString == "" {
			cookie, err := r.Cookie("token")
			if err == nil {
				tokenString = cookie.Value
			}
		}

		if tokenString == "" {
			respondError(w, http.StatusUnauthorized, "missing authentication token")
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return s.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			respondError(w, http.StatusUnauthorized, "invalid authentication token")
			return
		}

		// Attach UserID to context
		ctx := context.WithValue(r.Context(), userContextKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getUserID extracts the user ID from the request context.
func getUserID(r *http.Request) int {
	if val := r.Context().Value(userContextKey); val != nil {
		return val.(int)
	}
	return 0
}
