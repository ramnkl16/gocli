// Package deploy reads gocli's deploy.yml and runs the named pipelines.
//
// A pipeline is an ordered list of steps. Each step has a "type" that maps
// to a provider (script, docker, k8s). Providers are pluggable — see
// providers.go — so adding a new one is a matter of registering it.
package deploy

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Spec is the top-level deploy.yml schema.
type Spec struct {
	Version   int                 `yaml:"version"`
	Env       map[string]string   `yaml:"env,omitempty"`
	Pipelines map[string]Pipeline `yaml:"pipelines"`
}

// Pipeline is a named sequence of steps.
type Pipeline struct {
	Description string            `yaml:"description,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	Steps       []Step            `yaml:"steps"`
}

// Step is one unit of work delegated to a provider.
type Step struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"` // script | docker | k8s

	// Common: extra env applied just for this step.
	Env map[string]string `yaml:"env,omitempty"`

	// type=script
	Run     string `yaml:"run,omitempty"`     // raw command
	Shell   string `yaml:"shell,omitempty"`   // sh | bash | pwsh (defaults per OS)
	WorkDir string `yaml:"workdir,omitempty"` // cwd for the command

	// type=docker
	Image      string `yaml:"image,omitempty"`
	Context    string `yaml:"context,omitempty"`    // build context (for docker build)
	Dockerfile string `yaml:"dockerfile,omitempty"` // dockerfile path
	Push       bool   `yaml:"push,omitempty"`

	// type=k8s
	Manifest  string `yaml:"manifest,omitempty"`  // path to manifest dir/file
	Namespace string `yaml:"namespace,omitempty"` // -n NAMESPACE
	K8sAction string `yaml:"action,omitempty"`    // apply | rollout | delete

	// Provider-specific extra arguments. For docker: appended to
	// `docker build|run|push`. For k8s rollout: required positional
	// args (e.g. ["status", "deployment/web"]).
	Args []string `yaml:"args,omitempty"`
}

// LoadSpec parses a deploy.yml file from disk.
func LoadSpec(path string) (*Spec, error) {
	if path == "" {
		path = "deploy.yml"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var s Spec
	if err := yaml.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if s.Pipelines == nil {
		return nil, fmt.Errorf("%s defines no pipelines", path)
	}
	return &s, nil
}
