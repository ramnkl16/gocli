package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/yourorg/gocli/internal/copilot"
	ghint "github.com/yourorg/gocli/internal/github"
	"github.com/yourorg/gocli/internal/ui"
)

var (
	prRepoFlag string
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Pull request workflows (AI review, etc.)",
}

var prReviewCmd = &cobra.Command{
	Use:   "review <number>",
	Short: "Run an AI code review against a PR (Copilot / GitHub Models)",
	Args:  cobra.ExactArgs(1),
	RunE:  runPRReview,
}

func init() {
	prReviewCmd.Flags().StringVarP(&prRepoFlag, "repo", "R", "", "owner/name (defaults to git remote or config)")
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
	return nil
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
