// Package cursor shells out to the Cursor Agent CLI for automated coding tasks.
package cursor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/yourorg/gocli/internal/config"
)

// RunInput configures a non-interactive Cursor Agent run.
type RunInput struct {
	Prompt    string
	Workdir   string
	Model     string
	ExtraArgs []string
}

// Run invokes `agent -p --force --trust --workspace <dir> "<prompt>"`.
func Run(ctx context.Context, in RunInput) error {
	binary, err := resolveBinary()
	if err != nil {
		return err
	}
	prompt := strings.TrimSpace(in.Prompt)
	if prompt == "" {
		return fmt.Errorf("cursor agent: empty prompt")
	}
	workdir := strings.TrimSpace(in.Workdir)
	if workdir == "" {
		return fmt.Errorf("cursor agent: workdir is required")
	}

	args := []string{"-p", "--force", "--trust", "--workspace", workdir}
	if model := strings.TrimSpace(in.Model); model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, in.ExtraArgs...)
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = workdir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cursor agent: %w (is the Cursor CLI installed? https://cursor.com/docs/cli/overview)", err)
	}
	return nil
}

func resolveBinary() (string, error) {
	if v := strings.TrimSpace(os.Getenv("GOCLI_CURSOR_BINARY")); v != "" {
		return v, nil
	}
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	binary := strings.TrimSpace(cfg.Cursor.Binary)
	if binary == "" {
		binary = "agent"
	}
	path, err := exec.LookPath(binary)
	if err != nil {
		return "", fmt.Errorf("cursor CLI %q not found in PATH: %w", binary, err)
	}
	return path, nil
}
