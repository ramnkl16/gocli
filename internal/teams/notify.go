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

// MessagePayload is the adaptive card envelope accepted by Teams Workflows webhooks.
type MessagePayload struct {
	Type        string           `json:"type"`
	Attachments []CardAttachment `json:"attachments"`
}

type CardAttachment struct {
	ContentType string       `json:"contentType"`
	Content     AdaptiveCard `json:"content"`
}

type AdaptiveCard struct {
	Schema  string      `json:"$schema"`
	Type    string      `json:"type"`
	Version string      `json:"version"`
	Body    []TextBlock `json:"body"`
}

type TextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Wrap bool   `json:"wrap"`
}

// PostMessage POSTs a plain-text adaptive card to a Teams webhook URL.
func PostMessage(ctx context.Context, webhookURL, text string) error {
	webhookURL = strings.TrimSpace(webhookURL)
	if webhookURL == "" {
		return fmt.Errorf("teams webhook URL is empty")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("message text is empty")
	}

	body, err := json.Marshal(MessagePayload{
		Type: "message",
		Attachments: []CardAttachment{
			{
				ContentType: "application/vnd.microsoft.card.adaptive",
				Content: AdaptiveCard{
					Schema:  "http://adaptivecards.io/schemas/adaptive-card.json",
					Type:    "AdaptiveCard",
					Version: "1.2",
					Body: []TextBlock{
						{
							Type: "TextBlock",
							Text: text,
							Wrap: true,
						},
					},
				},
			},
		},
	})
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
