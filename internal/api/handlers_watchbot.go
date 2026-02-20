package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/RobinCoderZhao/devkit-suite/internal/watchbot"
)

type DashboardResponse struct {
	Competitors []DashboardCompetitor `json:"competitors"`
}

type DashboardCompetitor struct {
	ID                 int        `json:"id"`
	Name               string     `json:"name"`
	Domain             string     `json:"domain"`
	PagesTracked       int        `json:"pages_tracked"`
	LatestChangeTime   *time.Time `json:"latest_change_time"`
	LatestSeverity     string     `json:"latest_severity"`
	RecentAlertSnippet string     `json:"recent_alert_snippet"`
}

func (s *Server) handleDashboard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)

		// 1. Get user's competitors
		competitors, err := s.watchbotStore.ListCompetitorsByUser(r.Context(), userID)
		if err != nil {
			s.logger.Error("list competitors for dashboard", "error", err)
			respondError(w, http.StatusInternalServerError, "Database error")
			return
		}

		var dashData DashboardResponse

		// 2. Hydrate each competitor with stats
		// Note: In a real production environment with many users, this should be done
		// via a single complex SQL JOIN instead of an N+1 query loop.
		for _, comp := range competitors {
			dashComp := DashboardCompetitor{
				ID:     comp.ID,
				Name:   comp.Name,
				Domain: comp.Domain,
			}

			pages, err := s.watchbotStore.GetPagesByCompetitor(r.Context(), comp.ID)
			if err == nil {
				dashComp.PagesTracked = len(pages)

				// Fetch the latest analysis across all their pages
				if len(pages) > 0 {
					latestChange, _ := s.watchbotStore.GetLatestChange(r.Context(), pages[0].ID)
					if latestChange != nil {
						dashComp.LatestChangeTime = &latestChange.CreatedAt
						dashComp.LatestSeverity = latestChange.Severity
						if len(latestChange.Analysis) > 60 {
							dashComp.RecentAlertSnippet = latestChange.Analysis[:60] + "..."
						} else {
							dashComp.RecentAlertSnippet = latestChange.Analysis
						}
					}
				}
			}

			dashData.Competitors = append(dashData.Competitors, dashComp)
		}

		respondJSON(w, http.StatusOK, dashData)
	}
}

func (s *Server) handleListCompetitors() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)

		competitors, err := s.watchbotStore.ListCompetitorsByUser(r.Context(), userID)
		if err != nil {
			s.logger.Error("failed to list competitors", "error", err)
			respondError(w, http.StatusInternalServerError, "Database error")
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"competitors": competitors,
		})
	}
}

func (s *Server) handleCompetitorTimeline() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)

		idStr := r.PathValue("id")
		var compID int
		fmt.Sscanf(idStr, "%d", &compID)

		// Verify the user owns this competitor
		competitors, err := s.watchbotStore.ListCompetitorsByUser(r.Context(), userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Database error")
			return
		}

		var foundComp *watchbot.Competitor
		for _, c := range competitors {
			if c.ID == compID {
				foundComp = &c
				break
			}
		}

		if foundComp == nil {
			respondError(w, http.StatusForbidden, "Access denied or competitor not found")
			return
		}

		changes, err := s.watchbotStore.GetTimelineByCompetitor(r.Context(), compID)
		if err != nil {
			s.logger.Error("failed to get timeline", "error", err)
			respondError(w, http.StatusInternalServerError, "Database error")
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"competitor": foundComp,
			"timeline":   changes,
		})
	}
}

type AddCompetitorRequest struct {
	Name     string `json:"name"`
	Domain   string `json:"domain"`
	URL      string `json:"url"`
	PageType string `json:"page_type"`
}

func (s *Server) handleAddCompetitor() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)

		var req AddCompetitorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Name == "" || req.URL == "" {
			respondError(w, http.StatusBadRequest, "Name and URL are required")
			return
		}

		ctx := r.Context()
		u, err := s.userStore.GetUserByID(ctx, userID)
		if err != nil || u == nil {
			respondError(w, http.StatusUnauthorized, "User not found")
			return
		}

		// Retrieve current competitors count
		competitors, err := s.watchbotStore.ListCompetitorsByUser(ctx, userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Database error")
			return
		}

		// Gatekeeper logic
		maxCompetitors := 2
		if u.Plan == "pro" {
			maxCompetitors = 100 // effectively unlimited
		}

		if len(competitors) >= maxCompetitors {
			respondError(w, http.StatusPaymentRequired, "Subscription limit reached. Please upgrade to Pro to add more competitors.")
			return
		}

		// Add competitor and page
		compID, err := s.watchbotStore.AddCompetitor(ctx, userID, req.Name, req.Domain)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to add competitor")
			return
		}

		pageType := req.PageType
		if pageType == "" {
			pageType = "pricing"
		}

		_, err = s.watchbotStore.AddPage(ctx, compID, req.URL, pageType)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to add tracked page")
			return
		}

		respondJSON(w, http.StatusCreated, map[string]string{
			"message": "Competitor added successfully",
		})
	}
}

func (s *Server) handleGetAlertRules() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)

		rules, err := s.watchbotStore.GetUserAlertRules(r.Context(), userID)
		if err != nil {
			s.logger.Error("failed to get alert rules", "error", err)
			respondError(w, http.StatusInternalServerError, "Database error")
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"rules": rules,
		})
	}
}

type AddAlertRuleRequest struct {
	CompetitorID *int   `json:"competitor_id"`
	RuleType     string `json:"rule_type"`
	RuleValue    string `json:"rule_value"`
	Action       string `json:"action"`
}

func (s *Server) handleAddAlertRule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)

		var req AddAlertRuleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.RuleType == "" || req.RuleValue == "" || req.Action == "" {
			respondError(w, http.StatusBadRequest, "Missing required fields")
			return
		}

		id, err := s.watchbotStore.AddAlertRule(r.Context(), userID, req.CompetitorID, req.RuleType, req.RuleValue, req.Action)
		if err != nil {
			s.logger.Error("failed to add alert rule", "error", err)
			respondError(w, http.StatusInternalServerError, "Database error")
			return
		}

		respondJSON(w, http.StatusCreated, map[string]interface{}{
			"message": "Rule added",
			"rule_id": id,
		})
	}
}
