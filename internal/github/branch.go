package github

import (
	"fmt"
	"os/exec"
	"strings"
)

// CurrentBranch returns the current branch name (HEAD). If workdir is non-empty,
// runs git -C workdir.
func CurrentBranch(workdir string) (string, error) {
	var cmd *exec.Cmd
	if strings.TrimSpace(workdir) == "" {
		cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	} else {
		cmd = exec.Command("git", "-C", workdir, "rev-parse", "--abbrev-ref", "HEAD")
	}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// LastCommitSubject returns the first line of the latest commit message (for default PR title).
func LastCommitSubject(workdir string) (string, error) {
	var cmd *exec.Cmd
	if strings.TrimSpace(workdir) == "" {
		cmd = exec.Command("git", "log", "-1", "--format=%s")
	} else {
		cmd = exec.Command("git", "-C", workdir, "log", "-1", "--format=%s")
	}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// BranchDiff returns the git diff between base and head.
// It tries origin/<base>...<head>, then <base>...<head>, then fallback to HEAD~1..HEAD.
func BranchDiff(workdir, base, head string) (string, error) {
	var candidates []string
	if base != "" && head != "" {
		if !strings.HasPrefix(base, "origin/") {
			candidates = append(candidates, fmt.Sprintf("origin/%s...%s", base, head))
		}
		candidates = append(candidates, fmt.Sprintf("%s...%s", base, head))
	}
	if base != "" {
		if !strings.HasPrefix(base, "origin/") {
			candidates = append(candidates, fmt.Sprintf("origin/%s...HEAD", base))
		}
		candidates = append(candidates, fmt.Sprintf("%s...HEAD", base))
	}

	for _, rangeSpec := range candidates {
		args := []string{"diff", rangeSpec}
		var cmd *exec.Cmd
		if strings.TrimSpace(workdir) == "" {
			cmd = exec.Command("git", args...)
		} else {
			cmd = exec.Command("git", append([]string{"-C", workdir}, args...)...)
		}
		out, err := cmd.Output()
		if err == nil && len(out) > 0 {
			return string(out), nil
		}
	}

	// Fallback to diff against previous commit
	var cmd *exec.Cmd
	if strings.TrimSpace(workdir) == "" {
		cmd = exec.Command("git", "diff", "HEAD~1..HEAD")
	} else {
		cmd = exec.Command("git", "-C", workdir, "diff", "HEAD~1..HEAD")
	}
	if out, err := cmd.Output(); err == nil && len(out) > 0 {
		return string(out), nil
	}

	return "", nil
}

// CommitLog returns oneline commit subjects between base and head.
func CommitLog(workdir, base, head string) (string, error) {
	var candidates []string
	if base != "" && head != "" {
		if !strings.HasPrefix(base, "origin/") {
			candidates = append(candidates, fmt.Sprintf("origin/%s..%s", base, head))
		}
		candidates = append(candidates, fmt.Sprintf("%s..%s", base, head))
	}
	if base != "" {
		if !strings.HasPrefix(base, "origin/") {
			candidates = append(candidates, fmt.Sprintf("origin/%s..HEAD", base))
		}
		candidates = append(candidates, fmt.Sprintf("%s..HEAD", base))
	}

	for _, rangeSpec := range candidates {
		args := []string{"log", rangeSpec, "--oneline"}
		var cmd *exec.Cmd
		if strings.TrimSpace(workdir) == "" {
			cmd = exec.Command("git", args...)
		} else {
			cmd = exec.Command("git", append([]string{"-C", workdir}, args...)...)
		}
		out, err := cmd.Output()
		if err == nil && len(strings.TrimSpace(string(out))) > 0 {
			return strings.TrimSpace(string(out)), nil
		}
	}

	// Fallback: last 5 commits
	var cmd *exec.Cmd
	if strings.TrimSpace(workdir) == "" {
		cmd = exec.Command("git", "log", "-5", "--oneline")
	} else {
		cmd = exec.Command("git", "-C", workdir, "log", "-5", "--oneline")
	}
	if out, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(out)), nil
	}

	return "", nil
}
