package copilot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseInlineSuggestions(t *testing.T) {
	review := `## Summary
Great PR with minor fixes needed.

## Risk
- Low

## Suggestions
1. ` + "`internal/auth/token.go:42`" + ` - Check if token is expired before decoding:
` + "```suggestion" + `
if token.IsExpired() {
	return nil, ErrTokenExpired
}
` + "```" + `

2. [cmd/pr.go:80-85] Simplify HTTP request creation:
` + "```suggestion" + `
req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
` + "```" + `

3. ` + "`internal\\ui\\ui.go:12`" + `: Add nil check for message

## Tests
Add unit tests for expired token.

## Verdict
REQUEST CHANGES`

	suggestions := ParseInlineSuggestions(review)
	if len(suggestions) != 3 {
		t.Fatalf("got %d suggestions, want 3", len(suggestions))
	}

	// Suggestion 1
	s1 := suggestions[0]
	if s1.Path != "internal/auth/token.go" {
		t.Errorf("s1 path = %q, want internal/auth/token.go", s1.Path)
	}
	if s1.Line != 42 || s1.StartLine != 0 {
		t.Errorf("s1 lines: start=%d line=%d, want start=0 line=42", s1.StartLine, s1.Line)
	}
	if !strings.Contains(s1.Code, "ErrTokenExpired") {
		t.Errorf("s1 code missing ErrTokenExpired: %s", s1.Code)
	}

	// Suggestion 2 (multi-line)
	s2 := suggestions[1]
	if s2.Path != "cmd/pr.go" {
		t.Errorf("s2 path = %q, want cmd/pr.go", s2.Path)
	}
	if s2.StartLine != 80 || s2.Line != 85 {
		t.Errorf("s2 lines: start=%d line=%d, want start=80 line=85", s2.StartLine, s2.Line)
	}
	if !strings.Contains(s2.Code, "NewRequestWithContext") {
		t.Errorf("s2 code missing NewRequestWithContext: %s", s2.Code)
	}

	// Suggestion 3 (backslash path, no code block)
	s3 := suggestions[2]
	if s3.Path != "internal/ui/ui.go" {
		t.Errorf("s3 path = %q, want internal/ui/ui.go", s3.Path)
	}
	if s3.Line != 12 {
		t.Errorf("s3 line = %d, want 12", s3.Line)
	}
	if s3.Code != "" {
		t.Errorf("s3 code expected empty, got %q", s3.Code)
	}
}

func TestApplySuggestionsDirectly(t *testing.T) {
	dir := t.TempDir()

	code := `package main

import "fmt"

func main() {
	x := 10
	fmt.Println(x)
}
`
	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
		t.Fatal(err)
	}

	suggestions := []InlineSuggestion{
		{
			Path: "main.go",
			Line: 6,
			Code: "\tx := 20",
		},
		{
			Path: "main.go",
			Line: 7,
			Code: "\tfmt.Printf(\"x = %d\\n\", x)",
		},
	}

	modified, err := ApplySuggestionsDirectly(dir, suggestions)
	if err != nil {
		t.Fatalf("ApplySuggestionsDirectly: %v", err)
	}
	if len(modified) != 1 || modified[0] != "main.go" {
		t.Fatalf("modified files = %v, want [main.go]", modified)
	}

	updated, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	updatedStr := string(updated)
	if !strings.Contains(updatedStr, "x := 20") {
		t.Errorf("file not updated with line 6 suggestion: %s", updatedStr)
	}
	if !strings.Contains(updatedStr, `fmt.Printf("x = %d\n", x)`) {
		t.Errorf("file not updated with line 7 suggestion: %s", updatedStr)
	}
}

