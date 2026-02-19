package watchbot

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ValidateResult holds the result of URL validation.
type ValidateResult struct {
	URL   string // normalized URL
	Valid bool
	Error string
}

// ValidateURL normalizes and validates a URL.
func ValidateURL(ctx context.Context, rawURL string) ValidateResult {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ValidateResult{Error: "URL 不能为空"}
	}

	// Auto-add scheme
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}

	// Parse
	u, err := url.Parse(rawURL)
	if err != nil {
		return ValidateResult{Error: fmt.Sprintf("URL 格式错误: %v", err)}
	}

	// Protocol check
	if u.Scheme != "http" && u.Scheme != "https" {
		return ValidateResult{Error: fmt.Sprintf("不支持的协议: %s（仅支持 http/https）", u.Scheme)}
	}

	// Must have host
	if u.Host == "" {
		return ValidateResult{Error: "URL 缺少域名"}
	}

	// Remove fragment
	u.Fragment = ""

	// Normalize trailing slash (keep path, remove trailing slash on path-only)
	if u.Path == "/" {
		u.Path = ""
	}

	normalized := u.String()

	// DNS check
	host := u.Hostname()
	if _, err := net.LookupHost(host); err != nil {
		return ValidateResult{URL: normalized, Error: fmt.Sprintf("域名无法解析: %s", host)}
	}

	// HTTP check (HEAD request with timeout) — soft validation, warn but allow
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "HEAD", normalized, nil)
	if err != nil {
		// Can't even build request — still allow (DNS resolved)
		return ValidateResult{URL: normalized, Valid: true, Error: fmt.Sprintf("⚠️ 请求构建失败: %v（已添加，请确认 URL 正确）", err)}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; WatchBot/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		// Many sites block HEAD, try GET
		req.Method = "GET"
		resp, err = client.Do(req)
		if err != nil {
			// Network error but DNS resolves — allow with warning
			return ValidateResult{URL: normalized, Valid: true, Error: fmt.Sprintf("⚠️ 暂时无法访问（已添加，将在检查时重试）")}
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return ValidateResult{URL: normalized, Valid: true, Error: fmt.Sprintf("⚠️ HTTP %d（已添加，请确认 URL 正确）", resp.StatusCode)}
	}

	return ValidateResult{URL: normalized, Valid: true}
}

// IsURL checks if the input looks like a URL (not natural language).
func IsURL(input string) bool {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return true
	}
	// Check for domain-like patterns: xxx.xxx
	parts := strings.Fields(input)
	if len(parts) == 1 && strings.Contains(parts[0], ".") && !strings.Contains(parts[0], " ") {
		// Single token with a dot — likely a domain
		tld := parts[0][strings.LastIndex(parts[0], ".")+1:]
		commonTLDs := map[string]bool{
			"com": true, "org": true, "net": true, "io": true, "dev": true,
			"ai": true, "co": true, "me": true, "app": true, "xyz": true,
		}
		return commonTLDs[strings.ToLower(tld)]
	}
	return false
}

// ExtractDomain extracts the domain from a URL.
func ExtractDomain(rawURL string) string {
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Hostname()
}

// GuessPageType guesses the page type from URL path.
func GuessPageType(rawURL string) string {
	lower := strings.ToLower(rawURL)
	switch {
	case strings.Contains(lower, "/pricing"):
		return "pricing"
	case strings.Contains(lower, "/changelog"):
		return "changelog"
	case strings.Contains(lower, "/api") || strings.Contains(lower, "/docs"):
		return "api_docs"
	case strings.Contains(lower, "/blog"):
		return "blog"
	case strings.Contains(lower, "/features"):
		return "features"
	default:
		return "general"
	}
}
