// Package config loads and persists the user-level gocli configuration.
//
// Config file lives at ~/.gocli/config.yaml (overridable with GOCLI_CONFIG).
// Secrets are NEVER stored in this file; they live in the OS keyring
// (see internal/secrets) or environment variables.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config is the on-disk user configuration.
type Config struct {
	Jira   JiraConfig   `mapstructure:"jira"   yaml:"jira"`
	GitHub GitHubConfig `mapstructure:"github" yaml:"github"`
	AI     AIConfig     `mapstructure:"ai"     yaml:"ai"`
}

type JiraConfig struct {
	BaseURL string `mapstructure:"base_url" yaml:"base_url"`
	Email   string `mapstructure:"email"    yaml:"email"`
	Project string `mapstructure:"project"  yaml:"project"`
	// SprintField is the Jira custom field id for the Sprint field (Jira Software).
	// Most Cloud sites use "customfield_10020". Empty disables requesting it.
	SprintField string `mapstructure:"sprint_field" yaml:"sprint_field"`
	// AssigneeAliases map short names to Jira assignee identifiers (email, account id,
	// username, etc.). Used by `jira list --assignee <name>` when names match keys
	// (ASCII case-insensitive).
	AssigneeAliases map[string]string `mapstructure:"assignee_aliases" yaml:"assignee_aliases"`
}

type GitHubConfig struct {
	DefaultRepo string `mapstructure:"default_repo" yaml:"default_repo"`
	User        string `mapstructure:"user"         yaml:"user"`
}

type AIConfig struct {
	// Provider: "copilot" (uses gh copilot extension) or "github-models"
	// (calls https://models.github.ai with the GitHub token).
	Provider string `mapstructure:"provider" yaml:"provider"`
	Model    string `mapstructure:"model"    yaml:"model"`
}

// Dir returns the directory holding gocli's config and caches.
func Dir() (string, error) {
	if v := os.Getenv("GOCLI_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gocli"), nil
}

// Path returns the absolute path to the config file.
func Path() (string, error) {
	if v := os.Getenv("GOCLI_CONFIG"); v != "" {
		return v, nil
	}
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.yaml"), nil
}

// Load reads the config from disk. A missing file is not an error;
// you get a zero-value Config back so first-run works smoothly.
func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	v := viper.New()
	v.SetConfigFile(p)
	v.SetConfigType("yaml")

	v.SetDefault("ai.provider", "github-models")
	v.SetDefault("ai.model", "openai/gpt-4o-mini")

	if _, err := os.Stat(p); err == nil {
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("read config %s: %w", p, err)
		}
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	return &c, nil
}

// Save writes the config to disk, creating the directory if needed.
func Save(c *Config) error {
	d, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0o700); err != nil {
		return err
	}
	p, err := Path()
	if err != nil {
		return err
	}

	v := viper.New()
	v.SetConfigFile(p)
	v.SetConfigType("yaml")
	v.Set("jira", c.Jira)
	v.Set("github", c.GitHub)
	v.Set("ai", c.AI)
	return v.WriteConfigAs(p)
}
