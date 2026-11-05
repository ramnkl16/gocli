package copilot

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// InlineSuggestion represents a code review comment targeted at a specific file and line.
type InlineSuggestion struct {
	Path      string // File path relative to repository root
	StartLine int    // Optional start line for multi-line suggestion (0 if single-line)
	Line      int    // Target line (or end line for multi-line)
	Body      string // Complete suggestion text (including suggestion block)
	Code      string // Replacement code inside ```suggestion ... ``` if present
}

var (
	// Matches suggestion item headers such as:
	// 1. `internal/auth/token.go:42` - Add error handling
	// 1. [internal/auth/token.go:42-45] Refactor loop
	// - `cmd/pr.go:88`: Simplify condition
	// 2. internal/ui/ui.go:15: Check nil pointer
	itemHeaderRegex = regexp.MustCompile(`(?i)(?:^|\n)\s*(?:\d+\.|[-*])?\s*[` + "`" + `\[]?([a-zA-Z0-9_\-./\\]+\.[a-zA-Z0-9]+)[:#]L?(\d+)(?:-(\d+))?[` + "`" + `\]]?:?\s*(.*)`)

	// Matches suggestion code blocks
	suggestionCodeRegex = regexp.MustCompile("(?s)```suggestion\\s*\\r?\\n(.*?)\\r?\\n```")
)

// ParseInlineSuggestions parses review markdown to extract actionable line-level suggestions.
func ParseInlineSuggestions(reviewText string) []InlineSuggestion {
	var suggestions []InlineSuggestion

	// Extract ## Suggestions section if present
	section := extractSection(reviewText, "Suggestions")
	if section == "" {
		section = reviewText
	}

	lines := strings.Split(section, "\n")
	type rawItem struct {
		path      string
		startLine int
		line      int
		header    string
		bodyLines []string
	}

	var items []*rawItem
	var current *rawItem

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") && current != nil {
			// Hit next section
			break
		}

		if match := itemHeaderRegex.FindStringSubmatch(line); len(match) >= 4 {
			path := strings.ReplaceAll(match[1], `\`, `/`)
			path = strings.TrimPrefix(path, "./")
			line1, _ := strconv.Atoi(match[2])
			line2 := 0
			if len(match) > 3 && match[3] != "" {
				line2, _ = strconv.Atoi(match[3])
			}

			startLine := 0
			targetLine := line1
			if line2 > line1 {
				startLine = line1
				targetLine = line2
			}

			item := &rawItem{
				path:      path,
				startLine: startLine,
				line:      targetLine,
				header:    strings.TrimSpace(match[4]),
			}
			items = append(items, item)
			current = item
			continue
		}

		if current != nil {
			current.bodyLines = append(current.bodyLines, line)
		}
	}

	for _, it := range items {
		body := it.header
		if len(it.bodyLines) > 0 {
			extra := strings.TrimSpace(strings.Join(it.bodyLines, "\n"))
			if body == "" {
				body = extra
			} else if extra != "" {
				body = body + "\n\n" + extra
			}
		}

		code := ""
		if match := suggestionCodeRegex.FindStringSubmatch(body); len(match) > 1 {
			code = match[1]
		}

		suggestions = append(suggestions, InlineSuggestion{
			Path:      it.path,
			StartLine: it.startLine,
			Line:      it.line,
			Body:      body,
			Code:      code,
		})
	}

	return suggestions
}

// ApplySuggestionsDirectly applies all parsed suggestion code blocks to files on disk.
// Returns the list of modified file paths.
func ApplySuggestionsDirectly(workdir string, suggestions []InlineSuggestion) ([]string, error) {
	byFile := make(map[string][]InlineSuggestion)
	for _, s := range suggestions {
		if strings.TrimSpace(s.Code) == "" || s.Path == "" || s.Line <= 0 {
			continue
		}
		byFile[s.Path] = append(byFile[s.Path], s)
	}

	var modified []string

	for relPath, fileSuggestions := range byFile {
		fullPath := filepath.Join(workdir, filepath.FromSlash(relPath))
		contentBytes, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", relPath, err)
		}

		content := string(contentBytes)
		hasCRLF := strings.Contains(content, "\r\n")
		normalized := strings.ReplaceAll(content, "\r\n", "\n")
		lines := strings.Split(normalized, "\n")

		// Sort suggestions descending by Line so edits near the bottom do not offset earlier lines
		sort.Slice(fileSuggestions, func(i, j int) bool {
			return fileSuggestions[i].Line > fileSuggestions[j].Line
		})

		for _, sug := range fileSuggestions {
			startIdx := sug.Line - 1
			endIdx := sug.Line
			if sug.StartLine > 0 && sug.StartLine <= sug.Line {
				startIdx = sug.StartLine - 1
			}

			if startIdx < 0 {
				startIdx = 0
			}
			if startIdx > len(lines) {
				startIdx = len(lines)
			}
			if endIdx > len(lines) {
				endIdx = len(lines)
			}

			newLines := strings.Split(strings.ReplaceAll(sug.Code, "\r\n", "\n"), "\n")
			lines = append(lines[:startIdx], append(newLines, lines[endIdx:]...)...)
		}

		joiner := "\n"
		if hasCRLF {
			joiner = "\r\n"
		}
		output := strings.Join(lines, joiner)
		if err := os.WriteFile(fullPath, []byte(output), 0644); err != nil {
			return nil, fmt.Errorf("write %s: %w", relPath, err)
		}

		modified = append(modified, relPath)
	}

	sort.Strings(modified)
	return modified, nil
}

func extractSection(markdown, heading string) string {
	lowerHeading := strings.ToLower(heading)
	lines := strings.Split(markdown, "\n")
	var sectionLines []string
	capturing := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			title := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")))
			if strings.Contains(title, lowerHeading) {
				capturing = true
				continue
			} else if capturing {
				break
			}
		}
		if capturing {
			sectionLines = append(sectionLines, line)
		}
	}

	return strings.TrimSpace(strings.Join(sectionLines, "\n"))
}

