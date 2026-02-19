package notify

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// EmailConfig holds email notification configuration.
type EmailConfig struct {
	SMTPHost string // e.g. "smtp.gmail.com"
	SMTPPort string // e.g. "465" or "587"
	From     string // sender email
	Password string // SMTP password or app-specific password
	To       string // comma-separated recipient emails
}

type emailNotifier struct {
	cfg EmailConfig
}

// NewEmailNotifier creates an email notifier.
func NewEmailNotifier(cfg EmailConfig) Notifier {
	return &emailNotifier{cfg: cfg}
}

func (e *emailNotifier) Channel() Channel {
	return ChannelEmail
}

func (e *emailNotifier) Send(ctx context.Context, msg Message) error {
	recipients := strings.Split(e.cfg.To, ",")
	for i := range recipients {
		recipients[i] = strings.TrimSpace(recipients[i])
	}

	body := buildEmailBody(e.cfg.From, recipients, msg)

	var client *smtp.Client
	var err error
	addr := net.JoinHostPort(e.cfg.SMTPHost, e.cfg.SMTPPort)

	if e.cfg.SMTPPort == "465" {
		client, err = dialTLS(addr, e.cfg.SMTPHost)
	} else {
		client, err = dialSTARTTLS(addr, e.cfg.SMTPHost)
	}
	if err != nil {
		// Fallback: try the other method
		if e.cfg.SMTPPort == "465" {
			altAddr := net.JoinHostPort(e.cfg.SMTPHost, "587")
			client, err = dialSTARTTLS(altAddr, e.cfg.SMTPHost)
		} else {
			altAddr := net.JoinHostPort(e.cfg.SMTPHost, "465")
			client, err = dialTLS(altAddr, e.cfg.SMTPHost)
		}
		if err != nil {
			return fmt.Errorf("SMTP connect failed: %w", err)
		}
	}
	defer client.Close()

	auth := smtp.PlainAuth("", e.cfg.From, e.cfg.Password, e.cfg.SMTPHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth: %w", err)
	}
	if err := client.Mail(e.cfg.From); err != nil {
		return fmt.Errorf("SMTP MAIL FROM: %w", err)
	}
	for _, to := range recipients {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("SMTP RCPT TO %s: %w", to, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}
	if _, err := w.Write([]byte(body)); err != nil {
		return fmt.Errorf("SMTP write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP close data: %w", err)
	}
	return client.Quit()
}

func dialTLS(addr, host string) (*smtp.Client, error) {
	tlsConfig := &tls.Config{ServerName: host}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("TLS dial %s: %w", addr, err)
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("SMTP client: %w", err)
	}
	return client, nil
}

func dialSTARTTLS(addr, host string) (*smtp.Client, error) {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("SMTP client: %w", err)
	}
	tlsConfig := &tls.Config{ServerName: host}
	if err := client.StartTLS(tlsConfig); err != nil {
		client.Close()
		return nil, fmt.Errorf("STARTTLS: %w", err)
	}
	return client, nil
}

// encodeRFC2047 encodes a UTF-8 string for email headers using RFC 2047 base64 encoding.
func encodeRFC2047(s string) string {
	return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(s)) + "?="
}

func buildEmailBody(from string, to []string, msg Message) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("From: =?UTF-8?B?%s?= <%s>\r\n",
		base64.StdEncoding.EncodeToString([]byte("DevKit NewsBot")), from))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", encodeRFC2047(msg.Title)))
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	sb.WriteString("Content-Transfer-Encoding: base64\r\n")
	sb.WriteString("\r\n")

	// Use pre-rendered HTML if available, otherwise plain text
	htmlContent := msg.HTMLBody
	if htmlContent == "" {
		htmlContent = "<pre>" + msg.Body + "</pre>"
	}
	sb.WriteString(base64.StdEncoding.EncodeToString([]byte(htmlContent)))

	return sb.String()
}
