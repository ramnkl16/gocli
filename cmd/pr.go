package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	gh "github.com/google/go-github/v66/github"
	"github.com/spf13/cobra"

	"github.com/yourorg/gocli/internal/config"
	"github.com/yourorg/gocli/internal/copilot"
	cursorint "github.com/yourorg/gocli/internal/cursor"
	ghint "github.com/yourorg/gocli/internal/github"
	"github.com/yourorg/gocli/internal/ui"
)

var (
	prRepoFlag         string
	prReviewComment    bool
	prReviewInline     bool
	prReviewDraft      bool
	prReviewVerdict    string
	prReviewApply      bool
	prReviewWorkdir    string
	prReviewAllowDirty bool
	prReviewNoAgent    bool
	prReviewDryRun     bool
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Pull request workflows (AI review, etc.)",
}

var prReviewCmd = &cobra.Command{
	Use:   "review <number>",
	Short: "Run an AI code review against a PR (Copilot / GitHub Models)",
	Long: `Runs an AI code review against a PR diff using GitHub Models or GitHub Copilot.

By default, the structured review is printed to stdout in markdown format.

GitHub Review Comments:
  Use --comment to submit the review directly to GitHub via the Pull Request
  Reviews API. Use --inline to attach line-level comments and GitHub suggestion
  blocks to the diff. Use --draft to save the review as a pending draft on GitHub.

Auto-Fix Mode:
  Use --apply to automatically implement the review suggestions on your local
  working tree. If the Cursor Agent CLI is available, it resolves the issues
  contextually and runs tests; otherwise, concrete code suggestion blocks are
  applied directly to the source files.`,
	Args: cobra.ExactArgs(1),
	RunE: runPRReview,
}

func init() {
	prReviewCmd.Flags().StringVarP(&prRepoFlag, "repo", "R", "", "owner/name (defaults to git remote or config)")
	prReviewCmd.Flags().BoolVar(&prReviewComment, "comment", false, "post review to GitHub PR as an official review comment")
	prReviewCmd.Flags().BoolVar(&prReviewInline, "inline", false, "when posting review, attach line-level comments to diff lines")
	prReviewCmd.Flags().BoolVar(&prReviewDraft, "draft", false, "create review as pending/draft on GitHub instead of publishing")
	prReviewCmd.Flags().StringVar(&prReviewVerdict, "verdict", "COMMENT", "review verdict when posting: COMMENT, APPROVE, or REQUEST_CHANGES")
	prReviewCmd.Flags().BoolVar(&prReviewApply, "apply", false, "automatically apply review suggestions to local working tree")
	prReviewCmd.Flags().StringVarP(&prReviewWorkdir, "workdir", "C", "", "local git clone directory (for --apply)")
	prReviewCmd.Flags().BoolVar(&prReviewAllowDirty, "allow-dirty", false, "allow dirty working tree when running --apply")
	prReviewCmd.Flags().BoolVar(&prReviewNoAgent, "no-agent", false, "disable Cursor Agent when running --apply (apply direct code blocks)")
	prReviewCmd.Flags().BoolVar(&prReviewDryRun, "dry-run", false, "preview review submission or local file modifications without applying")

	prCmd.AddCommand(prReviewCmd)
}

func runPRReview(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	c, err := ghint.New(ctx)
	if err != nil {
		return err
	}
	owner, name, err := c.ResolveRepo(prRepoFlag)
	if err != nil {
		return err
	}
	num, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("pr number: %w", err)
	}

	pr, _, err := c.PullRequests.Get(ctx, owner, name, num)
	if err != nil {
		return fmt.Errorf("get pr: %w", err)
	}

	tok, err := ghint.Token()
	if err != nil {
		return err
	}
	diff, err := fetchDiff(ctx, owner, name, num, tok)
	if err != nil {
		return fmt.Errorf("fetch diff: %w", err)
	}

	ui.Section(fmt.Sprintf("AI review of #%d %s", pr.GetNumber(), pr.GetTitle()))
	fmt.Println(ui.Dim("(diff size: " + strconv.Itoa(len(diff)) + " bytes — asking model…)"))
	fmt.Println()

	out, err := copilot.Review(ctx, copilot.ReviewInput{
		Repo:        owner + "/" + name,
		PRNumber:    num,
		Title:       pr.GetTitle(),
		Description: pr.GetBody(),
		Diff:        diff,
	})
	if err != nil {
		return err
	}
	fmt.Println(out)

	// Post review to GitHub if requested
	if prReviewComment {
		if err := handlePRReviewComment(ctx, c, owner, name, num, pr, out); err != nil {
			return err
		}
	}

	// Apply review suggestions locally if requested
	if prReviewApply {
		if err := handlePRReviewApply(ctx, num, pr, out); err != nil {
			return err
		}
	}

	return nil
}

func handlePRReviewComment(ctx context.Context, c *ghint.Client, owner, name string, num int, pr *gh.PullRequest, reviewText string) error {
	ui.Section(fmt.Sprintf("Submitting PR Review to GitHub (%s/%s #%d)", owner, name, num))

	headSHA := pr.GetHead().GetSHA()
	req := &gh.PullRequestReviewRequest{
		CommitID: gh.String(headSHA),
		Body:     gh.String(reviewText),
	}

	if !prReviewDraft {
		req.Event = gh.String(normalizeVerdict(prReviewVerdict))
	}

	var inlineSuggestions []copilot.InlineSuggestion
	if prReviewInline {
		inlineSuggestions = copilot.ParseInlineSuggestions(reviewText)
		for _, s := range inlineSuggestions {
			side := "RIGHT"
			cm := &gh.DraftReviewComment{
				Path: gh.String(s.Path),
				Side: gh.String(side),
				Body: gh.String(s.Body),
				Line: gh.Int(s.Line),
			}
			if s.StartLine > 0 && s.StartLine < s.Line {
				cm.StartSide = gh.String(side)
				cm.StartLine = gh.Int(s.StartLine)
			}
			req.Comments = append(req.Comments, cm)
		}
	}

	if prReviewDryRun {
		fmt.Println(ui.Dim(fmt.Sprintf("dry-run: would submit PR review with %d inline comment(s) (verdict: %s, draft: %v)", len(req.Comments), prReviewVerdict, prReviewDraft)))
		return nil
	}

	createdReview, _, err := c.PullRequests.CreateReview(ctx, owner, name, num, req)
	if err != nil && len(req.Comments) > 0 {
		fmt.Printf("%s Could not anchor some inline diff comments (%v). Retrying as top-level review comment...\n", ui.Warn("!"), err)
		req.Comments = nil
		createdReview, _, err = c.PullRequests.CreateReview(ctx, owner, name, num, req)
	}
	if err != nil {
		return fmt.Errorf("create pr review: %w", err)
	}

	status := "✓ Submitted PR review"
	if prReviewDraft {
		status = "✓ Created pending draft PR review"
	}
	fmt.Println(ui.OK(fmt.Sprintf("%s on #%d (%s)", status, num, createdReview.GetHTMLURL())))
	return nil
}

func handlePRReviewApply(ctx context.Context, num int, pr *gh.PullRequest, reviewText string) error {
	ui.Section("Auto-fixing review suggestions locally")

	absWorkdir, err := resolveGitWorkdir(prReviewWorkdir)
	if err != nil {
		return err
	}
	if absWorkdir == "" {
		if wd, err := os.Getwd(); err == nil {
			absWorkdir = wd
		} else {
			absWorkdir = "."
		}
	}

	if err := gitInWorkTree(absWorkdir); err != nil {
		return err
	}

	currentBranch, _ := ghint.CurrentBranch(absWorkdir)
	headRef := pr.GetHead().GetRef()
	if currentBranch != "" && headRef != "" && currentBranch != headRef {
		fmt.Printf("%s Note: local branch is %q, but PR #%d is for branch %q.\n", ui.Warn("!"), currentBranch, num, headRef)
	}

	if !prReviewAllowDirty {
		dirty, err := gitIsDirty(absWorkdir)
		if err != nil {
			return err
		}
		if dirty {
			return fmt.Errorf("working tree is dirty; commit or stash changes before running --apply (or pass --allow-dirty)")
		}
	}

	if prReviewDryRun {
		suggestions := copilot.ParseInlineSuggestions(reviewText)
		fmt.Println(ui.Dim(fmt.Sprintf("dry-run: would apply %d suggestion(s) in %s", len(suggestions), absWorkdir)))
		for _, s := range suggestions {
			fmt.Printf("  • %s:%d\n", s.Path, s.Line)
		}
		return nil
	}

	suggestions := copilot.ParseInlineSuggestions(reviewText)

	// If Cursor Agent is installed and enabled, run it for contextual, compile-checked fixes
	if cursorint.IsAvailable() && !prReviewNoAgent {
		fmt.Println(ui.Dim("Invoking Cursor Agent for contextual fixes..."))
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		agentPrompt := fmt.Sprintf("Apply the following code review feedback to Pull Request #%d (%s):\n\n%s\n\nMake the necessary code changes and verify tests pass.", num, pr.GetTitle(), reviewText)
		if err := cursorint.Run(ctx, cursorint.RunInput{
			Prompt:    agentPrompt,
			Workdir:   absWorkdir,
			Model:     cfg.Cursor.Model,
			ExtraArgs: cfg.Cursor.Args,
		}); err != nil {
			return fmt.Errorf("cursor agent fix: %w", err)
		}
		fmt.Println(ui.OK("✓ Applied fixes via Cursor Agent"))
		return nil
	}

	// Fallback to applying direct suggestion code blocks
	modifiedFiles, err := copilot.ApplySuggestionsDirectly(absWorkdir, suggestions)
	if err != nil {
		return fmt.Errorf("apply suggestions: %w", err)
	}

	if len(modifiedFiles) > 0 {
		fmt.Println(ui.OK(fmt.Sprintf("✓ Applied suggestion(s) to %d file(s): %s", len(modifiedFiles), strings.Join(modifiedFiles, ", "))))
	} else {
		fmt.Println(ui.Dim("No concrete code suggestion blocks found to apply directly. (Install Cursor Agent CLI for automated full-file fixes)"))
	}

	return nil
}

func normalizeVerdict(v string) string {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "APPROVE":
		return "APPROVE"
	case "REQUEST_CHANGES", "REQUEST-CHANGES", "REJECT":
		return "REQUEST_CHANGES"
	default:
		return "COMMENT"
	}
}

// fetchDiff retrieves a PR's unified diff via the GitHub diff media type.
func fetchDiff(ctx context.Context, owner, name string, num int, token string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d", owner, name, num)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3.diff")

	hc := &http.Client{Timeout: 30 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("github %d: %s", resp.StatusCode, string(b))
	}
	return string(b), nil
}
