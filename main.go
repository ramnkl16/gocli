// gocli - one CLI to rule Jira, GitHub, Copilot PR review, and deployments.
//
// See `gocli --help` for the full command tree.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"

	"github.com/yourorg/gocli/cmd"
)

// These variables are populated by `go build -ldflags`.
var (
	version = "0.1.0-dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	loadDotenv()
	cmd.SetBuildInfo(version, commit, date)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// loadDotenv reads project-local .env files, in this order:
//
//  1. <cwd>/.env
//  2. <cwd>/.env.local              (personal overrides, gitignored)
//  3. <binary-dir>/.env             (so it works when launched from elsewhere)
//
// Real OS environment variables always win over .env values — godotenv
// never overwrites an existing key. Missing files are silently ignored,
// so production / CI behaviour is unchanged.
//
// Set GOCLI_NO_DOTENV=1 to disable this entirely.
func loadDotenv() {
	if os.Getenv("GOCLI_NO_DOTENV") != "" {
		return
	}

	candidates := []string{".env", ".env.local"}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), ".env"))
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		_ = godotenv.Load(p)
	}
}
