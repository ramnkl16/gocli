package jira

import (
	"context"
	"fmt"
	"strings"

	gojira "github.com/andygrunwald/go-jira/v2/cloud"
)

// TransitionTo moves the issue to a workflow status that matches the given
// name (compared to transition name or destination status name). It returns
// the destination status name as reported by Jira.
func (c *Client) TransitionTo(ctx context.Context, key, target string) (toStatusName string, err error) {
	transitions, _, err := c.Issue.GetTransitions(ctx, key)
	if err != nil {
		return "", fmt.Errorf("transitions: %w", err)
	}
	var match *gojira.Transition
	for i := range transitions {
		if strings.EqualFold(transitions[i].Name, target) ||
			strings.EqualFold(transitions[i].To.Name, target) {
			match = &transitions[i]
			break
		}
	}
	if match == nil {
		names := make([]string, 0, len(transitions))
		for _, t := range transitions {
			names = append(names, t.Name)
		}
		return "", fmt.Errorf("no transition matches %q. available: %s", target, strings.Join(names, ", "))
	}
	if _, err := c.Issue.DoTransition(ctx, key, match.ID); err != nil {
		return "", fmt.Errorf("transition: %w", err)
	}
	if match.To.Name != "" {
		return match.To.Name, nil
	}
	return target, nil
}
