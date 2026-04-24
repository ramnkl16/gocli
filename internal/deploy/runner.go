package deploy

import (
	"context"
	"fmt"
	"time"

	"github.com/yourorg/gocli/internal/ui"
)

// Runner executes pipelines defined in a Spec.
type Runner struct {
	Spec   *Spec
	DryRun bool
}

// New builds a Runner.
func New(s *Spec, dryRun bool) *Runner { return &Runner{Spec: s, DryRun: dryRun} }

// Run executes a single named pipeline. Steps stop at the first failure.
func (r *Runner) Run(ctx context.Context, name string) error {
	pl, ok := r.Spec.Pipelines[name]
	if !ok {
		return fmt.Errorf("pipeline %q not found", name)
	}

	ui.Section(fmt.Sprintf("Running pipeline: %s", name))
	if pl.Description != "" {
		fmt.Println("  " + ui.Dim(pl.Description))
	}

	for i, step := range pl.Steps {
		label := fmt.Sprintf("[%d/%d] %s (%s)", i+1, len(pl.Steps), step.Name, step.Type)
		fmt.Println()
		fmt.Println(ui.Title(label))

		env := mergeEnvMaps(r.Spec.Env, pl.Env, step.Env)
		if r.DryRun {
			fmt.Println(ui.Dim("  dry-run: would execute step"))
			continue
		}

		p, err := providerFor(step.Type)
		if err != nil {
			return fmt.Errorf("step %q: %w", step.Name, err)
		}
		start := time.Now()
		if err := p.Run(ctx, step, env); err != nil {
			fmt.Println(ui.Err(fmt.Sprintf("✗ %s failed after %s: %v", step.Name, time.Since(start).Round(time.Millisecond), err)))
			return err
		}
		fmt.Println(ui.OK(fmt.Sprintf("✓ %s (%s)", step.Name, time.Since(start).Round(time.Millisecond))))
	}
	return nil
}

func mergeEnvMaps(maps ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}
