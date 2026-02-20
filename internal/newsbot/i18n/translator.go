// Package i18n provides multi-language translation for NewsBot digests.
package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/RobinCoderZhao/devkit-suite/internal/newsbot/analyzer"
	"github.com/RobinCoderZhao/devkit-suite/pkg/llm"
)

// Translator translates DailyDigest content to other languages using LLM.
type Translator struct {
	client llm.Client
}

// NewTranslator creates a new Translator with the given LLM client.
func NewTranslator(client llm.Client) *Translator {
	return &Translator{client: client}
}

// translatePayload is the JSON structure sent to/from the LLM for translation.
type translatePayload struct {
	Headlines []translateHeadline `json:"headlines"`
	Summary   string              `json:"summary"`
}

type translateHeadline struct {
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Tags    []string `json:"tags"`
}

// Translate translates a DailyDigest to the target language.
// Returns a new DailyDigest with translated text fields; structural fields remain unchanged.
func (t *Translator) Translate(ctx context.Context, digest *analyzer.DailyDigest, targetLang Language) (*analyzer.DailyDigest, error) {
	if targetLang == LangZH {
		// Source language, no translation needed
		return digest, nil
	}

	// Build translation payload (only text fields)
	payload := translatePayload{
		Summary: digest.Summary,
	}
	for _, h := range digest.Headlines {
		payload.Headlines = append(payload.Headlines, translateHeadline{
			Title:   h.Title,
			Summary: h.Summary,
			Tags:    h.Tags,
		})
	}

	payloadJSON, _ := json.Marshal(payload)

	prompt := fmt.Sprintf(`Translate the following JSON content to %s (%s).
Rules:
- Translate the "title", "summary", and "tags" fields
- Keep the JSON structure exactly the same
- Keep proper nouns (company names, product names, person names) in their original form
- Tags should be short translated keywords (1-3 words each)
- Output valid JSON only, no extra text

%s`, LanguageName(targetLang), string(targetLang), string(payloadJSON))

	resp, err := t.client.Generate(ctx, &llm.Request{
		System:      "You are a professional tech news translator. Output valid JSON only.",
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   8192,
		Temperature: 0.2,
	})
	if err != nil {
		return nil, fmt.Errorf("translate to %s: %w", targetLang, err)
	}

	// Parse translated content
	content := extractJSON(resp.Content)
	var translated translatePayload
	if err := json.Unmarshal([]byte(content), &translated); err != nil {
		return nil, fmt.Errorf("parse translation for %s: %w", targetLang, err)
	}

	// Build translated digest (copy structural fields, replace text)
	result := &analyzer.DailyDigest{
		Date:        digest.Date,
		Summary:     translated.Summary,
		GeneratedAt: digest.GeneratedAt,
		TokensUsed:  digest.TokensUsed + resp.TokensIn + resp.TokensOut,
		Cost:        digest.Cost + resp.Cost,
	}

	for i, h := range digest.Headlines {
		headline := analyzer.Headline{
			URL:        h.URL,
			Source:     h.Source,
			Importance: h.Importance,
			Tags:       h.Tags,
		}
		if i < len(translated.Headlines) {
			headline.Title = translated.Headlines[i].Title
			headline.Summary = translated.Headlines[i].Summary
			if len(translated.Headlines[i].Tags) > 0 {
				headline.Tags = translated.Headlines[i].Tags
			}
		} else {
			headline.Title = h.Title
			headline.Summary = h.Summary
		}
		result.Headlines = append(result.Headlines, headline)
	}

	return result, nil
}

// TranslateAll translates a digest to all specified languages concurrently.
// Returns a map of language → translated digest. The source language (zh) is included as-is.
func (t *Translator) TranslateAll(ctx context.Context, digest *analyzer.DailyDigest, langs []Language) map[Language]*analyzer.DailyDigest {
	results := make(map[Language]*analyzer.DailyDigest)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, lang := range langs {
		wg.Add(1)
		go func(l Language) {
			defer wg.Done()
			translated, err := t.Translate(ctx, digest, l)
			if err != nil {
				log.Printf("WARN: translation to %s failed: %v, using source language", l, err)
				translated = digest // Fallback to Chinese
			}
			mu.Lock()
			results[l] = translated
			mu.Unlock()
		}(lang)
	}

	wg.Wait()
	return results
}

// extractJSON extracts a JSON object from a string that may contain markdown fences.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return strings.TrimSpace(s)
}
