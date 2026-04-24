package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/yourorg/gocli/internal/config"
	"github.com/yourorg/gocli/internal/secrets"
	"github.com/yourorg/gocli/internal/ui"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Login / logout / status for Jira and GitHub",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Interactively configure Jira and GitHub credentials",
	RunE:  runAuthLogin,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show which integrations are configured",
	RunE:  runAuthStatus,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored Jira and GitHub credentials",
	RunE:  runAuthLogout,
}

func init() {
	authCmd.AddCommand(authLoginCmd, authStatusCmd, authLogoutCmd)
}

func runAuthLogin(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	in := bufio.NewReader(os.Stdin)

	ui.Section("Jira")
	cfg.Jira.BaseURL = prompt(in, "Jira base URL (e.g. https://acme.atlassian.net)", cfg.Jira.BaseURL)
	cfg.Jira.Email = prompt(in, "Jira account email", cfg.Jira.Email)
	cfg.Jira.Project = prompt(in, "Default Jira project key (optional, e.g. ENG)", cfg.Jira.Project)
	if tok, err := promptSecret("Jira API token (https://id.atlassian.com/manage-profile/security/api-tokens)"); err != nil {
		return err
	} else if tok != "" {
		if err := secrets.Set(secrets.KeyJiraToken, tok); err != nil {
			return fmt.Errorf("save jira token: %w", err)
		}
		fmt.Println(ui.OK("✓ Jira token saved to OS keyring"))
	}

	ui.Section("GitHub")
	cfg.GitHub.User = prompt(in, "GitHub username", cfg.GitHub.User)
	cfg.GitHub.DefaultRepo = prompt(in, "Default repo (owner/name, optional)", cfg.GitHub.DefaultRepo)
	if tok, err := promptSecret("GitHub token (classic or fine-grained, scopes: repo, read:org)"); err != nil {
		return err
	} else if tok != "" {
		if err := secrets.Set(secrets.KeyGitHubToken, tok); err != nil {
			return fmt.Errorf("save github token: %w", err)
		}
		fmt.Println(ui.OK("✓ GitHub token saved to OS keyring"))
	}

	ui.Section("AI / Copilot")
	cfg.AI.Provider = promptChoice(in, "AI provider for PR review", []string{"github-models", "copilot"}, def(cfg.AI.Provider, "github-models"))
	if cfg.AI.Provider == "github-models" {
		cfg.AI.Model = prompt(in, "Model id (e.g. openai/gpt-4o-mini)", def(cfg.AI.Model, "openai/gpt-4o-mini"))
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	p, _ := config.Path()
	fmt.Println(ui.OK("✓ Config saved to " + p))
	return nil
}

func runAuthStatus(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	p, _ := config.Path()
	fmt.Println("Config:", p)

	jt, _ := secrets.Get(secrets.KeyJiraToken)
	gt, _ := secrets.Get(secrets.KeyGitHubToken)

	rows := [][]string{
		{"Jira URL", cfg.Jira.BaseURL, mark(cfg.Jira.BaseURL != "")},
		{"Jira email", cfg.Jira.Email, mark(cfg.Jira.Email != "")},
		{"Jira token", maskSecret(jt), mark(jt != "")},
		{"GitHub user", cfg.GitHub.User, mark(cfg.GitHub.User != "")},
		{"GitHub default repo", cfg.GitHub.DefaultRepo, mark(cfg.GitHub.DefaultRepo != "")},
		{"GitHub token", maskSecret(gt), mark(gt != "")},
		{"AI provider", cfg.AI.Provider, mark(cfg.AI.Provider != "")},
		{"AI model", cfg.AI.Model, mark(cfg.AI.Model != "")},
	}
	ui.Table(os.Stdout, []string{"Setting", "Value", "OK"}, rows)
	return nil
}

func runAuthLogout(cmd *cobra.Command, _ []string) error {
	if err := secrets.Delete(secrets.KeyJiraToken); err != nil {
		return err
	}
	if err := secrets.Delete(secrets.KeyGitHubToken); err != nil {
		return err
	}
	fmt.Println(ui.OK("✓ Cleared Jira and GitHub tokens from keyring"))
	return nil
}

func prompt(r *bufio.Reader, label, current string) string {
	if current != "" {
		fmt.Printf("%s [%s]: ", label, current)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return current
	}
	return line
}

func promptChoice(r *bufio.Reader, label string, choices []string, current string) string {
	fmt.Printf("%s (%s) [%s]: ", label, strings.Join(choices, "/"), current)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return current
	}
	for _, c := range choices {
		if strings.EqualFold(c, line) {
			return c
		}
	}
	fmt.Println(ui.Warn("(unknown choice, keeping " + current + ")"))
	return current
}

func promptSecret(label string) (string, error) {
	fmt.Printf("%s (leave blank to skip): ", label)
	b, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func mark(ok bool) string {
	if ok {
		return ui.OK("yes")
	}
	return ui.Dim("no")
}

func maskSecret(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 6 {
		return "******"
	}
	return s[:3] + "…" + s[len(s)-3:]
}

func def(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
