package jira

import (
	"fmt"
	"strings"

	gojira "github.com/andygrunwald/go-jira/v2/cloud"
)

// FormatSprintColumn returns human-readable sprint names for the list view.
// The API encodes sprints in a custom field (array of sprint objects) or, in
// rare cases, the legacy "sprint" key on issue fields.
func FormatSprintColumn(f *gojira.IssueFields) string {
	if f == nil {
		return "—"
	}
	if f.Sprint != nil && f.Sprint.Name != "" {
		return f.Sprint.Name
	}
	// Jira Software: sprints are almost always in a custom field, stored in Unknowns
	// (key e.g. customfield_10020). Prefer keys that look like the Sprint field
	// so we do not pick up unrelated custom fields.
	if f.Unknowns != nil {
		// 1) Keys whose name or clause hints at sprints
		for k, v := range f.Unknowns {
			lk := strings.ToLower(k)
			if strings.Contains(lk, "sprint") || strings.Contains(lk, "greenhopper") {
				if s := sprintNamesFromValue(v); s != "" {
					return s
				}
			}
		}
		// 2) Any customfield_* with sprint-shaped JSON
		for k, v := range f.Unknowns {
			if !strings.HasPrefix(k, "customfield_") {
				continue
			}
			if s := sprintNamesFromValue(v); s != "" {
				return s
			}
		}
	}
	return "—"
}

func sprintNamesFromValue(v interface{}) string {
	switch x := v.(type) {
	case []interface{}:
		return joinSprintNames(x)
	case map[string]interface{}:
		// single sprint object
		if n, ok := x["name"].(string); ok && n != "" {
			return n
		}
	case string:
		if x != "" {
			return x
		}
	}
	return ""
}

func joinSprintNames(items []interface{}) string {
	var names []string
	for _, it := range items {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		n := sprintNameFromMap(m)
		if strings.TrimSpace(n) != "" {
			names = append(names, strings.TrimSpace(n))
		}
	}
	if len(names) == 0 {
		return ""
	}
	return strings.Join(names, ", ")
}

func sprintNameFromMap(m map[string]interface{}) string {
	if v, ok := m["name"]; ok && v != nil {
		switch t := v.(type) {
		case string:
			if t != "" {
				return t
			}
		default:
			s := fmt.Sprint(t)
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	if v, ok := m["value"].(string); ok && v != "" {
		return v
	}
	return ""
}
