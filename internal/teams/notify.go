// Package teams sends messages to Microsoft Teams via Incoming Webhook URLs.
package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SimplePayload is the JSON shape accepted by classic Teams Incoming Webhooks.
type SimplePayload struct {
	Text string `json:"text"`
}

// PostMessage POSTs plain text to a webhook URL.
func PostMessage(ctx context.Context, webhookURL, text string) error {
	webhookURL = strings.TrimSpace(webhookURL)
	if webhookURL == "" {
		return fmt.Errorf("teams webhook URL is empty")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("message text is empty")
	}

	body, err := json.Marshal(SimplePayload{Text: text})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("teams webhook request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			return fmt.Errorf("teams webhook returned %s", resp.Status)
		}
		return fmt.Errorf("teams webhook returned %s: %s", resp.Status, msg)
	}
	return nil
}
