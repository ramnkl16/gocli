package jira

import (
	"testing"
)

func TestExtractIssueKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"fea-KAN-224-auth-flow", "KAN-224"},
		{"bug-LPAD-26763-login-error", "LPAD-26763"},
		{"chg-ENG-123-update-db", "ENG-123"},
		{"fea-kan-224-auth-flow", "KAN-224"},
		{"feature/KAN-224", "KAN-224"},
		{"feature/kan-224", "KAN-224"},
		{"KAN-224", "KAN-224"},
		{"kan-224", "KAN-224"},
		{"fix/KAN-224-some-bug", "KAN-224"},
		{"user/feature-ENG-999_test", "ENG-999"},
		{"[PROJ-456] Fix something", "PROJ-456"},
		{"main", ""},
		{"master", ""},
		{"develop", ""},
		{"HEAD", ""},
		{"release-1.2.0", ""},
		{"v1-2-patch", ""},
		{"", ""},
	}

	for _, tc := range tests {
		actual := ExtractIssueKey(tc.input)
		if actual != tc.expected {
			t.Errorf("ExtractIssueKey(%q) = %q; want %q", tc.input, actual, tc.expected)
		}
	}
}

func TestBranchNameForWorkflow(t *testing.T) {
	tests := []struct {
		ticketType string
		issueKey   string
		summary    string
		expected   string
	}{
		{"fea", "KAN-224", "Add OAuth login support", "fea-KAN-224-add-oauth-login-supp"},
		{"BUG", "eng-123", "crash on startup", "bug-ENG-123-crash-on-startup"},
		{"chg", "PROJ-1", "", "chg-PROJ-1-no-title"},
	}

	for _, tc := range tests {
		actual := BranchNameForWorkflow(tc.ticketType, tc.issueKey, tc.summary)
		if actual != tc.expected {
			t.Errorf("BranchNameForWorkflow(%q, %q, %q) = %q; want %q",
				tc.ticketType, tc.issueKey, tc.summary, actual, tc.expected)
		}
	}
}

