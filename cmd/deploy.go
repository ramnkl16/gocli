package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/yourorg/gocli/internal/deploy"
	"github.com/yourorg/gocli/internal/ui"
)

var (
	deployFile   string
	deployDryRun bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Run deploy.yml pipelines (script, docker, k8s/EKS, ssh)",
}

var deployListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the pipelines defined in deploy.yml",
	RunE:  runDeployList,
}

var deployRunCmd = &cobra.Command{
	Use:   "run <pipeline>",
	Short: "Run a named pipeline",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeployRun,
}

var deployValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate deploy.yml without executing it",
	RunE:  runDeployValidate,
}

func init() {
	for _, c := range []*cobra.Command{deployListCmd, deployRunCmd, deployValidateCmd} {
		c.Flags().StringVarP(&deployFile, "file", "f", "deploy.yml", "path to deploy spec")
	}
	deployRunCmd.Flags().BoolVar(&deployDryRun, "dry-run", false, "print the plan without executing")

	deployCmd.AddCommand(deployListCmd, deployRunCmd, deployValidateCmd)
}

func runDeployList(cmd *cobra.Command, _ []string) error {
	s, err := deploy.LoadSpec(deployFile)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(s.Pipelines))
	for k := range s.Pipelines {
		names = append(names, k)
	}
	sort.Strings(names)
	rows := make([][]string, 0, len(names))
	for _, n := range names {
		pl := s.Pipelines[n]
		rows = append(rows, []string{n, fmt.Sprintf("%d", len(pl.Steps)), pl.Description})
	}
	ui.Section("Pipelines in " + deployFile)
	ui.Table(os.Stdout, []string{"NAME", "STEPS", "DESCRIPTION"}, rows)
	return nil
}

func runDeployRun(cmd *cobra.Command, args []string) error {
	s, err := deploy.LoadSpec(deployFile)
	if err != nil {
		return err
	}
	r := deploy.New(s, deployDryRun)
	return r.Run(context.Background(), args[0])
}

func runDeployValidate(cmd *cobra.Command, _ []string) error {
	s, err := deploy.LoadSpec(deployFile)
	if err != nil {
		return err
	}
	fmt.Println(ui.OK(fmt.Sprintf("✓ %s is valid (%d pipelines)", deployFile, len(s.Pipelines))))
	return nil
}
