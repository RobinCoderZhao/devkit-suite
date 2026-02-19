// Package sources defines the data source interface and implementations
// for fetching AI news from various platforms.
package sources

import (
	"context"
	"time"
)

// Article represents a single news article.
type Article struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Source      string    `json:"source"`
	Author      string    `json:"author,omitempty"`
	Summary     string    `json:"summary,omitempty"`
	Content     string    `json:"content"`
	PublishedAt time.Time `json:"published_at"`
	FetchedAt   time.Time `json:"fetched_at"`
	Tags        []string  `json:"tags,omitempty"`
}

// Source is the interface that all news data sources must implement.
type Source interface {
	// Name returns the human-readable name of the source.
	Name() string

	// Fetch retrieves articles from this source.
	Fetch(ctx context.Context) ([]Article, error)
}

// Registry holds all registered data sources.
type Registry struct {
	sources []Source
}

// NewRegistry creates a new source registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a source to the registry.
func (r *Registry) Register(s Source) {
	r.sources = append(r.sources, s)
}

// FetchAll fetches articles from all registered sources concurrently.
func (r *Registry) FetchAll(ctx context.Context) ([]Article, error) {
	type result struct {
		articles []Article
		err      error
		source   string
	}

	ch := make(chan result, len(r.sources))
	for _, s := range r.sources {
		go func(src Source) {
			articles, err := src.Fetch(ctx)
			ch <- result{articles: articles, err: err, source: src.Name()}
		}(s)
	}

	var allArticles []Article
	for range r.sources {
		res := <-ch
		if res.err != nil {
			// Log but don't fail — partial results are acceptable
			continue
		}
		allArticles = append(allArticles, res.articles...)
	}

	return allArticles, nil
}
