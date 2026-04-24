package jira

import (
	"strings"
	"unicode"
)

const summarySlugRunes = 20

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
