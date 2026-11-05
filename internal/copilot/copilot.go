// Package copilot provides AI-powered PR review and PR description generation.
//
// Two providers are supported:
//
//  1. "github-models" — calls https://models.github.ai/inference/chat/completions
//     using the user's GitHub token (Copilot subscribers + many free tiers).
//     This is the default because it returns structured output suitable for
//     a CLI.
//
//  2. "copilot" — shells out to the `gh copilot` extension. Only used when
//     explicitly configured, since the extension is installed separately.
package copilot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/yourorg/gocli/internal/config"
	ghint "github.com/yourorg/gocli/internal/github"
)

const modelsEndpoint = "https://models.github.ai/inference/chat/completions"

// ReviewInput is what we send to the model for code review.
type ReviewInput struct {
	Repo        string // "owner/name"
	PRNumber    int
	Title       string
	Description string
	Diff        string
}

// GeneratePRInput provides context for AI-generated PR title and description.
type GeneratePRInput struct {
	Repo        string // "owner/name"
	Base        string
	Head        string
	Diff        string
	Commits     string
	JiraKey     string
	JiraSummary string
	JiraDesc    string
}

// GeneratedPR contains the AI-generated title and body for a pull request.
type GeneratedPR struct {
	Title string
	Body  string
}

// Review runs a PR review and returns markdown the CLI should print.
func Review(ctx context.Context, in ReviewInput) (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	switch cfg.AI.Provider {
	case "copilot":
		return reviewViaGhCopilot(ctx, in)
	default:
		model := cfg.AI.Model
		if model == "" {
			model = "openai/gpt-4o-mini"
		}
		return reviewViaGitHubModels(ctx, in, model)
	}
}

// GeneratePR creates a PR title and description using AI from diff, commit history, and Jira metadata.
func GeneratePR(ctx context.Context, in GeneratePRInput, jiraURL string) (*GeneratedPR, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	var raw string
	switch cfg.AI.Provider {
	case "copilot":
		raw, err = generatePRViaGhCopilot(ctx, in, jiraURL)
	default:
		model := cfg.AI.Model
		if model == "" {
			model = "openai/gpt-4o-mini"
		}
		raw, err = generatePRViaGitHubModels(ctx, in, jiraURL, model)
	}
	if err != nil {
		return nil, err
	}
	return parseGeneratedPR(raw)
}

func systemPrompt() string {
	return `You are a senior staff engineer doing a pull request review.
Be concise, concrete, and prioritise correctness, security, and clarity.

Output strict markdown with these sections, in order:

## Summary
One paragraph: what the PR does and why.

## Risk
Bulleted list of risks (bugs, regressions, race conditions, security,
backwards compatibility). Mark each as **HIGH / MED / LOW**.

## Suggestions
Numbered list. For each suggestion:
- Start with the file path and target line: 1. [path/to/file:line] explanation or 1. [path/to/file:startLine-endLine] explanation.
- Whenever proposing a concrete code fix, provide a GitHub suggestion block:
` + "```suggestion" + `
replacement code
` + "```" + `

## Tests
What test coverage is missing or weak.

## Verdict
One of: APPROVE, REQUEST CHANGES, COMMENT — with one short reason.`
}

func userPrompt(in ReviewInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Repository: %s\nPR #%d: %s\n\n", in.Repo, in.PRNumber, in.Title)
	if in.Description != "" {
		fmt.Fprintf(&b, "Description:\n%s\n\n", in.Description)
	}
	b.WriteString("Unified diff:\n```diff\n")
	b.WriteString(truncate(in.Diff, 60_000))
	b.WriteString("\n```")
	return b.String()
}

func generatePRSystemPrompt() string {
	return `You are a senior software engineer writing a clean, professional GitHub Pull Request title and description.

Rules:
1. Output format: You MUST return your response in this exact format with no conversational preamble:
TITLE: <concise PR title>
BODY:
<markdown PR description>

2. Title:
   - Use conventional commit style if applicable (e.g. feat: ..., fix: ..., chore: ..., refactor: ...).
   - If a Jira ticket key is provided (e.g. KAN-123), prefix the title with the ticket key like: "feat(KAN-123): summary" or "KAN-123: summary".
   - Keep the title clear, imperative, and under 72 characters.

3. Body:
   - Must be structured markdown.
   - Include the following sections:
     ## Summary
     Concise 1-2 sentence overview of why this change is needed and what it accomplishes.

     ## Changes
     Bulleted list of key code and architectural changes.

     ## Testing / Verification
     How this change was tested or can be verified by reviewers.

   - If a Jira ticket is provided, make sure to reference it in the Summary or at the top as:
     **Jira Ticket:** [KEY](URL) (or [KEY] if no URL is provided).
   - Do NOT invent changes that are not in the diff or commit history.`
}

func generatePRUserPrompt(in GeneratePRInput, jiraURL string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Repository: %s\nBase branch: %s\nHead branch: %s\n\n", in.Repo, in.Base, in.Head)
	if in.JiraKey != "" {
		fmt.Fprintf(&b, "Jira Ticket: %s\n", in.JiraKey)
		if jiraURL != "" {
			fmt.Fprintf(&b, "Jira URL: %s\n", jiraURL)
		}
		if in.JiraSummary != "" {
			fmt.Fprintf(&b, "Jira Summary: %s\n", in.JiraSummary)
		}
		if in.JiraDesc != "" {
			fmt.Fprintf(&b, "Jira Description:\n%s\n\n", truncate(in.JiraDesc, 2000))
		}
	}
	if in.Commits != "" {
		fmt.Fprintf(&b, "Commit History:\n%s\n\n", in.Commits)
	}
	if in.Diff != "" {
		b.WriteString("Unified Diff:\n```diff\n")
		b.WriteString(truncate(in.Diff, 50_000))
		b.WriteString("\n```\n")
	}
	return b.String()
}

// callGitHubModels makes a chat completion request to the GitHub Models inference endpoint.
func callGitHubModels(ctx context.Context, model, sysPrompt, usrPrompt string) (string, error) {
	tok, err := ghint.Token()
	if err != nil {
		return "", fmt.Errorf("github token required for github-models: %w", err)
	}

	body, err := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": sysPrompt},
			{"role": "user", "content": usrPrompt},
		},
		"temperature": 0.2,
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, modelsEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	hc := &http.Client{Timeout: 90 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("github models: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("github models %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("empty response from model")
	}
	return parsed.Choices[0].Message.Content, nil
}

func reviewViaGitHubModels(ctx context.Context, in ReviewInput, model string) (string, error) {
	return callGitHubModels(ctx, model, systemPrompt(), userPrompt(in))
}

func generatePRViaGitHubModels(ctx context.Context, in GeneratePRInput, jiraURL, model string) (string, error) {
	return callGitHubModels(ctx, model, generatePRSystemPrompt(), generatePRUserPrompt(in, jiraURL))
}

func reviewViaGhCopilot(ctx context.Context, in ReviewInput) (string, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", fmt.Errorf("gh CLI not found: %w", err)
	}
	prompt := systemPrompt() + "\n\n" + userPrompt(in)
	cmd := exec.CommandContext(ctx, "gh", "copilot", "explain", prompt)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh copilot: %w (is the extension installed? `gh extension install github/gh-copilot`)\n%s", err, string(out))
	}
	return string(out), nil
}

func generatePRViaGhCopilot(ctx context.Context, in GeneratePRInput, jiraURL string) (string, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", fmt.Errorf("gh CLI not found: %w", err)
	}
	prompt := generatePRSystemPrompt() + "\n\n" + generatePRUserPrompt(in, jiraURL)
	cmd := exec.CommandContext(ctx, "gh", "copilot", "explain", prompt)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh copilot: %w (is the extension installed? `gh extension install github/gh-copilot`)\n%s", err, string(out))
	}
	return string(out), nil
}

func isTitleLine(line string) (string, bool) {
	norm := strings.TrimSpace(line)
	norm = strings.TrimPrefix(norm, "**")
	norm = strings.TrimPrefix(norm, "#")
	norm = strings.TrimSpace(norm)
	lower := strings.ToLower(norm)
	if strings.HasPrefix(lower, "title:") {
		val := norm[len("title:"):]
		val = strings.TrimPrefix(val, "**")
		val = strings.TrimSpace(val)
		val = strings.Trim(val, "`*# \"")
		return val, true
	}
	return "", false
}

func isBodyHeader(line string) bool {
	norm := strings.TrimSpace(line)
	norm = strings.TrimPrefix(norm, "**")
	norm = strings.TrimPrefix(norm, "#")
	norm = strings.TrimSpace(norm)
	lower := strings.ToLower(norm)
	return strings.HasPrefix(lower, "body:") || lower == "body"
}

func parseGeneratedPR(raw string) (*GeneratedPR, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty response from model")
	}

	lines := strings.Split(raw, "\n")
	var title string
	bodyStartIndex := -1
	var firstBodyLine string

	for i, line := range lines {
		if t, ok := isTitleLine(line); ok && title == "" {
			title = t
			continue
		}
		if isBodyHeader(line) && bodyStartIndex == -1 {
			bodyStartIndex = i + 1
			norm := strings.TrimSpace(line)
			norm = strings.TrimPrefix(norm, "**")
			norm = strings.TrimPrefix(norm, "#")
			norm = strings.TrimSpace(norm)
			if idx := strings.Index(strings.ToLower(norm), "body:"); idx != -1 {
				rest := strings.TrimSpace(norm[idx+len("body:"):])
				rest = strings.TrimPrefix(rest, "**")
				rest = strings.TrimSpace(rest)
				if rest != "" {
					firstBodyLine = rest
				}
			}
			break
		}
	}

	var body string
	if bodyStartIndex != -1 {
		var bodyParts []string
		if firstBodyLine != "" {
			bodyParts = append(bodyParts, firstBodyLine)
		}
		if bodyStartIndex < len(lines) {
			bodyParts = append(bodyParts, lines[bodyStartIndex:]...)
		}
		body = strings.TrimSpace(strings.Join(bodyParts, "\n"))
	}

	// Fallback if formatting was non-standard
	if title == "" {
		if len(lines) > 0 {
			title = strings.Trim(strings.TrimSpace(strings.TrimPrefix(lines[0], "#")), "`*# ")
			if body == "" && len(lines) > 1 {
				body = strings.TrimSpace(strings.Join(lines[1:], "\n"))
			}
		}
	}

	if title == "" {
		return nil, fmt.Errorf("could not extract PR title from AI output")
	}

	return &GeneratedPR{
		Title: title,
		Body:  body,
	}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n... (truncated to " + fmt.Sprint(n) + " bytes)"
}
