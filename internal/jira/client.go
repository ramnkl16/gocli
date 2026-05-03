// Package jira is a thin wrapper around go-jira/v2 that pulls credentials
// from gocli's config + keyring.
package jira

import (
	"context"
	"errors"
	"fmt"

	gojira "github.com/andygrunwald/go-jira/v2/cloud"

	"github.com/yourorg/gocli/internal/config"
	"github.com/yourorg/gocli/internal/secrets"
)

// Client wraps a go-jira client with the connected user's email handy.
type Client struct {
	*gojira.Client
	Email           string
	Project         string
	BaseURL         string
	AssigneeAliases map[string]string
}

// New constructs a Jira client from saved config + keyring secrets.
func New(_ context.Context) (*Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if cfg.Jira.BaseURL == "" || cfg.Jira.Email == "" {
		return nil, errors.New("jira not configured — run `gocli auth login`")
	}
	tok, err := secrets.MustGet(secrets.KeyJiraToken)
	if err != nil {
		return nil, fmt.Errorf("jira token: %w", err)
	}

	tp := gojira.BasicAuthTransport{
		Username: cfg.Jira.Email,
		APIToken: tok,
	}
	c, err := gojira.NewClient(cfg.Jira.BaseURL, tp.Client())
	if err != nil {
		return nil, fmt.Errorf("jira client: %w", err)
	}
	aliases := cfg.Jira.AssigneeAliases
	if aliases == nil {
		aliases = map[string]string{}
	}
	return &Client{
		Client:          c,
		Email:           cfg.Jira.Email,
		Project:         cfg.Jira.Project,
		BaseURL:         cfg.Jira.BaseURL,
		AssigneeAliases: aliases,
	}, nil
}
