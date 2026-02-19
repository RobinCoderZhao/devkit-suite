package sources

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RSSSource fetches articles from any RSS/Atom feed.
type RSSSource struct {
	name   string
	url    string
	client *http.Client
}

// NewRSSSource creates a new RSS source with a given name and feed URL.
func NewRSSSource(name, feedURL string) *RSSSource {
	return &RSSSource{
		name:   name,
		url:    feedURL,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (r *RSSSource) Name() string { return r.name }

func (r *RSSSource) Fetch(ctx context.Context) ([]Article, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", r.url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "DevkitSuite-NewsBot/1.0")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch RSS feed %s: %w", r.name, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read RSS feed: %w", err)
	}

	// Try RSS 2.0 first
	var rss rssFeed
	if err := xml.Unmarshal(body, &rss); err == nil && len(rss.Channel.Items) > 0 {
		return r.convertRSSItems(rss.Channel.Items), nil
	}

	// Try Atom
	var atom atomFeed
	if err := xml.Unmarshal(body, &atom); err == nil && len(atom.Entries) > 0 {
		return r.convertAtomEntries(atom.Entries), nil
	}

	return nil, fmt.Errorf("failed to parse feed as RSS or Atom")
}

// RSS 2.0 types
type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title string    `xml:"title"`
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Author      string `xml:"author"`
}

// Atom types
type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Title   string      `xml:"title"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title   string   `xml:"title"`
	Link    atomLink `xml:"link"`
	Summary string   `xml:"summary"`
	Updated string   `xml:"updated"`
	Author  struct {
		Name string `xml:"name"`
	} `xml:"author"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
}

func (r *RSSSource) convertRSSItems(items []rssItem) []Article {
	articles := make([]Article, 0, len(items))
	for _, item := range items {
		pubDate, _ := time.Parse(time.RFC1123Z, item.PubDate)
		if pubDate.IsZero() {
			pubDate, _ = time.Parse(time.RFC1123, item.PubDate)
		}
		articles = append(articles, Article{
			Title:       item.Title,
			URL:         item.Link,
			Source:      r.name,
			Author:      item.Author,
			Content:     item.Description,
			PublishedAt: pubDate,
			FetchedAt:   time.Now(),
			Tags:        []string{r.name},
		})
	}
	return articles
}

func (r *RSSSource) convertAtomEntries(entries []atomEntry) []Article {
	articles := make([]Article, 0, len(entries))
	for _, entry := range entries {
		updated, _ := time.Parse(time.RFC3339, entry.Updated)
		articles = append(articles, Article{
			Title:       entry.Title,
			URL:         entry.Link.Href,
			Source:      r.name,
			Author:      entry.Author.Name,
			Content:     entry.Summary,
			PublishedAt: updated,
			FetchedAt:   time.Now(),
			Tags:        []string{r.name},
		})
	}
	return articles
}
