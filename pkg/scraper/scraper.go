// Package scraper provides HTTP content fetching and HTML parsing utilities.
package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// FetchOptions configures the behavior of a Fetch call.
type FetchOptions struct {
	UserAgent  string            `yaml:"user_agent"`
	Timeout    time.Duration     `yaml:"timeout"`
	RetryCount int               `yaml:"retry_count"`
	Headers    map[string]string `yaml:"headers"`
}

// DefaultFetchOptions returns sensible defaults for fetching.
func DefaultFetchOptions() *FetchOptions {
	return &FetchOptions{
		UserAgent:  "DevkitSuite/1.0 (compatible; Bot; +https://github.com/RobinCoderZhao/API-Change-Sentinel)",
		Timeout:    15 * time.Second,
		RetryCount: 2,
	}
}

// FetchResult holds the result of fetching a URL.
type FetchResult struct {
	URL        string        `json:"url"`
	StatusCode int           `json:"status_code"`
	RawHTML    string        `json:"raw_html"`
	CleanText  string        `json:"clean_text"`
	Title      string        `json:"title"`
	FetchedAt  time.Time     `json:"fetched_at"`
	Duration   time.Duration `json:"duration"`
}

// Fetcher defines the interface for fetching web content.
type Fetcher interface {
	Fetch(ctx context.Context, url string, opts *FetchOptions) (*FetchResult, error)
}

// HTTPFetcher implements Fetcher using standard HTTP.
type HTTPFetcher struct {
	client *http.Client
}

// NewHTTPFetcher creates a new HTTP-based fetcher.
func NewHTTPFetcher() *HTTPFetcher {
	return &HTTPFetcher{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Fetch retrieves a URL and extracts clean text from the HTML.
func (f *HTTPFetcher) Fetch(ctx context.Context, url string, opts *FetchOptions) (*FetchResult, error) {
	if opts == nil {
		opts = DefaultFetchOptions()
	}
	f.client.Timeout = opts.Timeout

	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", opts.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,zh-CN;q=0.8")
	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}

	var resp *http.Response
	var lastErr error
	for attempt := 0; attempt <= opts.RetryCount; attempt++ {
		resp, lastErr = f.client.Do(req)
		if lastErr == nil {
			break
		}
		if attempt < opts.RetryCount {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, lastErr)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	rawHTML := string(body)
	title := extractTitle(rawHTML)
	cleanText := ExtractText(rawHTML)

	return &FetchResult{
		URL:        url,
		StatusCode: resp.StatusCode,
		RawHTML:    rawHTML,
		CleanText:  cleanText,
		Title:      title,
		FetchedAt:  time.Now(),
		Duration:   time.Since(start),
	}, nil
}

// ExtractText converts HTML to clean structured text, removing navigation/footer/scripts.
func ExtractText(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return htmlContent
	}

	var sb strings.Builder
	extractTextFromNode(doc, &sb, map[string]bool{
		"script": true, "style": true, "nav": true, "footer": true,
		"header": true, "noscript": true, "svg": true, "iframe": true,
	})
	return strings.TrimSpace(sb.String())
}

func extractTextFromNode(n *html.Node, sb *strings.Builder, skipTags map[string]bool) {
	if n.Type == html.ElementNode {
		if skipTags[n.Data] {
			return
		}
		switch n.Data {
		case "h1":
			sb.WriteString("\n# ")
		case "h2":
			sb.WriteString("\n## ")
		case "h3":
			sb.WriteString("\n### ")
		case "h4":
			sb.WriteString("\n#### ")
		case "li":
			sb.WriteString("- ")
		case "br", "p", "div", "tr":
			sb.WriteString("\n")
		}
	}

	if n.Type == html.TextNode {
		text := strings.TrimSpace(n.Data)
		if text != "" {
			sb.WriteString(text)
			sb.WriteString(" ")
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractTextFromNode(c, sb, skipTags)
	}

	if n.Type == html.ElementNode {
		switch n.Data {
		case "h1", "h2", "h3", "h4", "p", "li", "tr":
			sb.WriteString("\n")
		}
	}
}

func extractTitle(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return ""
	}
	return findTitle(doc)
}

func findTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" {
		if n.FirstChild != nil {
			return strings.TrimSpace(n.FirstChild.Data)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if title := findTitle(c); title != "" {
			return title
		}
	}
	return ""
}
