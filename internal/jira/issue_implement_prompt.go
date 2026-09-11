package jira

import (
	"fmt"
	"regexp"
	"strings"

	gojira "github.com/andygrunwald/go-jira/v2/cloud"
)

var acceptanceCriteriaHeading = regexp.MustCompile(`(?im)^\s*(?:#{1,3}\s*)?(?:acceptance\s+criteria|ac)\s*:?\s*$`)

// IssueImplementPrompt builds an agent prompt from a Jira issue title and acceptance criteria.
func IssueImplementPrompt(c *Client, is *gojira.Issue) string {
	if is == nil {
		return ""
	}
	base := strings.TrimRight(c.BaseURL, "/")
	url := fmt.Sprintf("%s/browse/%s", base, is.Key)

	summary := ""
	description := ""
	if is.Fields != nil {
		summary = strings.TrimSpace(is.Fields.Summary)
		description = strings.TrimSpace(is.Fields.Description)
	}

	body, acceptance := splitAcceptanceCriteria(description)

	var b strings.Builder
	fmt.Fprintf(&b, "Implement this Jira ticket.\n\n")
	fmt.Fprintf(&b, "# %s: %s\n\n", is.Key, summary)
	fmt.Fprintf(&b, "- **URL:** %s\n", url)
	if is.Fields != nil {
		if is.Fields.Status != nil {
			fmt.Fprintf(&b, "- **Status:** %s\n", is.Fields.Status.Name)
		}
		if is.Fields.Priority != nil {
			fmt.Fprintf(&b, "- **Priority:** %s\n", is.Fields.Priority.Name)
		}
	}
	if body != "" {
		fmt.Fprintf(&b, "\n## Description\n\n%s\n", body)
	}
	if acceptance != "" {
		fmt.Fprintf(&b, "\n## Acceptance Criteria\n\n%s\n", acceptance)
	} else if description != "" {
		fmt.Fprintf(&b, "\n## Description (includes acceptance criteria)\n\n%s\n", description)
	}
	fmt.Fprintf(&b, `
## Instructions

- Implement the ticket to satisfy the acceptance criteria.
- Match existing code patterns and keep the change focused.
- Run relevant tests (e.g. go test ./...) before finishing.
- Do not commit or push; the CLI handles git after you finish.
`)
	return b.String()
}

// splitAcceptanceCriteria separates a Jira description into body and acceptance criteria sections.
func splitAcceptanceCriteria(description string) (body, acceptance string) {
	description = strings.TrimSpace(description)
	if description == "" {
		return "", ""
	}
	loc := acceptanceCriteriaHeading.FindStringIndex(description)
	if loc == nil {
		return description, ""
	}
	body = strings.TrimSpace(description[:loc[0]])
	acceptance = strings.TrimSpace(description[loc[1]:])
	return body, acceptance
}
