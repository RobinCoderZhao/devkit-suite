package api

import (
	"encoding/json"
	"fmt"
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
		userID := getUserID(r)
		var isSubscribed bool
		var subLangs string

		subs, err := s.newsbotStore.GetUserSubscribers(r.Context(), userID)
		if err == nil && len(subs) > 0 {
			isSubscribed = true
			subLangs = subs[0].Languages
		} else if subs == nil {
			// We cannot initialize with newsbot_store.Subscriber here because newsbot_store is not imported in this file directly
			// It's fine to let subs remain nil, json.Marshal will marshal it to null, but we can also just use an empty slice of any.
		}

		var subsResponse any = subs
		if subs == nil {
			subsResponse = []int{}
		}

		// Mock Data for NewsBot Feed to get the UI built quickly for Phase 2 Aha Moment.
		// In future phases, this will hook into the real NewsBot multi-tenant scraper.
		now := time.Now()
		todayLocal := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		lang := r.URL.Query().Get("lang")
		var feed []NewsItem
		if lang == "zh" || lang == "zh-CN" || lang == "zh-TW" || lang == "zh-HK" {
			feed = []NewsItem{
				{
					ID:          1,
					Title:       "OpenAI 发布内置全能模式的 GPT-4o 模型",
					Source:      "TechCrunch",
					URL:         "https://techcrunch.com",
					Summary:     "这款全新旗舰模型能实时进行跨音频、视觉和文本的推理，大幅降低延迟并极大地拓宽了使用场景。",
					PublishedAt: now.Add(-2 * time.Hour),
				},
				{
					ID:          2,
					Title:       "马斯克 xAI 开源 Grok-1.5 视觉语言模型权重",
					Source:      "GitHub",
					URL:         "https://github.com",
					Summary:     "Grok-1.5 的多模态能力迎来了巨大的提升，xAI 选择将其视觉模块一同开源，推动社区协同创新。",
					PublishedAt: now.Add(-5 * time.Hour),
				},
				{
					ID:          3,
					Title:       "苹果发布针对端端推理优化的 Apple Intelligence 本地模型",
					Source:      "Apple Newsroom",
					URL:         "https://apple.com",
					Summary:     "为了保护用户隐私，Apple 推出了可直接在手机神经网络引擎上运行的微型语言模型，体验流畅且无需联网。",
					PublishedAt: now.Add(-10 * time.Hour),
				},
				{
					ID:          4,
					Title:       "Anthropic Claude 3.5 Sonnet 超越众多竞争对手",
					Source:      "VentureBeat",
					URL:         "https://venturebeat.com",
					Summary:     "Anthropic 的中端模型以两倍的速度，在关键的代码和推理基准测试中令人惊叹地击败了顶级的竞争模型。",
					PublishedAt: todayLocal.Add(-4 * time.Hour), // Yesterday
				},
				{
					ID:          5,
					Title:       "Google Vids 借助 Gemini 实现工作空间的自动化视频创作",
					Source:      "The Verge",
					URL:         "https://theverge.com",
					Summary:     "谷歌的新款工作空间工具利用生成式AI（GenAI）自主进行故事板设计、脚本撰写，并将素材组装成精美的演示视频。",
					PublishedAt: todayLocal.Add(-8 * time.Hour), // Yesterday
				},
			}
		} else {
			feed = []NewsItem{
				{
					ID:          1,
					Title:       "OpenAI Releases GPT-4o Model with Built-in Omni Capabilities",
					Source:      "TechCrunch",
					URL:         "https://techcrunch.com",
					Summary:     "The new flagship model can reason across audio, vision, and text in real time, drastically reducing latency and expanding use cases.",
					PublishedAt: now.Add(-2 * time.Hour),
				},
				{
					ID:          2,
					Title:       "Elon Musk's xAI Open Sources Grok-1.5 Vision Model Weights",
					Source:      "GitHub",
					URL:         "https://github.com",
					Summary:     "Grok-1.5 brings massive improvements to multimodal capabilities. xAI chose to open source its vision module alongside to foster community innovation.",
					PublishedAt: now.Add(-5 * time.Hour),
				},
				{
					ID:          3,
					Title:       "Apple Launches On-Device AI Models for Apple Intelligence",
					Source:      "Apple Newsroom",
					URL:         "https://apple.com",
					Summary:     "Prioritizing user privacy, Apple has released an on-device language model running natively on the neural engine with no internet required.",
					PublishedAt: now.Add(-10 * time.Hour),
				},
				{
					ID:          4,
					Title:       "Anthropic Claude 3.5 Sonnet Outperforms Rivals",
					Source:      "VentureBeat",
					URL:         "https://venturebeat.com",
					Summary:     "Anthropic's mid-tier model surprisingly beats out top-tier competitors in key coding and reasoning benchmarks with double the speed.",
					PublishedAt: todayLocal.Add(-4 * time.Hour), // Yesterday
				},
				{
					ID:          5,
					Title:       "Google Vids leverages Gemini to automate video creation for workspaces",
					Source:      "The Verge",
					URL:         "https://theverge.com",
					Summary:     "Google's new workspace tool uses GenAI to storyboard, write scripts, and assemble stock footage into presentations autonomously.",
					PublishedAt: todayLocal.Add(-8 * time.Hour), // Yesterday
				},
			}
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"feed":               feed,
			"is_subscribed":      isSubscribed,
			"subscription_langs": subLangs,
			"subscriptions":      subsResponse,
		})
	}
}

func (s *Server) handleNewsBotSubscribe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TargetType string `json:"target_type"`
			TargetID   string `json:"target_id"`
			Languages  string `json:"languages"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.TargetType == "" {
			req.TargetType = "email"
		}
		if req.TargetID == "" {
			respondError(w, http.StatusBadRequest, "Target ID cannot be empty")
			return
		}

		userID := getUserID(r)

		// Implement IP geolocation fallback for languages if missing
		if req.Languages == "" {
			clientIP := r.Header.Get("X-Forwarded-For")
			if clientIP == "" {
				clientIP = r.RemoteAddr
			}
			// TODO: Add real GeoIP resolution lookup mapping IP to Language Code.
			// Currently defaulting to english fallback as per architecture specs.
			req.Languages = "en"
		}

		// Save subscriber into DB
		if err := s.newsbotStore.AddSubscriber(r.Context(), userID, req.TargetType, req.TargetID, req.Languages); err != nil {
			s.logger.Error("Failed to add NewsBot subscriber", "error", err)
			respondError(w, http.StatusInternalServerError, "Failed to subscribe")
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{"message": "Subscribed successfully"})
	}
}

func (s *Server) handleListNewsBotSubscriptions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		subs, err := s.newsbotStore.GetUserSubscribers(r.Context(), userID)
		if err != nil {
			s.logger.Error("Failed to list NewsBot subscriptions", "error", err, "userID", userID)
			respondError(w, http.StatusInternalServerError, "Failed to load subscriptions")
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"subscriptions": subs,
		})
	}
}

func (s *Server) handleDeleteNewsBotSubscription() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		subIDStr := r.URL.Query().Get("id")
		if subIDStr == "" {
			respondError(w, http.StatusBadRequest, "Missing subscription ID")
			return
		}

		var subID int
		if _, err := fmt.Sscanf(subIDStr, "%d", &subID); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid subscription ID")
			return
		}

		if err := s.newsbotStore.RemoveSubscriber(r.Context(), subID, userID); err != nil {
			s.logger.Error("Failed to delete NewsBot subscription", "error", err, "subID", subID, "userID", userID)
			respondError(w, http.StatusInternalServerError, "Failed to delete subscription")
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{"message": "Subscription deleted successfully"})
	}
}
