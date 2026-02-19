package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TelegramConfig holds Telegram bot configuration.
type TelegramConfig struct {
	BotToken  string `yaml:"bot_token" json:"bot_token"`
	ChannelID string `yaml:"channel_id" json:"channel_id"`
}

// TelegramNotifier sends messages via Telegram Bot API.
type TelegramNotifier struct {
	config TelegramConfig
	http   *http.Client
}

// NewTelegramNotifier creates a new Telegram notifier.
func NewTelegramNotifier(cfg TelegramConfig) *TelegramNotifier {
	return &TelegramNotifier{
		config: cfg,
		http:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *TelegramNotifier) Channel() Channel { return ChannelTelegram }

// Send sends a message via Telegram.
func (t *TelegramNotifier) Send(ctx context.Context, msg Message) error {
	text := msg.Body
	if msg.Title != "" {
		text = fmt.Sprintf("*%s*\n\n%s", escapeMarkdown(msg.Title), msg.Body)
	}
	if msg.URL != "" {
		text += fmt.Sprintf("\n\n🔗 [查看详情](%s)", msg.URL)
	}

	payload := map[string]interface{}{
		"chat_id":    t.config.ChannelID,
		"text":       text,
		"parse_mode": "MarkdownV2",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.config.BotToken)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.http.Do(req)
	if err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// escapeMarkdown escapes special characters for Telegram MarkdownV2.
func escapeMarkdown(text string) string {
	special := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	result := text
	for _, ch := range special {
		result = replaceAll(result, ch, "\\"+ch)
	}
	return result
}

func replaceAll(s, old, new string) string {
	var b bytes.Buffer
	for i := 0; i < len(s); {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			b.WriteString(new)
			i += len(old)
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}
