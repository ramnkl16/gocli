package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Provider executes a single Step.
type Provider interface {
	Run(ctx context.Context, s Step, env map[string]string) error
}

// providers is the registry, keyed by Step.Type.
var providers = map[string]Provider{
	"script": scriptProvider{},
	"docker": dockerProvider{},
	"k8s":    k8sProvider{},
	"ssh":    sshProvider{},
}

// Register adds a custom provider (call from another file/package init).
func Register(name string, p Provider) { providers[name] = p }

// providerFor returns the registered provider for a step type.
func providerFor(t string) (Provider, error) {
	p, ok := providers[t]
	if !ok {
		return nil, fmt.Errorf("unknown step type %q (available: %s)", t, strings.Join(providerNames(), ", "))
	}
	return p, nil
}

func providerNames() []string {
	out := make([]string, 0, len(providers))
	for k := range providers {
		out = append(out, k)
	}
	return out
}

// runShell executes a command, streaming stdio, with merged env and optional cwd.
func runShell(ctx context.Context, name string, args []string, env map[string]string, workdir string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if workdir != "" {
		cmd.Dir = workdir
	}
	cmd.Env = mergedEnv(env)
	return cmd.Run()
}

func mergedEnv(extra map[string]string) []string {
	out := os.Environ()
	for k, v := range extra {
		out = append(out, k+"="+v)
	}
	return out
}

// ── script ────────────────────────────────────────────────────────────

type scriptProvider struct{}

func (scriptProvider) Run(ctx context.Context, s Step, env map[string]string) error {
	if strings.TrimSpace(s.Run) == "" {
		return fmt.Errorf("script step %q is missing `run`", s.Name)
	}
	shell, args := pickShell(s.Shell, s.Run)
	return runShell(ctx, shell, args, env, s.WorkDir)
}

func pickShell(requested, script string) (string, []string) {
	if requested == "" {
		if runtime.GOOS == "windows" {
			requested = "powershell"
		} else {
			requested = "bash"
		}
	}
	switch strings.ToLower(requested) {
	case "powershell", "pwsh":
		return requested, []string{"-NoProfile", "-Command", script}
	case "cmd":
		return "cmd", []string{"/C", script}
	default: // bash / sh / zsh
		return requested, []string{"-c", script}
	}
}

// ── docker ────────────────────────────────────────────────────────────

type dockerProvider struct{}

func (dockerProvider) Run(ctx context.Context, s Step, env map[string]string) error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker not found: %w", err)
	}
	switch {
	case s.Context != "" || s.Dockerfile != "":
		return runDockerBuild(ctx, s, env)
	case s.Push:
		if s.Image == "" {
			return fmt.Errorf("docker push step %q requires `image`", s.Name)
		}
		return runShell(ctx, "docker", append([]string{"push", s.Image}, s.Args...), env, s.WorkDir)
	case s.Image != "":
		args := append([]string{"run", "--rm"}, s.Args...)
		args = append(args, s.Image)
		return runShell(ctx, "docker", args, env, s.WorkDir)
	default:
		return fmt.Errorf("docker step %q needs one of: image, context, dockerfile, push", s.Name)
	}
}

func runDockerBuild(ctx context.Context, s Step, env map[string]string) error {
	args := []string{"build"}
	if s.Dockerfile != "" {
		args = append(args, "-f", s.Dockerfile)
	}
	if s.Image != "" {
		args = append(args, "-t", s.Image)
	}
	args = append(args, s.Args...)
	context := s.Context
	if context == "" {
		context = "."
	}
	args = append(args, context)
	if err := runShell(ctx, "docker", args, env, s.WorkDir); err != nil {
		return err
	}
	if s.Push && s.Image != "" {
		return runShell(ctx, "docker", []string{"push", s.Image}, env, s.WorkDir)
	}
	return nil
}

// ── k8s ───────────────────────────────────────────────────────────────

type k8sProvider struct{}

func (k8sProvider) Run(ctx context.Context, s Step, env map[string]string) error {
	if _, err := exec.LookPath("kubectl"); err != nil {
		return fmt.Errorf("kubectl not found: %w", err)
	}
	action := strings.ToLower(s.K8sAction)
	if action == "" {
		action = "apply"
	}
	var args []string
	switch action {
	case "apply":
		if s.Manifest == "" {
			return fmt.Errorf("k8s apply step %q requires `manifest`", s.Name)
		}
		args = []string{"apply", "-f", s.Manifest}
	case "delete":
		if s.Manifest == "" {
			return fmt.Errorf("k8s delete step %q requires `manifest`", s.Name)
		}
		args = []string{"delete", "-f", s.Manifest}
	case "rollout":
		// args holds the rollout target, e.g. ["status","deployment/web"]
		if len(s.Args) == 0 {
			return fmt.Errorf("k8s rollout step %q requires `args` (e.g. [status, deployment/web])", s.Name)
		}
		args = append([]string{"rollout"}, s.Args...)
	default:
		return fmt.Errorf("unknown k8s action %q", action)
	}
	if s.Namespace != "" {
		args = append(args, "-n", s.Namespace)
	}
	if kc := strings.TrimSpace(s.KubeContext); kc != "" {
		args = append(args, "--context", kc)
	}
	return runShell(ctx, "kubectl", args, env, s.WorkDir)
}

// ── ssh ───────────────────────────────────────────────────────────────

type sshProvider struct{}

func (sshProvider) Run(ctx context.Context, s Step, env map[string]string) error {
	if _, err := exec.LookPath("ssh"); err != nil {
		return fmt.Errorf("ssh not found: %w", err)
	}
	host := strings.TrimSpace(s.Host)
	if host == "" {
		return fmt.Errorf("ssh step %q requires `host`", s.Name)
	}
	if strings.TrimSpace(s.Run) == "" {
		return fmt.Errorf("ssh step %q requires `run`", s.Name)
	}

	var sshArgs []string
	sshArgs = append(sshArgs, s.Args...)
	if p := s.Port; p > 0 && p != 22 {
		sshArgs = append(sshArgs, "-p", fmt.Sprintf("%d", p))
	}
	if id := strings.TrimSpace(s.IdentityFile); id != "" {
		path, err := expandUserPath(id)
		if err != nil {
			return fmt.Errorf("ssh step %q identity_file: %w", s.Name, err)
		}
		sshArgs = append(sshArgs, "-i", path)
	}

	target := host
	if u := strings.TrimSpace(s.User); u != "" {
		target = u + "@" + host
	}
	sshArgs = append(sshArgs, target, "bash", "-s")

	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = strings.NewReader(s.Run)
	cmd.Env = mergedEnv(env)
	return cmd.Run()
}

func expandUserPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", nil
	}
	if p == "~" {
		return os.UserHomeDir()
	}
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(p, "~/")), nil
	}
	return filepath.Clean(p), nil
}
