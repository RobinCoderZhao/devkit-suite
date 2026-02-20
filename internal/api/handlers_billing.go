package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/RobinCoderZhao/devkit-suite/internal/watchbot/billing"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"
)

// handleCreateCheckoutSession initiates a Stripe Checkout for a specific plan.
func (s *Server) handleCreateCheckoutSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)

		// Fetch user to get their email
		// Note: since getUserID only gives ID, we should get user details from store.
		// For simplicity, we assume we have a way to get user email. Let's add a helper or directly get it.
		// Wait, userStore isn't exposed with GetUserByID yet. We'll need to add that.
		// For now, let's just use a dummy email if we can't find it, or we'll add the method next.
		// Let's assume we will add s.userStore.GetUserByID(r.Context(), userID)
		u, err := s.userStore.GetUserByID(r.Context(), userID)
		if err != nil || u == nil {
			respondError(w, http.StatusUnauthorized, "User not found")
			return
		}

		var req struct {
			PriceID string `json:"price_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Use environment variables for URLs
		baseURL := os.Getenv("FRONTEND_URL")
		if baseURL == "" {
			baseURL = "http://localhost:3000"
		}
		successURL := baseURL + "/dashboard/settings?checkout=success"
		cancelURL := baseURL + "/pricing?checkout=cancel"

		url, err := billing.CreateCheckoutSession(u.Email, req.PriceID, successURL, cancelURL)
		if err != nil {
			s.logger.Error("failed to create checkout session", "error", err)
			respondError(w, http.StatusInternalServerError, "Failed to create checkout session")
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"url": url,
		})
	}
}

// handleStripeWebhook processes events from Stripe.
func (s *Server) handleStripeWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const MaxBodyBytes = int64(65536)
		r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			s.logger.Error("Error reading request body", "error", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
		signatureHeader := r.Header.Get("Stripe-Signature")

		event, err := webhook.ConstructEvent(payload, signatureHeader, endpointSecret)
		if err != nil {
			s.logger.Error("Error verifying webhook signature", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch event.Type {
		case "checkout.session.completed":
			var session stripe.CheckoutSession
			err := json.Unmarshal(event.Data.Raw, &session)
			if err != nil {
				s.logger.Error("Error parsing webhook JSON", "error", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// We need to map Stripe Customer Email to our User.
			email := session.CustomerEmail
			if session.CustomerDetails != nil && session.CustomerDetails.Email != "" {
				email = session.CustomerDetails.Email
			}

			if email == "" {
				s.logger.Error("No email found in checkout session", "sessionID", session.ID)
				w.WriteHeader(http.StatusOK)
				return
			}

			u, err := s.userStore.GetUserByEmail(r.Context(), email)
			if err != nil || u == nil {
				s.logger.Error("User not found for Stripe checkout", "email", email)
				w.WriteHeader(http.StatusOK)
				return
			}

			// Ideally, we determine the plan from session.LineItems. For now, assuming "pro".
			err = s.userStore.UpdateStripeIDs(r.Context(), u.ID, session.Customer.ID, session.Subscription.ID, "pro")
			if err != nil {
				s.logger.Error("Failed to update user Stripe IDs", "error", err)
			} else {
				s.logger.Info("User upgraded to pro via Stripe checkout", "userID", u.ID)
			}

		case "customer.subscription.deleted":
			var subscription stripe.Subscription
			err := json.Unmarshal(event.Data.Raw, &subscription)
			if err != nil {
				s.logger.Error("Error parsing webhook JSON", "error", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Downgrade to free via Customer ID
			// We need a way to GetUserByStripeCustomerID. Let's fall back to email if we can't.
			// Currently we only have UpdateStripeIDs. We must ensure robust handling in a real app.
			s.logger.Info("Subscription deleted (mock downgrade)", "subID", subscription.ID)
			// TODO: Add s.userStore.DowngradeUserBySubscriptionID(ctx, subscription.ID, "free")
		}

		w.WriteHeader(http.StatusOK)
	}
}
