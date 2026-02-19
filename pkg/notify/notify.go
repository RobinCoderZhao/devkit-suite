// Package notify provides a unified notification dispatch system
// supporting Telegram, Email, Slack, and Webhook channels.
package notify

import (
	"context"
	"fmt"
	"log/slog"
)

// Channel represents a notification channel type.
type Channel string

const (
	ChannelTelegram Channel = "telegram"
	ChannelEmail    Channel = "email"
	ChannelSlack    Channel = "slack"
	ChannelWebhook  Channel = "webhook"
)

// Message represents a notification message.
type Message struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	HTMLBody string `json:"html_body,omitempty"` // Rich HTML for email
	Format   string `json:"format"`              // "markdown", "html", "plain"
	URL      string `json:"url,omitempty"`
}

// Notifier defines the interface for sending notifications.
type Notifier interface {
	Send(ctx context.Context, msg Message) error
	Channel() Channel
}

// Dispatcher routes messages to the appropriate notification channels.
type Dispatcher struct {
	notifiers map[Channel]Notifier
	emailCfg  EmailConfig
	logger    *slog.Logger
}

// NewDispatcher creates a new notification dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		notifiers: make(map[Channel]Notifier),
		logger:    slog.Default(),
	}
}

// SetEmailConfig stores the email configuration for per-recipient dispatch.
func (d *Dispatcher) SetEmailConfig(cfg EmailConfig) {
	d.emailCfg = cfg
}

// EmailConfig returns the stored email configuration.
func (d *Dispatcher) EmailConfig() EmailConfig {
	return d.emailCfg
}

// Register adds a notifier to the dispatcher.
func (d *Dispatcher) Register(n Notifier) {
	d.notifiers[n.Channel()] = n
}

// Dispatch sends a message to the specified channels.
func (d *Dispatcher) Dispatch(ctx context.Context, channels []Channel, msg Message) error {
	var errs []error
	for _, ch := range channels {
		notifier, ok := d.notifiers[ch]
		if !ok {
			d.logger.Warn("notifier not registered", "channel", ch)
			continue
		}
		if err := notifier.Send(ctx, msg); err != nil {
			d.logger.Error("notification failed", "channel", ch, "error", err)
			errs = append(errs, fmt.Errorf("%s: %w", ch, err))
		} else {
			d.logger.Info("notification sent", "channel", ch, "title", msg.Title)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to send %d/%d notifications", len(errs), len(channels))
	}
	return nil
}

// SendAll sends a message to all registered channels.
func (d *Dispatcher) SendAll(ctx context.Context, msg Message) error {
	channels := make([]Channel, 0, len(d.notifiers))
	for ch := range d.notifiers {
		channels = append(channels, ch)
	}
	return d.Dispatch(ctx, channels, msg)
}
