package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	gojira "github.com/andygrunwald/go-jira/v2/cloud"
	gh "github.com/google/go-github/v66/github"
	"github.com/spf13/cobra"

	"github.com/yourorg/gocli/internal/copilot"
	ghint "github.com/yourorg/gocli/internal/github"
	jiraint "github.com/yourorg/gocli/internal/jira"
	"github.com/yourorg/gocli/internal/ui"
)

var (
	prCreateRepo           string
	prCreateTitle          string
	prCreateBody           string
	prCreateBodyFile       string
	prCreateBase           string
	prCreateHead           string
	prCreateDraft          bool
	prCreateWorkdir        string
	prCreateJiraKey        string
	prCreateNoJira         bool
	prCreateTransition     string
	prCreateNoComment      bool
	prCreateCommentMessage string
	prCreateAI             bool
)

var prCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Open a new pull request on GitHub",
	Long: `Creates a pull request via the GitHub API.

Repository: --repo owner/name, else origin in the current dir (or --workdir),
else default_repo from config.

--head defaults to the current git branch; --base defaults to the repo's
default branch on GitHub (usually main). The head branch must already be
pushed. For a PR from a fork, use --head forkowner:branch.

Use --ai to automatically analyze git diff, commit history, and Jira issue
details to generate a concise conventional-commit PR title and structured description.

If your branch name contains a Jira issue key (e.g. fea-KAN-224-auth) or you
pass --jira <KEY>, gocli automatically links the PR to Jira:
  - References the Jira ticket in the PR description
  - Posts an automated comment back to the Jira ticket with the PR link
  - Optionally transitions the ticket to a new status via --transition`,
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
	prCreateCmd.Flags().StringVar(&prCreateJiraKey, "jira", "", "Jira issue key to link (default: auto-detected from branch)")
	prCreateCmd.Flags().BoolVar(&prCreateNoJira, "no-jira", false, "disable Jira linking, comment, and transitions")
	prCreateCmd.Flags().StringVar(&prCreateTransition, "transition", "", "transition Jira issue to this status (e.g. 'In Review')")
	prCreateCmd.Flags().BoolVar(&prCreateNoComment, "no-comment", false, "do not post a comment to Jira with the PR link")
	prCreateCmd.Flags().StringVar(&prCreateCommentMessage, "comment-message", "", "custom comment for Jira (default: 'Opened PR #<N>: <URL>')")
	prCreateCmd.Flags().BoolVar(&prCreateAI, "ai", false, "generate PR title and description using AI from diff and Jira issue")

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

	head := strings.TrimSpace(prCreateHead)
	if head == "" {
		hb, err := ghint.CurrentBranch(prCreateWorkdir)
		if err != nil {
			return fmt.Errorf("set --head or run inside a git repo: %w", err)
		}
		head = hb
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

	// Detect and resolve Jira issue if enabled
	var jiraKey string
	if !prCreateNoJira {
		if trimmed := strings.TrimSpace(prCreateJiraKey); trimmed != "" {
			jiraKey = strings.ToUpper(trimmed)
		} else {
			jiraKey = jiraint.ExtractIssueKey(head)
		}
	}

	var jiraClient *jiraint.Client
	var jiraIssue *gojira.Issue
	var jiraIssueURL string
	if jiraKey != "" {
		jc, err := jiraint.New(ctx)
		if err != nil {
			if prCreateJiraKey != "" {
				return fmt.Errorf("jira client: %w", err)
			}
		} else {
			jiraClient = jc
			jiraIssueURL = fmt.Sprintf("%s/browse/%s", strings.TrimRight(jc.BaseURL, "/"), jiraKey)
			is, _, err := jc.Issue.Get(ctx, jiraKey, nil)
			if err != nil {
				if prCreateJiraKey != "" {
					return fmt.Errorf("get jira issue %s: %w", jiraKey, err)
				}
			} else {
				jiraIssue = is
			}
		}
	}

	title := strings.TrimSpace(prCreateTitle)
	body, err := prCreateReadBody()
	if err != nil {
		return err
	}

	if prCreateAI && (title == "" || strings.TrimSpace(body) == "") {
		ui.Section("Generating PR with AI")
		fmt.Println(ui.Dim("Analyzing branch diff and commits…"))

		diff, _ := ghint.BranchDiff(prCreateWorkdir, base, head)
		commits, _ := ghint.CommitLog(prCreateWorkdir, base, head)

		// Fallback to GitHub Compare API if local diff is empty
		if strings.TrimSpace(diff) == "" {
			comp, _, compErr := c.Repositories.CompareCommits(ctx, owner, name, base, head, nil)
			if compErr == nil && comp != nil {
				var diffBuf strings.Builder
				for _, f := range comp.Files {
					if f != nil && f.GetPatch() != "" {
						fmt.Fprintf(&diffBuf, "diff --git a/%s b/%s\n%s\n", f.GetFilename(), f.GetFilename(), f.GetPatch())
					}
				}
				diff = diffBuf.String()

				if commits == "" {
					var commitBuf strings.Builder
					for _, cm := range comp.Commits {
						if cm != nil && cm.GetCommit() != nil {
							msg := strings.Split(cm.GetCommit().GetMessage(), "\n")[0]
							sha := cm.GetSHA()
							if len(sha) > 7 {
								sha = sha[:7]
							}
							fmt.Fprintf(&commitBuf, "%s %s\n", sha, msg)
						}
					}
					commits = commitBuf.String()
				}
			}
		}

		var jiraSummary, jiraDesc string
		if jiraIssue != nil && jiraIssue.Fields != nil {
			jiraSummary = strings.TrimSpace(jiraIssue.Fields.Summary)
			jiraDesc = strings.TrimSpace(jiraIssue.Fields.Description)
		}

		gen, err := copilot.GeneratePR(ctx, copilot.GeneratePRInput{
			Repo:        owner + "/" + name,
			Base:        base,
			Head:        head,
			Diff:        diff,
			Commits:     commits,
			JiraKey:     jiraKey,
			JiraSummary: jiraSummary,
			JiraDesc:    jiraDesc,
		}, jiraIssueURL)
		if err != nil {
			return fmt.Errorf("ai generate pr: %w", err)
		}

		if title == "" && gen.Title != "" {
			title = gen.Title
		}
		if strings.TrimSpace(body) == "" && gen.Body != "" {
			body = gen.Body
		}
		fmt.Println(ui.OK("✓ AI generated PR title & description"))
	}

	if title == "" {
		subj, err := ghint.LastCommitSubject(prCreateWorkdir)
		if err == nil && strings.TrimSpace(subj) != "" {
			title = strings.TrimSpace(subj)
		} else if jiraIssue != nil && jiraIssue.Fields != nil && jiraIssue.Fields.Summary != "" {
			title = fmt.Sprintf("%s: %s", jiraKey, strings.TrimSpace(jiraIssue.Fields.Summary))
		}
		if title == "" {
			if err != nil {
				return fmt.Errorf("set --title or run inside a git repo: %w", err)
			}
			return fmt.Errorf("empty --title and could not read last commit or Jira issue")
		}
	}

	if jiraKey != "" && jiraIssueURL != "" {
		if strings.TrimSpace(body) == "" {
			var b strings.Builder
			fmt.Fprintf(&b, "## Jira Ticket\n- [%s](%s)", jiraKey, jiraIssueURL)
			if jiraIssue != nil && jiraIssue.Fields != nil && jiraIssue.Fields.Summary != "" {
				fmt.Fprintf(&b, ": %s\n", strings.TrimSpace(jiraIssue.Fields.Summary))
			} else {
				b.WriteString("\n")
			}
			if jiraIssue != nil && jiraIssue.Fields != nil && jiraIssue.Fields.Description != "" {
				fmt.Fprintf(&b, "\n## Jira Description\n%s\n", strings.TrimSpace(jiraIssue.Fields.Description))
			}
			body = b.String()
		} else if !strings.Contains(body, jiraIssueURL) && !strings.Contains(body, jiraKey) {
			body = fmt.Sprintf("**Jira:** [%s](%s)\n\n%s", jiraKey, jiraIssueURL, body)
		}
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

	fmt.Println(ui.OK(fmt.Sprintf("✓ Opened pull request #%d: %s", p.GetNumber(), title)))
	fmt.Println(p.GetHTMLURL())

	if jiraClient != nil && jiraKey != "" {
		if !prCreateNoComment {
			commentMsg := strings.TrimSpace(prCreateCommentMessage)
			if commentMsg == "" {
				commentMsg = fmt.Sprintf("Opened GitHub pull request #%d:\n%s", p.GetNumber(), p.GetHTMLURL())
			}
			if _, err := jiraClient.AddComment(ctx, jiraKey, commentMsg); err != nil {
				fmt.Printf("%s Could not post comment to Jira %s: %v\n", ui.Warn("!"), jiraKey, err)
			} else {
				fmt.Println(ui.OK(fmt.Sprintf("✓ Added comment to Jira %s", jiraKey)))
			}
		}

		if prCreateTransition != "" {
			to, err := jiraClient.TransitionTo(ctx, jiraKey, prCreateTransition)
			if err != nil {
				fmt.Printf("%s Could not transition Jira %s to %q: %v\n", ui.Warn("!"), jiraKey, prCreateTransition, err)
			} else {
				fmt.Println(ui.OK(fmt.Sprintf("✓ Jira %s → %s", jiraKey, to)))
			}
		}
	}

	return nil
}

func enhancePRCreateError(err error) error {
	var er *gh.ErrorResponse
	if errors.As(err, &er) {
		if er.Response != nil && er.Response.StatusCode == 403 {
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
		if er.Response != nil && er.Response.StatusCode == 422 {
			for _, e := range er.Errors {
				if e.Field == "head" {
					return fmt.Errorf("%w\n\nGitHub cannot find the head branch on remote repository.\nFix: push your branch to GitHub before creating a pull request:\n  git push -u origin <branch-name>\n", err)
				}
			}
		}
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
