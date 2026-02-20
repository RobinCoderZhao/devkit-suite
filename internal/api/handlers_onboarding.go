package api

import (
	"encoding/json"
	"net/http"
)

type OnboardingRequest struct {
	Industry string `json:"industry"`
}

// handleOnboarding automatically provisions the user's dashboard with
// industry-specific mock competitors to provide an immediate "Aha Moment".
func (s *Server) handleOnboarding() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)

		var req OnboardingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		ctx := r.Context()

		// Provision templates based on industry
		var templates []struct{ Name, Domain, URL, PageType string }

		switch req.Industry {
		case "devtools":
			templates = []struct{ Name, Domain, URL, PageType string }{
				{"Vercel", "vercel.com", "https://vercel.com/pricing", "pricing"},
				{"Supabase", "supabase.com", "https://supabase.com/pricing", "pricing"},
				{"Cursor", "cursor.com", "https://cursor.com/features", "features"},
			}
		case "llm":
			templates = []struct{ Name, Domain, URL, PageType string }{
				{"OpenAI", "openai.com", "https://openai.com/api/pricing", "pricing"},
				{"Anthropic", "anthropic.com", "https://www.anthropic.com/pricing", "pricing"},
				{"Google Gemini", "deepmind.google", "https://aistudio.google.com/pricing", "pricing"},
			}
		default:
			// Generic SaaS defaults
			templates = []struct{ Name, Domain, URL, PageType string }{
				{"Stripe", "stripe.com", "https://stripe.com/pricing", "pricing"},
				{"Notion", "notion.so", "https://www.notion.so/pricing", "pricing"},
				{"Linear", "linear.app", "https://linear.app/pricing", "pricing"},
			}
		}

		// Insert the competitors and pages
		for _, t := range templates {
			compID, err := s.watchbotStore.AddCompetitor(ctx, userID, t.Name, t.Domain)
			if err == nil {
				_, _ = s.watchbotStore.AddPage(ctx, compID, t.URL, t.PageType)
			}
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"message":           "Onboarding complete. Competitors provisioned!",
			"provisioned_count": len(templates),
		})
	}
}
