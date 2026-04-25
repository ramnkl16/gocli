package github

import (
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
