package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	jiraint "github.com/yourorg/gocli/internal/jira"
	"github.com/yourorg/gocli/internal/ui"
)

var (
	jiraJQL      string
	jiraLimit    int
	jiraMine     bool
	jiraStatus   string
	jiraAssignee string
)

var jiraCmd = &cobra.Command{
	Use:   "jira",
	Short: "Work with Jira issues",
}

var jiraListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues (defaults to your open issues)",
	RunE:  runJiraList,
}

var jiraViewCmd = &cobra.Command{
	Use:   "view <ISSUE-KEY>",
	Short: "Show details for a single issue",
	Args:  cobra.ExactArgs(1),
	RunE:  runJiraView,
}

var jiraOpenCmd = &cobra.Command{
	Use:   "open <ISSUE-KEY>",
	Short: "Open an issue in your browser",
	Args:  cobra.ExactArgs(1),
	RunE:  runJiraOpen,
}

var jiraTransitionCmd = &cobra.Command{
	Use:   "transition <ISSUE-KEY> <STATUS>",
	Short: "Move an issue to a new status (e.g. 'In Progress', 'Done')",
	Args:  cobra.ExactArgs(2),
	RunE:  runJiraTransition,
}

var (
	jiraBranchType         string
	jiraBranchToStatus     string
	jiraBranchWorkdir      string
	jiraBranchAllowDirty   bool
	jiraBranchNoTransition bool
	jiraBranchDryRun       bool
)

var jiraBranchCmd = &cobra.Command{
	Use:   "branch <ISSUE-KEY>",
	Short: "Create a git branch from ticket, set Jira to In Progress, and checkout",
	Long: `Builds a branch name: {bug|fea|chg}-{KEY}-{summary-slug}

Summary slug is the first 20 characters of the issue title, lowercased, with
spaces and punctuation normalized to single hyphens. Then transitions the
ticket to the configured status (default "In Progress") and runs
git checkout -b. By default the git repository is the current directory; use
--workdir to point at another clone (same as git -C <path>).`,
	Args: cobra.ExactArgs(1),
	RunE: runJiraBranch,
}

func init() {
	jiraListCmd.Flags().StringVar(&jiraJQL, "jql", "", "raw JQL (overrides --mine / --status / --assignee)")
	jiraListCmd.Flags().BoolVar(&jiraMine, "mine", true, "only issues assigned to me (--assignee disables this)")
	jiraListCmd.Flags().StringVarP(&jiraAssignee, "assignee", "u", "", `filter by assignee: Jira id/email, or key from jira.assignee_aliases (see config)`)
	jiraListCmd.Flags().StringVar(&jiraStatus, "status", "", "filter by status (e.g. 'In Progress')")
	jiraListCmd.Flags().IntVarP(&jiraLimit, "limit", "n", 25, "max issues to return")

	jiraBranchCmd.Flags().StringVar(&jiraBranchType, "type", "", "ticket type prefix: bug | fea | chg (required)")
	jiraBranchCmd.Flags().StringVar(&jiraBranchToStatus, "to-status", "In Progress", "Jira status to transition to after creating the branch")
	jiraBranchCmd.Flags().StringVarP(&jiraBranchWorkdir, "workdir", "C", "", "path to the local git repository (default: current directory; same as `git -C`")
	jiraBranchCmd.Flags().BoolVar(&jiraBranchAllowDirty, "allow-dirty", false, "allow a dirty git working tree")
	jiraBranchCmd.Flags().BoolVar(&jiraBranchNoTransition, "no-transition", false, "only create and checkout the branch; do not change Jira")
	jiraBranchCmd.Flags().BoolVar(&jiraBranchDryRun, "dry-run", false, "print branch name and actions only")

	jiraCmd.AddCommand(jiraListCmd, jiraViewCmd, jiraOpenCmd, jiraTransitionCmd, jiraBranchCmd)
}

func runJiraList(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()
	c, err := jiraint.New(ctx)
	if err != nil {
		return err
	}

	jql := jiraJQL
	if jql == "" {
		var parts []string
		if trimmed := strings.TrimSpace(jiraAssignee); trimmed != "" {
			resolved, err := jiraint.ResolveAssignee(c.AssigneeAliases, trimmed)
			if err != nil {
				return err
			}
			parts = append(parts, fmt.Sprintf("assignee = %q", resolved))
		} else if jiraMine {
			parts = append(parts, "assignee = currentUser()")
		}
		if jiraStatus != "" {
			parts = append(parts, fmt.Sprintf("status = %q", jiraStatus))
		} else {
			parts = append(parts, "statusCategory != Done")
		}
		parts = append(parts, "ORDER BY updated DESC")
		jql = strings.Join(parts, " AND ")
		jql = strings.Replace(jql, " AND ORDER BY", " ORDER BY", 1)
	}

	// Jira Cloud deprecated GET /rest/api/2|3/search. Use GET /rest/api/3/search/jql
	// (cursor + JQL) — see https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issue-search/
	fields := []string{
		"summary", "status", "assignee", "priority", "updated",
	}
	sprintF, err := c.ResolvedSprintFieldID(ctx)
	if err != nil {
		return fmt.Errorf("resolve sprint field: %w", err)
	}
	if sprintF != "" {
		fields = append(fields, sprintF)
	}
	issues, err := c.SearchIssuesJQL(ctx, jql, jiraLimit, fields)
	if err != nil {
		return fmt.Errorf("jql search: %w", err)
	}

	ui.Section(fmt.Sprintf("Jira issues (%d) — %s", len(issues), jql))
	base := strings.TrimRight(c.BaseURL, "/")
	rows := make([][]string, 0, len(issues))
	for _, is := range issues {
		issueURL := fmt.Sprintf("%s/browse/%s", base, is.Key)
		keyCol := ui.Hyperlink(issueURL, is.Key)
		assignee := "—"
		if is.Fields != nil && is.Fields.Assignee != nil {
			assignee = is.Fields.Assignee.DisplayName
		}
		status := "—"
		if is.Fields != nil && is.Fields.Status != nil {
			status = is.Fields.Status.Name
		}
		sprint := "—"
		if is.Fields != nil {
			sprint = ui.Truncate(jiraint.FormatSprintColumn(is.Fields), 28)
		}
		summary := ""
		if is.Fields != nil {
			summary = ui.Truncate(is.Fields.Summary, 70)
		}
		rows = append(rows, []string{keyCol, status, sprint, assignee, summary})
	}
	ui.Table(os.Stdout, []string{"KEY", "STATUS", "SPRINT", "ASSIGNEE", "SUMMARY"}, rows)
	return nil
}

func runJiraView(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	c, err := jiraint.New(ctx)
	if err != nil {
		return err
	}
	key := args[0]
	is, _, err := c.Issue.Get(ctx, key, nil)
	if err != nil {
		return fmt.Errorf("get %s: %w", key, err)
	}

	ui.Section(is.Key + " — " + is.Fields.Summary)
	row := func(k, v string) { fmt.Printf("  %-10s %s\n", ui.Dim(k+":"), v) }
	row("Status", is.Fields.Status.Name)
	if is.Fields.Assignee != nil {
		row("Assignee", is.Fields.Assignee.DisplayName)
	}
	if is.Fields.Reporter != nil {
		row("Reporter", is.Fields.Reporter.DisplayName)
	}
	if is.Fields.Priority != nil {
		row("Priority", is.Fields.Priority.Name)
	}
	row("URL", fmt.Sprintf("%s/browse/%s", strings.TrimRight(c.BaseURL, "/"), is.Key))
	if is.Fields.Description != "" {
		fmt.Println()
		fmt.Println(is.Fields.Description)
	}
	return nil
}

func runJiraOpen(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	c, err := jiraint.New(ctx)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/browse/%s", strings.TrimRight(c.BaseURL, "/"), args[0])
	return openBrowser(url)
}

func runJiraTransition(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	c, err := jiraint.New(ctx)
	if err != nil {
		return err
	}
	key, target := args[0], args[1]
	to, err := c.TransitionTo(ctx, key, target)
	if err != nil {
		return err
	}
	fmt.Println(ui.OK(fmt.Sprintf("✓ %s → %s", key, to)))
	return nil
}

func runJiraBranch(cmd *cobra.Command, args []string) error {
	typ := strings.ToLower(strings.TrimSpace(jiraBranchType))
	if typ == "" {
		return fmt.Errorf("required flag: --type (bug|fea|chg)")
	}
	if typ != "bug" && typ != "fea" && typ != "chg" {
		return fmt.Errorf(`--type must be one of: bug, fea, chg, got %q`, typ)
	}
	key := strings.TrimSpace(args[0])

	ctx := context.Background()
	c, err := jiraint.New(ctx)
	if err != nil {
		return err
	}
	is, _, err := c.Issue.Get(ctx, key, nil)
	if err != nil {
		return fmt.Errorf("get %s: %w", key, err)
	}
	summary := ""
	if is.Fields != nil {
		summary = is.Fields.Summary
	}
	branch := jiraint.BranchNameForWorkflow(typ, is.Key, summary)
	repoDir, err := resolveGitWorkdir(jiraBranchWorkdir)
	if err != nil {
		return err
	}
	repoDisplay := repoDir
	if repoDisplay == "" {
		if wd, err := os.Getwd(); err == nil {
			repoDisplay = wd
		} else {
			repoDisplay = "."
		}
	}
	if jiraBranchDryRun {
		fmt.Println(ui.Dim("dry-run: would run: git -C " + repoDisplay + " checkout -b " + branch))
		if !jiraBranchNoTransition {
			fmt.Println(ui.Dim("dry-run: would transition " + is.Key + ` to "` + jiraBranchToStatus + `"`))
		}
		fmt.Println(branch)
		return nil
	}

	if err := gitInWorkTree(repoDir); err != nil {
		return err
	}
	if !jiraBranchAllowDirty {
		if dirty, err := gitIsDirty(repoDir); err != nil {
			return err
		} else if dirty {
			return fmt.Errorf("working tree is dirty; commit/stash or pass --allow-dirty")
		}
	}
	exists, err := localGitBranchExists(repoDir, branch)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("branch %q already exists", branch)
	}

	if !jiraBranchNoTransition {
		to, err := c.TransitionTo(ctx, is.Key, jiraBranchToStatus)
		if err != nil {
			return err
		}
		fmt.Println(ui.OK("✓ Jira " + is.Key + " → " + to))
	}

	xc := gitCmd(repoDir, "checkout", "-b", branch)
	xc.Stdout, xc.Stderr, xc.Stdin = os.Stdout, os.Stderr, os.Stdin
	if err := xc.Run(); err != nil {
		return fmt.Errorf("git checkout -b: %w", err)
	}
	fmt.Println(ui.OK("✓ checked out " + branch + " in " + repoDisplay))
	return nil
}

// resolveGitWorkdir returns an absolute path to the repo root, or "" for "current directory" (git default).
func resolveGitWorkdir(flag string) (string, error) {
	flag = strings.TrimSpace(flag)
	if flag == "" {
		return "", nil
	}
	abs, err := filepath.Abs(flag)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("workdir: %w", err)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("workdir is not a directory: %s", abs)
	}
	return abs, nil
}

func gitInWorkTree(dir string) error {
	xc := gitCmd(dir, "rev-parse", "--is-inside-work-tree")
	xc.Stderr = io.Discard
	out, err := xc.Output()
	if err != nil {
		prefix := "not a git repository"
		if dir != "" {
			prefix = "not a git repository at " + dir
		}
		return fmt.Errorf("%s (or git is not in PATH): %w", prefix, err)
	}
	if strings.TrimSpace(string(out)) != "true" {
		return fmt.Errorf("not a git work tree")
	}
	return nil
}

func gitIsDirty(dir string) (bool, error) {
	xc := gitCmd(dir, "status", "--porcelain")
	xc.Stderr = io.Discard
	out, err := xc.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}

func localGitBranchExists(dir, name string) (bool, error) {
	xc := gitCmd(dir, "rev-parse", "--verify", "refs/heads/"+name)
	xc.Stderr = io.Discard
	if err := xc.Run(); err != nil {
		var ex *exec.ExitError
		if errors.As(err, &ex) {
			// ref does not exist
			if ex.ExitCode() == 128 || ex.ExitCode() == 1 {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}

// gitCmd runs a git subcommand. If dir is non-empty, runs as `git -C <dir> <name> <arg>...`.
func gitCmd(dir, name string, arg ...string) *exec.Cmd {
	argv := []string{"git"}
	if dir != "" {
		argv = append(argv, "-C", dir)
	}
	argv = append(argv, name)
	argv = append(argv, arg...)
	return exec.Command(argv[0], argv[1:]...)
}

func openBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	c := exec.Command(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Start()
}
