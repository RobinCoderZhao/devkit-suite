package api

import (
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
