package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	gh "github.com/google/go-github/v66/github"
	"github.com/spf13/cobra"

	ghint "github.com/yourorg/gocli/internal/github"
	"github.com/yourorg/gocli/internal/ui"
)

var (
	ghRepoFlag   string
	ghPRState    string
	ghPRMine     bool
	ghIssueState string
	ghLimit      int
)

var ghCmd = &cobra.Command{
	Use:   "gh",
	Short: "Work with GitHub PRs and issues",
}

var ghPRsCmd = &cobra.Command{
	Use:   "prs",
	Short: "List pull requests",
	RunE:  runGhPRs,
}

var ghPRViewCmd = &cobra.Command{
	Use:   "pr-view <number>",
	Short: "Show PR details (title, author, state, checks summary)",
	Args:  cobra.ExactArgs(1),
	RunE:  runGhPRView,
}

var ghPRCheckoutCmd = &cobra.Command{
	Use:   "pr-checkout <number>",
	Short: "Checkout a PR locally (delegates to gh CLI)",
	Args:  cobra.ExactArgs(1),
	RunE:  runGhPRCheckout,
}

var ghIssuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "List GitHub issues",
	RunE:  runGhIssues,
}

var (
	ghPRCreateTitle string
	ghPRCreateBody  string
	ghPRCreateBase  string
	ghPRCreateDraft bool
)

var ghPRCreateCmd = &cobra.Command{
	Use:   "pr-create",
	Short: "Create a pull request from the current branch",
	Long: `Create a pull request from the current branch to the base branch.

The head branch is detected from your current git branch. You must push
the branch to GitHub before creating the PR.

Examples:
  gocli gh pr-create --title "Fix login bug" --base main
  gocli gh pr-create -t "Add feature" -b "Implements new feature" --base develop --draft`,
	RunE: runGhPRCreate,
}

func init() {
	for _, c := range []*cobra.Command{ghPRsCmd, ghPRViewCmd, ghPRCheckoutCmd, ghIssuesCmd} {
		c.Flags().StringVarP(&ghRepoFlag, "repo", "R", "", "owner/name (defaults to git remote or config)")
	}
	ghPRsCmd.Flags().StringVar(&ghPRState, "state", "open", "open|closed|all")
	ghPRsCmd.Flags().BoolVar(&ghPRMine, "mine", false, "only PRs authored by you")
	ghPRsCmd.Flags().IntVarP(&ghLimit, "limit", "n", 30, "max PRs to return")

	ghIssuesCmd.Flags().StringVar(&ghIssueState, "state", "open", "open|closed|all")
	ghIssuesCmd.Flags().IntVarP(&ghLimit, "limit", "n", 30, "max issues to return")

	ghPRCreateCmd.Flags().StringVarP(&ghRepoFlag, "repo", "R", "", "owner/name (defaults to git remote or config)")
	ghPRCreateCmd.Flags().StringVarP(&ghPRCreateTitle, "title", "t", "", "PR title (required)")
	ghPRCreateCmd.Flags().StringVarP(&ghPRCreateBody, "body", "b", "", "PR body/description")
	ghPRCreateCmd.Flags().StringVar(&ghPRCreateBase, "base", "main", "base branch to merge into")
	ghPRCreateCmd.Flags().BoolVar(&ghPRCreateDraft, "draft", false, "create as draft PR")
	ghPRCreateCmd.MarkFlagRequired("title")

	ghCmd.AddCommand(ghPRsCmd, ghPRViewCmd, ghPRCheckoutCmd, ghIssuesCmd, ghPRCreateCmd)
}

func runGhPRs(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()
	c, err := ghint.New(ctx)
	if err != nil {
		return err
	}
	owner, name, err := c.ResolveRepo(ghRepoFlag)
	if err != nil {
		return err
	}

	prs, _, err := c.PullRequests.List(ctx, owner, name, &gh.PullRequestListOptions{
		State:       ghPRState,
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: gh.ListOptions{PerPage: ghLimit},
	})
	if err != nil {
		return fmt.Errorf("list prs: %w", err)
	}

	ui.Section(fmt.Sprintf("Pull requests in %s/%s (%s)", owner, name, ghPRState))
	rows := make([][]string, 0, len(prs))
	for _, p := range prs {
		if ghPRMine && c.User != "" && (p.User == nil || p.User.GetLogin() != c.User) {
			continue
		}
		rows = append(rows, []string{
			"#" + strconv.Itoa(p.GetNumber()),
			p.GetUser().GetLogin(),
			p.GetState(),
			ui.Truncate(p.GetTitle(), 70),
		})
	}
	ui.Table(os.Stdout, []string{"PR", "AUTHOR", "STATE", "TITLE"}, rows)
	return nil
}

func runGhPRView(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	c, err := ghint.New(ctx)
	if err != nil {
		return err
	}
	owner, name, err := c.ResolveRepo(ghRepoFlag)
	if err != nil {
		return err
	}
	num, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("pr number: %w", err)
	}
	p, _, err := c.PullRequests.Get(ctx, owner, name, num)
	if err != nil {
		return fmt.Errorf("get pr: %w", err)
	}

	ui.Section(fmt.Sprintf("#%d %s", p.GetNumber(), p.GetTitle()))
	row := func(k, v string) { fmt.Printf("  %-12s %s\n", ui.Dim(k+":"), v) }
	row("Author", p.GetUser().GetLogin())
	row("State", p.GetState())
	row("Branch", fmt.Sprintf("%s ← %s", p.GetBase().GetRef(), p.GetHead().GetRef()))
	row("Mergeable", fmt.Sprintf("%v", p.GetMergeable()))
	row("URL", p.GetHTMLURL())
	row("Changes", fmt.Sprintf("+%d / -%d in %d files",
		p.GetAdditions(), p.GetDeletions(), p.GetChangedFiles()))

	if sha := p.GetHead().GetSHA(); sha != "" {
		cs, _, err := c.Checks.ListCheckRunsForRef(ctx, owner, name, sha, nil)
		if err == nil && cs != nil && cs.GetTotal() > 0 {
			fmt.Println()
			fmt.Println(ui.Title("Checks"))
			rows := make([][]string, 0, cs.GetTotal())
			for _, r := range cs.CheckRuns {
				rows = append(rows, []string{r.GetName(), r.GetStatus(), r.GetConclusion()})
			}
			ui.Table(os.Stdout, []string{"NAME", "STATUS", "CONCLUSION"}, rows)
		}
	}

	if body := p.GetBody(); body != "" {
		fmt.Println()
		fmt.Println(body)
	}
	return nil
}

func runGhPRCheckout(cmd *cobra.Command, args []string) error {
	c := exec.Command("gh", "pr", "checkout", args[0])
	c.Stdout, c.Stderr, c.Stdin = os.Stdout, os.Stderr, os.Stdin
	return c.Run()
}

func runGhPRCreate(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()
	c, err := ghint.New(ctx)
	if err != nil {
		return err
	}
	owner, name, err := c.ResolveRepo(ghRepoFlag)
	if err != nil {
		return err
	}

	// Get current branch name
	headBranch, err := getCurrentBranch()
	if err != nil {
		return fmt.Errorf("get current branch: %w", err)
	}

	ui.Section(fmt.Sprintf("Creating PR in %s/%s", owner, name))
	fmt.Printf("  %s %s\n", ui.Dim("Head:"), headBranch)
	fmt.Printf("  %s %s\n", ui.Dim("Base:"), ghPRCreateBase)
	fmt.Printf("  %s %s\n", ui.Dim("Title:"), ghPRCreateTitle)
	if ghPRCreateDraft {
		fmt.Printf("  %s %s\n", ui.Dim("Draft:"), "yes")
	}
	fmt.Println()

	newPR := &gh.NewPullRequest{
		Title: gh.String(ghPRCreateTitle),
		Head:  gh.String(headBranch),
		Base:  gh.String(ghPRCreateBase),
		Body:  gh.String(ghPRCreateBody),
		Draft: gh.Bool(ghPRCreateDraft),
	}

	pr, resp, err := c.PullRequests.Create(ctx, owner, name, newPR)
	if err != nil {
		if resp != nil && resp.StatusCode == 403 {
			return fmt.Errorf("403 Forbidden - check your GitHub token has 'repo' scope and is valid")
		}
		return fmt.Errorf("create pr: %w", err)
	}

	fmt.Println(ui.OK(fmt.Sprintf("✓ Created PR #%d", pr.GetNumber())))
	fmt.Printf("  %s %s\n", ui.Dim("URL:"), pr.GetHTMLURL())
	return nil
}

func getCurrentBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return "", fmt.Errorf("could not determine current branch")
	}
	return branch, nil
}

func runGhIssues(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()
	c, err := ghint.New(ctx)
	if err != nil {
		return err
	}
	owner, name, err := c.ResolveRepo(ghRepoFlag)
	if err != nil {
		return err
	}
	issues, _, err := c.Issues.ListByRepo(ctx, owner, name, &gh.IssueListByRepoOptions{
		State:       ghIssueState,
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: gh.ListOptions{PerPage: ghLimit},
	})
	if err != nil {
		return fmt.Errorf("list issues: %w", err)
	}
	ui.Section(fmt.Sprintf("Issues in %s/%s (%s)", owner, name, ghIssueState))
	rows := make([][]string, 0, len(issues))
	for _, is := range issues {
		if is.IsPullRequest() {
			continue
		}
		rows = append(rows, []string{
			"#" + strconv.Itoa(is.GetNumber()),
			is.GetUser().GetLogin(),
			is.GetState(),
			ui.Truncate(is.GetTitle(), 70),
		})
	}
	ui.Table(os.Stdout, []string{"ISSUE", "AUTHOR", "STATE", "TITLE"}, rows)
	return nil
}
