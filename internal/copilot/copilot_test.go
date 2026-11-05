package copilot

import (
	"strings"
	"testing"
)

func TestParseGeneratedPR_Standard(t *testing.T) {
	raw := `TITLE: feat(KAN-224): add OAuth token refresh
BODY:
## Summary
Adds automated refresh token rotation.

## Changes
- Update client authentication handler
- Handle HTTP 401 with exponential backoff

## Testing / Verification
- Run unit tests with go test ./...
`

	got, err := parseGeneratedPR(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != "feat(KAN-224): add OAuth token refresh" {
		t.Errorf("got title %q, want %q", got.Title, "feat(KAN-224): add OAuth token refresh")
	}
	if !strings.Contains(got.Body, "Adds automated refresh token rotation.") {
		t.Errorf("body missing summary: %s", got.Body)
	}
	if !strings.Contains(got.Body, "## Changes") {
		t.Errorf("body missing changes section: %s", got.Body)
	}
}

func TestParseGeneratedPR_MarkdownFormatting(t *testing.T) {
	raw := `**TITLE:** ` + "`fix(LPAD-101): resolve nil pointer in logger`" + `
**BODY:**
## Summary
Fixes panic when logger is nil.
`

	got, err := parseGeneratedPR(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != "fix(LPAD-101): resolve nil pointer in logger" {
		t.Errorf("got title %q, want clean title", got.Title)
	}
	if !strings.Contains(got.Body, "Fixes panic when logger is nil.") {
		t.Errorf("unexpected body: %s", got.Body)
	}
}

func TestParseGeneratedPR_SameLineBody(t *testing.T) {
	raw := `Title: chore: update dependencies
Body: Short single-line body description.`

	got, err := parseGeneratedPR(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != "chore: update dependencies" {
		t.Errorf("got title %q", got.Title)
	}
	if got.Body != "Short single-line body description." {
		t.Errorf("got body %q", got.Body)
	}
}

func TestParseGeneratedPR_FallbackFirstLine(t *testing.T) {
	raw := `# feat: automatic changelog generator
This PR adds the changelog command.
It parses conventional commits.`

	got, err := parseGeneratedPR(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != "feat: automatic changelog generator" {
		t.Errorf("got title %q", got.Title)
	}
	if !strings.Contains(got.Body, "This PR adds the changelog command.") {
		t.Errorf("unexpected body: %s", got.Body)
	}
}

func TestParseGeneratedPR_Empty(t *testing.T) {
	_, err := parseGeneratedPR("   ")
	if err == nil {
		t.Fatal("expected error for empty raw output")
	}
}

