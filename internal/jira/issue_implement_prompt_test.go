package jira

import "testing"

func TestSplitAcceptanceCriteria(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		in         string
		wantBody   string
		wantAccept string
	}{
		{
			name: "heading section",
			in: "User story text.\n\nAcceptance Criteria:\n- Login works\n- Errors shown",
			wantBody:   "User story text.",
			wantAccept: "- Login works\n- Errors shown",
		},
		{
			name:       "markdown heading",
			in:         "## Description\n\nDetails here.\n\n## Acceptance Criteria\n\n- AC one",
			wantBody:   "## Description\n\nDetails here.",
			wantAccept: "- AC one",
		},
		{
			name:       "no section",
			in:         "Only a description.",
			wantBody:   "Only a description.",
			wantAccept: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			body, accept := splitAcceptanceCriteria(tt.in)
			if body != tt.wantBody {
				t.Fatalf("body = %q, want %q", body, tt.wantBody)
			}
			if accept != tt.wantAccept {
				t.Fatalf("acceptance = %q, want %q", accept, tt.wantAccept)
			}
		})
	}
}
