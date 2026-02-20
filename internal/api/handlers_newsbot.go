package api

import (
	"net/http"
	"time"
)

type NewsItem struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Source      string    `json:"source"`
	URL         string    `json:"url"`
	Summary     string    `json:"summary"`
	PublishedAt time.Time `json:"published_at"`
}

func (s *Server) handleNewsFeed() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Mock Data for NewsBot Feed to get the UI built quickly for Phase 2 Aha Moment.
		// In future phases, this will hook into the real NewsBot multi-tenant scraper.
		feed := []NewsItem{
			{
				ID:          1,
				Title:       "OpenAI Releases GPT-4o Model with Built-in Omni Capabilities",
				Source:      "TechCrunch",
				URL:         "https://techcrunch.com",
				Summary:     "The new flagship model can reason across audio, vision, and text in real time, drastically reducing latency and expanding use cases.",
				PublishedAt: time.Now().Add(-2 * time.Hour),
			},
			{
				ID:          2,
				Title:       "Anthropic Claude 3.5 Sonnet Outperforms Rivals",
				Source:      "VentureBeat",
				URL:         "https://venturebeat.com",
				Summary:     "Anthropic's mid-tier model surprisingly beats out top-tier competitors in key coding and reasoning benchmarks with double the speed.",
				PublishedAt: time.Now().Add(-24 * time.Hour),
			},
			{
				ID:          3,
				Title:       "Google Vids leverages Gemini to automate video creation for workspaces",
				Source:      "The Verge",
				URL:         "https://theverge.com",
				Summary:     "Google's new workspace tool uses GenAI to storyboard, write scripts, and assemble stock footage into presentations autonomously.",
				PublishedAt: time.Now().Add(-48 * time.Hour),
			},
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"feed": feed,
		})
	}
}
