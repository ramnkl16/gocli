// Package github wraps go-github with credential plumbing and helpers
// for resolving the "current repo".
package github

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	gh "github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"

	"github.com/yourorg/gocli/internal/config"
	"github.com/yourorg/gocli/internal/secrets"
)

// Client wraps a go-github client + the resolved default repo.
type Client struct {
	*gh.Client
	User        string
	DefaultRepo string // "owner/name"
}

// New constructs a GitHub client using the saved token.
func New(ctx context.Context) (*Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	tok, err := secrets.MustGet(secrets.KeyGitHubToken)
	if err != nil {
		return nil, fmt.Errorf("github token: %w", err)
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tok})
	httpClient := oauth2.NewClient(ctx, ts)
	return &Client{
		Client:      gh.NewClient(httpClient),
		User:        cfg.GitHub.User,
		DefaultRepo: cfg.GitHub.DefaultRepo,
	}, nil
}

// Token returns the raw GitHub token (used by AI providers).
func Token() (string, error) {
	return secrets.MustGet(secrets.KeyGitHubToken)
}

// ResolveRepo returns (owner, name) using, in order:
//  1. the explicit override "owner/name" (if non-empty)
//  2. the "origin" git remote in the current dir
//  3. the configured default repo
func (c *Client) ResolveRepo(override string) (owner, name string, err error) {
	if override != "" {
		return splitRepo(override)
	}
	if r, ok := remoteRepo(); ok {
		return splitRepo(r)
	}
	if c.DefaultRepo != "" {
		return splitRepo(c.DefaultRepo)
	}
	return "", "", errors.New("no repo specified, no git remote, no default_repo configured")
}

func splitRepo(s string) (string, string, error) {
	parts := strings.SplitN(strings.TrimSpace(s), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repo must be owner/name, got %q", s)
	}
	return parts[0], strings.TrimSuffix(parts[1], ".git"), nil
}

// remoteRepo runs `git remote get-url origin` and parses out owner/name.
func remoteRepo() (string, bool) {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "", false
	}
	url := strings.TrimSpace(string(out))
	switch {
	case strings.HasPrefix(url, "git@github.com:"):
		url = strings.TrimPrefix(url, "git@github.com:")
	case strings.HasPrefix(url, "https://github.com/"):
		url = strings.TrimPrefix(url, "https://github.com/")
	case strings.HasPrefix(url, "ssh://git@github.com/"):
		url = strings.TrimPrefix(url, "ssh://git@github.com/")
	default:
		return "", false
	}
	url = strings.TrimSuffix(url, ".git")
	if !strings.Contains(url, "/") {
		return "", false
	}
	return url, true
}
