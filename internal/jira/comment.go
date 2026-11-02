package jira

import (
	"context"
	"fmt"
	"strings"

	gojira "github.com/andygrunwald/go-jira/v2/cloud"
)

// AddComment posts a new comment to a Jira issue.
func (c *Client) AddComment(ctx context.Context, key, body string) (*gojira.Comment, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("issue key is required")
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, fmt.Errorf("comment body is required")
	}

	cm, _, err := c.Issue.AddComment(ctx, key, &gojira.Comment{
		Body: body,
	})
	if err != nil {
		return nil, fmt.Errorf("add comment to %s: %w", key, err)
	}
	return cm, nil
}

