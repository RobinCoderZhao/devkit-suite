package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HackerNewsSource fetches top stories from Hacker News API.
type HackerNewsSource struct {
	client   *http.Client
	maxItems int
}

// NewHackerNewsSource creates a new HN source.
func NewHackerNewsSource(maxItems int) *HackerNewsSource {
	if maxItems <= 0 {
		maxItems = 30
	}
	return &HackerNewsSource{
		client:   &http.Client{Timeout: 15 * time.Second},
		maxItems: maxItems,
	}
}

func (h *HackerNewsSource) Name() string { return "Hacker News" }

func (h *HackerNewsSource) Fetch(ctx context.Context) ([]Article, error) {
	// Get top story IDs
	ids, err := h.fetchTopStoryIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch top stories: %w", err)
	}

	if len(ids) > h.maxItems {
		ids = ids[:h.maxItems]
	}

	// Fetch each story (using a limited concurrency pool)
	type storyResult struct {
		article Article
		err     error
	}

	sem := make(chan struct{}, 5) // 5 concurrent fetches
	results := make(chan storyResult, len(ids))

	for _, id := range ids {
		go func(storyID int) {
			sem <- struct{}{}
			defer func() { <-sem }()

			article, err := h.fetchStory(ctx, storyID)
			results <- storyResult{article: article, err: err}
		}(id)
	}

	var articles []Article
	for range ids {
		res := <-results
		if res.err != nil {
			continue
		}
		articles = append(articles, res.article)
	}

	return articles, nil
}

func (h *HackerNewsSource) fetchTopStoryIDs(ctx context.Context) ([]int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://hacker-news.firebaseio.com/v0/topstories.json", nil)
	if err != nil {
		return nil, err
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var ids []int
	if err := json.Unmarshal(body, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

type hnStory struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
	By    string `json:"by"`
	Time  int64  `json:"time"`
	Text  string `json:"text"`
	Type  string `json:"type"`
	Score int    `json:"score"`
}

func (h *HackerNewsSource) fetchStory(ctx context.Context, id int) (Article, error) {
	url := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return Article{}, err
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return Article{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Article{}, err
	}

	var story hnStory
	if err := json.Unmarshal(body, &story); err != nil {
		return Article{}, err
	}

	return Article{
		Title:       story.Title,
		URL:         story.URL,
		Source:      "hackernews",
		Author:      story.By,
		Content:     story.Text,
		PublishedAt: time.Unix(story.Time, 0),
		FetchedAt:   time.Now(),
		Tags:        []string{"hackernews"},
	}, nil
}
