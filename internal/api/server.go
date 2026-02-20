// Package api provides the REST API server for DevKit Suite.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/RobinCoderZhao/devkit-suite/internal/user"
	"github.com/RobinCoderZhao/devkit-suite/internal/watchbot"
)

// Server holds the dependencies for the API.
type Server struct {
	userStore     *user.Store
	watchbotStore *watchbot.Store
	jwtSecret     []byte
	logger        *slog.Logger
}

// NewServer creates a new API Server instance.
func NewServer(uStore *user.Store, wStore *watchbot.Store, jwtSecret string) *Server {
	return &Server{
		userStore:     uStore,
		watchbotStore: wStore,
		jwtSecret:     []byte(jwtSecret),
		logger:        slog.Default(),
	}
}

// Routes returns the configured http.Handler (ServeMux) for the API.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Auth routes (Public)
	mux.HandleFunc("POST /api/auth/register", s.handleRegister())
	mux.HandleFunc("POST /api/auth/login", s.handleLogin())

	// Protected routes (Require JWT)
	protected := s.requireAuth(mux)

	// User
	mux.Handle("GET /api/users/me", s.requireAuthHandler(http.HandlerFunc(s.handleGetMe())))
	mux.Handle("POST /api/onboarding", s.requireAuthHandler(http.HandlerFunc(s.handleOnboarding())))

	// WatchBot
	mux.Handle("GET /api/watchbot/dashboard", s.requireAuthHandler(http.HandlerFunc(s.handleDashboard())))
	mux.Handle("GET /api/watchbot/competitors", s.requireAuthHandler(http.HandlerFunc(s.handleListCompetitors())))
	mux.Handle("GET /api/watchbot/competitor/{id}", s.requireAuthHandler(http.HandlerFunc(s.handleCompetitorTimeline())))
	mux.Handle("POST /api/watchbot/competitors", s.requireAuthHandler(http.HandlerFunc(s.handleAddCompetitor())))
	mux.Handle("GET /api/watchbot/rules", s.requireAuthHandler(http.HandlerFunc(s.handleGetAlertRules())))
	mux.Handle("POST /api/watchbot/rules", s.requireAuthHandler(http.HandlerFunc(s.handleAddAlertRule())))

	// NewsBot
	mux.Handle("GET /api/newsbot/feed", s.requireAuthHandler(http.HandlerFunc(s.handleNewsFeed())))

	// Billing (Protected)
	mux.Handle("POST /api/billing/create-checkout-session", s.requireAuthHandler(http.HandlerFunc(s.handleCreateCheckoutSession())))

	// Webhooks (Public)
	mux.HandleFunc("POST /api/webhooks/stripe", s.handleStripeWebhook())

	return protected
}

// --- Helpers ---

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
