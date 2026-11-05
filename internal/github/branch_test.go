package github

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	t.Cleanup(func() {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err == nil {
				_ = os.Chmod(path, 0777)
			}
			return nil
		})
	})

	runGit := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
		}
	}

	runGit("init")
	runGit("config", "user.name", "Test User")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "commit.gpgSign", "false")

	// Commit 1 on main
	file1 := filepath.Join(dir, "README.md")
	if err := os.WriteFile(file1, []byte("# Hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "README.md")
	runGit("commit", "-m", "Initial commit")

	// Branch feature
	runGit("checkout", "-b", "feature")
	file2 := filepath.Join(dir, "app.go")
	if err := os.WriteFile(file2, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "app.go")
	runGit("commit", "-m", "Add app.go")

	return dir
}

func TestCurrentBranchAndLastCommit(t *testing.T) {
	dir := setupTestGitRepo(t)

	branch, err := CurrentBranch(dir)
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if branch != "feature" {
		t.Errorf("got branch %q, want feature", branch)
	}

	subj, err := LastCommitSubject(dir)
	if err != nil {
		t.Fatalf("LastCommitSubject: %v", err)
	}
	if subj != "Add app.go" {
		t.Errorf("got subject %q, want 'Add app.go'", subj)
	}
}

func TestBranchDiffAndCommitLog(t *testing.T) {
	dir := setupTestGitRepo(t)

	diff, err := BranchDiff(dir, "master", "feature")
	if err != nil {
		t.Fatalf("BranchDiff: %v", err)
	}
	if !strings.Contains(diff, "app.go") && !strings.Contains(diff, "package main") {
		diff, err = BranchDiff(dir, "main", "feature")
		if err != nil {
			t.Fatalf("BranchDiff with main: %v", err)
		}
		if !strings.Contains(diff, "app.go") {
			t.Errorf("diff expected to contain app.go, got: %s", diff)
		}
	}

	logs, err := CommitLog(dir, "HEAD~1", "HEAD")
	if err != nil {
		t.Fatalf("CommitLog: %v", err)
	}
	if !strings.Contains(logs, "Add app.go") {
		t.Errorf("commit log expected to contain 'Add app.go', got: %s", logs)
	}
}
