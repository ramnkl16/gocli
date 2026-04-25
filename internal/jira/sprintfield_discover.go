package jira

import (
	"context"
	"os"
	"strings"
	"sync"

	"github.com/yourorg/gocli/internal/config"
)

// discoverCache remembers the resolved Sprint custom field id per Jira site
// (BaseURL) for the life of the process.
var discoverCache = struct {
	mu    sync.Mutex
	byURL map[string]string
}{byURL: make(map[string]string)}

// DiscoverSprintFieldID calls GET /rest/api/2/field and returns the id of the
// custom field named "Sprint" (Jira Software). Returns ("", nil) if not found
// (e.g. Jira without Software, or sprints not licensed).
func (c *Client) DiscoverSprintFieldID(ctx context.Context) (string, error) {
	fields, _, err := c.Field.GetList(ctx)
	if err != nil {
		return "", err
	}
	// 1) Exact name match, custom
	for _, f := range fields {
		if !f.Custom {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(f.Name), "Sprint") {
			return f.ID, nil
		}
	}
	// 2) Team-managed / variations: "sprint" in the name, array schema
	for _, f := range fields {
		if !f.Custom {
			continue
		}
		ln := strings.ToLower(f.Name)
		if strings.Contains(ln, "sprint") && f.Schema.Type == "array" {
			return f.ID, nil
		}
	}
	return "", nil
}

// discoverSprintFieldIDCached returns cached result per BaseURL when possible.
func (c *Client) discoverSprintFieldIDCached(ctx context.Context) (string, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	discoverCache.mu.Lock()
	if id, ok := discoverCache.byURL[base]; ok {
		discoverCache.mu.Unlock()
		return id, nil
	}
	discoverCache.mu.Unlock()

	id, err := c.DiscoverSprintFieldID(ctx)
	if err != nil {
		return "", err
	}
	discoverCache.mu.Lock()
	if discoverCache.byURL == nil {
		discoverCache.byURL = make(map[string]string)
	}
	discoverCache.byURL[base] = id
	discoverCache.mu.Unlock()
	return id, nil
}

// ResolvedSprintFieldID returns which field id to pass to issue search, using:
//   - GOCLI_JIRA_SPRINT_FIELD: field id, or "none" / "-" to skip
//   - config jira.sprint_field: explicit id, "none", "auto" / empty → discovery
//   - otherwise: discover via GET /rest/api/2/field
func (c *Client) ResolvedSprintFieldID(ctx context.Context) (string, error) {
	if e := strings.TrimSpace(os.Getenv("GOCLI_JIRA_SPRINT_FIELD")); e != "" {
		if e == "none" || e == "-" {
			return "", nil
		}
		return e, nil
	}
	cfg, err := config.Load()
	if err == nil {
		s := strings.TrimSpace(cfg.Jira.SprintField)
		if s == "none" || s == "-" {
			return "", nil
		}
		if s != "" && !strings.EqualFold(s, "auto") {
			return s, nil
		}
	}
	return c.discoverSprintFieldIDCached(ctx)
}
