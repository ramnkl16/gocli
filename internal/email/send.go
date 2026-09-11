// Package email sends plain-text notifications via SMTP.
package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

const dialTimeout = 30 * time.Second

// SMTPSettings holds connection parameters for SendPlainText.
type SMTPSettings struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

// SendPlainText sends a plain-text message to a single recipient.
func SendPlainText(ctx context.Context, s SMTPSettings, to, subject, body string) error {
	host := strings.TrimSpace(s.Host)
	if host == "" {
		return fmt.Errorf("smtp host is empty")
	}
	port := s.Port
	if port <= 0 {
		port = 587
	}
	from := strings.TrimSpace(s.From)
	if from == "" {
		return fmt.Errorf("from address is empty")
	}
	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("recipient address is empty")
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return fmt.Errorf("subject is empty")
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return fmt.Errorf("message body is empty")
	}

	msg := buildMessage(from, to, subject, body)
	addr := fmt.Sprintf("%s:%d", host, port)

	dialer := net.Dialer{Timeout: dialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial %s: %w", addr, err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsCfg := &tls.Config{ServerName: host}
		if err := client.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("smtp STARTTLS: %w", err)
		}
	}

	user := strings.TrimSpace(s.User)
	if user == "" {
		user = from
	}
	password := strings.TrimSpace(s.Password)
	if password != "" {
		auth := smtp.PlainAuth("", user, password, host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		_ = w.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return client.Quit()
}

func buildMessage(from, to, subject, body string) string {
	var b strings.Builder
	b.WriteString("From: ")
	b.WriteString(from)
	b.WriteString("\r\nTo: ")
	b.WriteString(to)
	b.WriteString("\r\nSubject: ")
	b.WriteString(subject)
	b.WriteString("\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(body)
	b.WriteString("\r\n")
	return b.String()
}
