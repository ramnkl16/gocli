package jira

import (
	"fmt"
	"strings"
)

// ResolveAssignee maps a shorthand or alias (from config jira.assignee_aliases)
// to the Jira assignee operand for JQL. Unknown names are returned trimmed as-is.
func ResolveAssignee(aliases map[string]string, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("assignee cannot be empty")
	}
	if aliases == nil {
		return name, nil
	}
	lk := strings.ToLower(name)
	for k, v := range aliases {
		key := strings.TrimSpace(k)
		if strings.ToLower(key) != lk {
			continue
		}
		out := strings.TrimSpace(v)
		if out == "" {
			return "", fmt.Errorf("assignee alias %q maps to an empty value", key)
		}
		return out, nil
	}
	return name, nil
}
