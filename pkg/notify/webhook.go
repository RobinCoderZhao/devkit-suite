package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookConfig holds webhook configuration.
type WebhookConfig struct {
	URL     string            `yaml:"url" json:"url"`
	Headers map[string]string `yaml:"headers" json:"headers"`
}

// WebhookNotifier sends notifications to a webhook URL.
type WebhookNotifier struct {
	config WebhookConfig
	http   *http.Client
}

// NewWebhookNotifier creates a new webhook notifier.
func NewWebhookNotifier(cfg WebhookConfig) *WebhookNotifier {
	return &WebhookNotifier{
		config: cfg,
		http:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (w *WebhookNotifier) Channel() Channel { return ChannelWebhook }

// Send sends a message to the webhook URL.
func (w *WebhookNotifier) Send(ctx context.Context, msg Message) error {
	payload := map[string]string{
		"title":  msg.Title,
		"body":   msg.Body,
		"format": msg.Format,
		"url":    msg.URL,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", w.config.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.config.Headers {
		req.Header.Set(k, v)
	}

	resp, err := w.http.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
