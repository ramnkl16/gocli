// Package secrets stores tokens in the OS keyring.
//
// Lookups fall back to environment variables so CI / containers
// (where no keyring is available) keep working:
//
//	JIRA_API_TOKEN, GITHUB_TOKEN
package secrets

import (
	"errors"
	"os"

	"github.com/zalando/go-keyring"
)

const service = "gocli"

// Known keys.
const (
	KeyJiraToken   = "jira_api_token"
	KeyGitHubToken = "github_token"
)

// envFor returns the environment variable name a given key should fall back to.
func envFor(key string) string {
	switch key {
	case KeyJiraToken:
		return "JIRA_API_TOKEN"
	case KeyGitHubToken:
		return "GITHUB_TOKEN"
	default:
		return ""
	}
}

// Set writes a secret to the OS keyring.
func Set(key, value string) error {
	return keyring.Set(service, key, value)
}

// Get reads a secret from the keyring, falling back to env vars.
// Returns ("", nil) if the secret simply isn't set anywhere.
func Get(key string) (string, error) {
	if v, err := keyring.Get(service, key); err == nil {
		return v, nil
	} else if !errors.Is(err, keyring.ErrNotFound) {
		// Keyring failure (e.g. headless box) — try env before giving up.
		if env := envFor(key); env != "" {
			if v := os.Getenv(env); v != "" {
				return v, nil
			}
		}
		return "", err
	}
	if env := envFor(key); env != "" {
		if v := os.Getenv(env); v != "" {
			return v, nil
		}
	}
	return "", nil
}

// Delete removes a secret from the keyring.
// A missing entry is not an error.
func Delete(key string) error {
	err := keyring.Delete(service, key)
	if err != nil && errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

// MustGet is like Get but turns "" into a friendly error.
func MustGet(key string) (string, error) {
	v, err := Get(key)
	if err != nil {
		return "", err
	}
	if v == "" {
		return "", errors.New("not configured — run `gocli auth login`")
	}
	return v, nil
}
