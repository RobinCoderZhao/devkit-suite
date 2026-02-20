package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// GeoLocator resolves an IP address to a country code.
type GeoLocator interface {
	Locate(ctx context.Context, ip string) (countryCode string, err error)
}

// IPAPILocator uses ip-api.com (free, no API key, 45 req/min).
type IPAPILocator struct {
	cache  sync.Map // ip → countryCode
	client *http.Client
}

// NewIPAPILocator creates a locator using ip-api.com.
func NewIPAPILocator() *IPAPILocator {
	return &IPAPILocator{
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

type ipAPIResponse struct {
	Status      string `json:"status"`
	CountryCode string `json:"countryCode"`
	Country     string `json:"country"`
}

func (l *IPAPILocator) Locate(ctx context.Context, ip string) (string, error) {
	// Check cache
	if cached, ok := l.cache.Load(ip); ok {
		return cached.(string), nil
	}

	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,countryCode,country", ip)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ip-api request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result ipAPIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse ip-api response: %w", err)
	}

	if result.Status != "success" {
		return "", fmt.Errorf("ip-api lookup failed for %s", ip)
	}

	// Cache result
	l.cache.Store(ip, result.CountryCode)
	return result.CountryCode, nil
}

// GetPublicIP fetches the machine's public IP address via api.ipify.org.
func GetPublicIP(ctx context.Context) string {
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.ipify.org", nil)
	if err != nil {
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return strings.TrimSpace(string(body))
}

// DetectLanguage determines the output language based on priority:
//  1. User-specified language (--lang flag)
//  2. IP geolocation → country → language mapping
//  3. Default: English
func DetectLanguage(ctx context.Context, userLang string, ip string, locator GeoLocator) Language {
	// Priority 1: explicit user choice
	if userLang != "" {
		langs := ParseLanguages(userLang)
		return langs[0]
	}

	// Priority 2: IP-based detection
	if ip != "" && locator != nil {
		country, err := locator.Locate(ctx, ip)
		if err == nil {
			if lang, ok := CountryToLanguage(country); ok {
				return lang
			}
		}
	}

	// Priority 3: default English
	return LangEN
}
