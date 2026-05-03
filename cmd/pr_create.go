package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	gh "github.com/google/go-github/v66/github"
	"github.com/spf13/cobra"

	ghint "github.com/yourorg/gocli/internal/github"
	"github.com/yourorg/gocli/internal/ui"
)

var (
	prCreateRepo     string
	prCreateTitle    string
	prCreateBody     string
	prCreateBodyFile string
	prCreateBase     string
	prCreateHead     string
	prCreateDraft    bool
	prCreateWorkdir  string
)

var prCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Open a new pull request on GitHub",
	Long: `Creates a pull request via the GitHub API.

Repository: --repo owner/name, else origin in the current dir (or --workdir),
else default_repo from config.

--head defaults to the current git branch; --base defaults to the repo's
default branch on GitHub (usually main). The head branch must already be
pushed. For a PR from a fork, use --head forkowner:branch.`,
	Args: cobra.NoArgs,
	RunE: runPRCreate,
}

func init() {
	prCreateCmd.Flags().StringVarP(&prCreateRepo, "repo", "R", "", "owner/name (defaults to git remote or config)")
	prCreateCmd.Flags().StringVar(&prCreateTitle, "title", "", "PR title (default: subject of last commit)")
	prCreateCmd.Flags().StringVar(&prCreateBody, "body", "", "PR description (markdown)")
	prCreateCmd.Flags().StringVar(&prCreateBodyFile, "body-file", "", "read description from file (overrides --body)")
	prCreateCmd.Flags().StringVar(&prCreateBase, "base", "", "branch to merge into (default: repo default branch)")
	prCreateCmd.Flags().StringVar(&prCreateHead, "head", "", "branch with your changes (default: current git branch)")
	prCreateCmd.Flags().BoolVar(&prCreateDraft, "draft", false, "open as draft pull request")
	prCreateCmd.Flags().StringVarP(&prCreateWorkdir, "workdir", "C", "", "local git clone for remote/branch resolution (use with --head default)")

	prCmd.AddCommand(prCreateCmd)
}

func runPRCreate(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()
	c, err := ghint.New(ctx)
	if err != nil {
		return err
	}
	owner, name, err := c.ResolveRepoInDir(prCreateRepo, prCreateWorkdir)
	if err != nil {
		return err
	}

	title := strings.TrimSpace(prCreateTitle)
	if title == "" {
		subj, err := ghint.LastCommitSubject(prCreateWorkdir)
		if err != nil {
			return fmt.Errorf("set --title or run inside a git repo: %w", err)
		}
		title = subj
		if title == "" {
			return fmt.Errorf("empty --title and could not read last commit")
		}
	}

	body, err := prCreateReadBody()
	if err != nil {
		return err
	}

	base := strings.TrimSpace(prCreateBase)
	if base == "" {
		rep, _, err := c.Repositories.Get(ctx, owner, name)
		if err != nil {
			return fmt.Errorf("get repo: %w", err)
		}
		base = rep.GetDefaultBranch()
		if base == "" {
			return fmt.Errorf("repo has no default branch; pass --base explicitly")
		}
	}

	head := strings.TrimSpace(prCreateHead)
	if head == "" {
		hb, err := ghint.CurrentBranch(prCreateWorkdir)
		if err != nil {
			return fmt.Errorf("set --head or run inside a git repo: %w", err)
		}
		head = hb
	}

	newPR := &gh.NewPullRequest{
		Title: gh.String(title),
		Head:  gh.String(head),
		Base:  gh.String(base),
	}
	if body != "" {
		newPR.Body = gh.String(body)
	}
	if prCreateDraft {
		newPR.Draft = gh.Bool(true)
	}

	p, _, err := c.PullRequests.Create(ctx, owner, name, newPR)
	if err != nil {
		return enhancePRCreateError(err)
	}

	fmt.Println(ui.OK(fmt.Sprintf("✓ Opened pull request #%d", p.GetNumber())))
	fmt.Println(p.GetHTMLURL())
	return nil
}

func enhancePRCreateError(err error) error {
	var er *gh.ErrorResponse
	if errors.As(err, &er) && er.Response != nil && er.Response.StatusCode == 403 {
		repo := "this repository"
		if er.Response.Request != nil && er.Response.Request.URL != nil {
			repo = repoFromGitHubAPIPath(er.Response.Request.URL.Path)
		}
		return fmt.Errorf("%w\n\nGitHub returned 403: your token cannot open a pull request on %s. Fix:\n"+
			"  • Classic PAT: enable scope 'repo' (full), or for public-only repos you can try 'public_repo' (narrower).\n"+
			"  • Fine-grained PAT: add repository access for this repo, then set Pull requests: Read and write, Contents: Read or Read and write.\n"+
			"  • If the repo is under an org with SSO: fine-grained token row → Configure SSO and authorize the org.\n"+
			"  • The branch in --head must already exist on GitHub (run git push first).\n", err, repo)
	}
	if err != nil && strings.Contains(err.Error(), "not accessible by personal access token") {
		return fmt.Errorf("%w\n\n(see: classic PAT with 'repo' scope, or fine-grained: Pull requests + Contents on this repository; push your branch first)", err)
	}
	return err
}

// repoFromGitHubAPIPath turns "/repos/owner/name/pulls" into "owner/name" for user-facing text.
func repoFromGitHubAPIPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 && parts[0] == "repos" {
		return parts[1] + "/" + parts[2]
	}
	if path != "" {
		return path
	}
	return "this repository"
}

func prCreateReadBody() (string, error) {
	if prCreateBodyFile != "" {
		b, err := os.ReadFile(prCreateBodyFile)
		if err != nil {
			return "", fmt.Errorf("read --body-file: %w", err)
		}
		return string(b), nil
	}
	return prCreateBody, nil
}
