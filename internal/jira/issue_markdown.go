package jira

import (
	"fmt"
	"strings"

	gojira "github.com/andygrunwald/go-jira/v2/cloud"
)

// IssueAsMarkdown returns markdown suitable for pasting into AI assistants (e.g. Cursor).
func IssueAsMarkdown(c *Client, is *gojira.Issue) string {
	if is == nil {
		return ""
	}
	base := strings.TrimRight(c.BaseURL, "/")
	url := fmt.Sprintf("%s/browse/%s", base, is.Key)

	var b strings.Builder
	summary := ""
	if is.Fields != nil {
		summary = is.Fields.Summary
	}
	fmt.Fprintf(&b, "# %s: %s\n\n", is.Key, summary)
	fmt.Fprintf(&b, "- **URL:** %s\n", url)
	if is.Fields != nil {
		if is.Fields.Status != nil {
			fmt.Fprintf(&b, "- **Status:** %s\n", is.Fields.Status.Name)
		}
		if is.Fields.Assignee != nil {
			fmt.Fprintf(&b, "- **Assignee:** %s\n", is.Fields.Assignee.DisplayName)
		}
		if is.Fields.Reporter != nil {
			fmt.Fprintf(&b, "- **Reporter:** %s\n", is.Fields.Reporter.DisplayName)
		}
		if is.Fields.Priority != nil {
			fmt.Fprintf(&b, "- **Priority:** %s\n", is.Fields.Priority.Name)
		}
	}
	if is.Fields != nil && strings.TrimSpace(is.Fields.Description) != "" {
		fmt.Fprintf(&b, "\n## Description\n\n%s\n", strings.TrimSpace(is.Fields.Description))
	}
	return b.String()
}
