// Package copilot provides AI-powered PR review.
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

// ReviewInput is what we send to the model.
type ReviewInput struct {
	Repo        string // "owner/name"
	PRNumber    int
	Title       string
	Description string
	Diff        string
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
Numbered list. Each item must reference a file path and, when possible,
a line range. Prefer fixes that the author can apply with one keystroke.

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

// reviewViaGitHubModels calls the GitHub Models inference endpoint.
func reviewViaGitHubModels(ctx context.Context, in ReviewInput, model string) (string, error) {
	tok, err := ghint.Token()
	if err != nil {
		return "", fmt.Errorf("github token required for github-models: %w", err)
	}

	body, _ := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt()},
			{"role": "user", "content": userPrompt(in)},
		},
		"temperature": 0.2,
	})

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

// reviewViaGhCopilot pipes the diff into `gh copilot explain` and decorates
// the output with the standard gocli sections.
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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n... (diff truncated to " + fmt.Sprint(n) + " bytes)"
}
