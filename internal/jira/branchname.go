package jira

import (
	"regexp"
	"strings"
	"unicode"
)

const summarySlugRunes = 20

var (
	gocliBranchRe        = regexp.MustCompile(`(?i)^(?:bug|fea|chg)-([A-Za-z][A-Za-z0-9]+-[0-9]+)(?:-.*)?$`)
	uppercaseJiraKeyRe   = regexp.MustCompile(`(?:^|[[\]/_.\s():-])([A-Z][A-Z0-9]+-[0-9]+)(?:$|[[\]/_.\s():-])`)
	caseInsensitiveKeyRe = regexp.MustCompile(`(?i)(?:^|[[\]/_.\s():-])([A-Za-z][A-Za-z0-9]+-[0-9]+)(?:$|[[\]/_.\s():-])`)
)

// ExtractIssueKey extracts a Jira issue key (e.g. "KAN-224" or "ENG-1234") from a branch name,
// commit message, or arbitrary string. Returns "" if no valid Jira issue key is found.
func ExtractIssueKey(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	// 1. Check if branch matches standard gocli convention: {bug|fea|chg}-{KEY}-{slug}
	if m := gocliBranchRe.FindStringSubmatch(input); len(m) > 1 {
		return strings.ToUpper(m[1])
	}

	// 2. Check for an uppercase Jira key anywhere in the string
	if m := uppercaseJiraKeyRe.FindStringSubmatch(input); len(m) > 1 {
		key := m[1]
		if !isIgnoredKeyPrefix(key) {
			return key
		}
	}

	// 3. Fallback to case-insensitive check (e.g. kan-224 -> KAN-224)
	if m := caseInsensitiveKeyRe.FindStringSubmatch(input); len(m) > 1 {
		key := strings.ToUpper(m[1])
		if !isIgnoredKeyPrefix(key) {
			return key
		}
	}

	return ""
}

func isIgnoredKeyPrefix(key string) bool {
	parts := strings.SplitN(key, "-", 2)
	if len(parts) == 0 {
		return true
	}
	p := strings.ToLower(parts[0])
	switch p {
	case "release", "patch", "version", "build", "rev", "step":
		return true
	}
	// Check for version-like patterns such as v1, v2, etc.
	if strings.HasPrefix(p, "v") && len(p) > 1 {
		allDigits := true
		for _, c := range p[1:] {
			if c < '0' || c > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return true
		}
	}
	return false
}

// BranchNameForWorkflow builds a git branch from ticket kind, Jira key, and issue summary:
//
//	{type}-{KEY}-{slug}
//
// type is one of: bug, fea, chg. KEY is the Jira issue key (e.g. ENG-1234).
// slug is the first 20 runes of summary, lowercased, with spaces and special
// characters turned into single hyphens; empty slug becomes "no-title".
func BranchNameForWorkflow(ticketType, issueKey, summary string) string {
	typ := strings.ToLower(strings.TrimSpace(ticketType))
	key := strings.ToUpper(strings.TrimSpace(issueKey))
	slug := slugFromSummaryTruncated(summary, summarySlugRunes)
	return typ + "-" + key + "-" + slug
}

func slugFromSummaryTruncated(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "no-title"
	}
	r := []rune(s)
	if len(r) > maxRunes {
		r = r[:maxRunes]
	}
	s = string(r)
	var b strings.Builder
	for _, c := range s {
		switch {
		case unicode.IsLetter(c) || unicode.IsNumber(c):
			b.WriteRune(unicode.ToLower(c))
		default:
			b.WriteRune('-')
		}
	}
	out := b.String()
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	out = strings.Trim(out, "-")
	if out == "" {
		return "no-title"
	}
	return out
}
