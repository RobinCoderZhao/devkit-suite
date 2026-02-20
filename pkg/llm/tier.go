package llm

import (
	"fmt"
	"os"
	"time"
)

// ModelTier represents the quality/cost tier of a model.
//
// Usage in .env:
//
//	LLM_PROVIDER=gemini
//	LLM_API_KEY=AIzaSy...
//	LLM_MODEL=gemini-flash-latest          # fast tier (default)
//	LLM_MODEL_PRO=gemini-pro-latest        # pro tier (optional)
//
// In code:
//
//	fastClient := llm.NewTieredClient(llm.TierFast)   // daily digest, translation
//	proClient  := llm.NewTieredClient(llm.TierPro)    // content writing, deep analysis
type ModelTier string

const (
	// TierFast uses the default model — cheap, fast, good enough for daily tasks.
	// Env: LLM_MODEL (e.g. gemini-flash-latest, MiniMax-M2.5)
	TierFast ModelTier = "fast"

	// TierPro uses the premium model — higher quality for content creation.
	// Env: LLM_MODEL_PRO (e.g. gemini-pro-latest)
	// Falls back to LLM_MODEL if LLM_MODEL_PRO is not set.
	TierPro ModelTier = "pro"
)

// TierConfig returns a Config for the specified model tier, reading from env vars.
// Env vars:
//
//	LLM_PROVIDER    — provider name (gemini, minimax, openai, etc.)
//	LLM_API_KEY     — API key
//	LLM_MODEL       — fast tier model name (default)
//	LLM_MODEL_PRO   — pro tier model name (falls back to LLM_MODEL)
func TierConfig(tier ModelTier) Config {
	provider := Provider(getEnvDefault("LLM_PROVIDER", "gemini"))
	apiKey := os.Getenv("LLM_API_KEY")

	model := getEnvDefault("LLM_MODEL", "gemini-flash-latest")
	if tier == TierPro {
		proModel := os.Getenv("LLM_MODEL_PRO")
		if proModel != "" {
			model = proModel
		}
		// If no pro model set, fall back to fast model
	}

	cfg := Config{
		Provider:    provider,
		Model:       model,
		APIKey:      apiKey,
		MaxRetries:  5,
		Timeout:     90 * time.Second,
		Temperature: 0.3,
	}

	// Provider-specific defaults
	if provider == MiniMax {
		cfg.BaseURL = "https://api.minimax.io/v1"
	}

	return cfg
}

// NewTieredClient creates a Client for the specified tier from env vars.
// Returns nil and an error message if LLM_API_KEY is not set.
func NewTieredClient(tier ModelTier) (Client, error) {
	cfg := TierConfig(tier)
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("LLM_API_KEY not set")
	}
	return NewClient(cfg)
}

func getEnvDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
