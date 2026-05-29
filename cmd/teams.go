package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yourorg/gocli/internal/config"
	"github.com/yourorg/gocli/internal/teams"
	"github.com/yourorg/gocli/internal/ui"
)

var teamsNotifyMessage string

var teamsCmd = &cobra.Command{
	Use:   "teams",
	Short: "Microsoft Teams notifications (Incoming Webhooks)",
	Long: `Send messages to Teams channels using Incoming Webhook URLs.

Configure webhooks in ~/.gocli/config.yaml under teams.devops_webhook and
teams.deploy_webhook, or set GOCLI_TEAMS_DEVOPS_WEBHOOK and
GOCLI_TEAMS_DEPLOY_WEBHOOK (env wins).`,
}

var teamsNotifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Post a message to a configured webhook",
}

var teamsNotifyDevCompleteCmd = &cobra.Command{
	Use:   "dev-complete",
	Short: "Notify the DevOps group (teams.devops_webhook / GOCLI_TEAMS_DEVOPS_WEBHOOK)",
	RunE:  runTeamsNotifyDevComplete,
}

var teamsNotifyDeploymentCmd = &cobra.Command{
	Use:   "deployment",
	Short: "Notify the deployment channel (teams.deploy_webhook / GOCLI_TEAMS_DEPLOY_WEBHOOK)",
	RunE:  runTeamsNotifyDeployment,
}

func init() {
	teamsNotifyDevCompleteCmd.Flags().StringVarP(&teamsNotifyMessage, "message", "m", "", "message body (default: generic dev-complete text)")
	teamsNotifyDeploymentCmd.Flags().StringVarP(&teamsNotifyMessage, "message", "m", "", "message body (default: generic deployment-complete text)")

	teamsNotifyCmd.AddCommand(teamsNotifyDevCompleteCmd, teamsNotifyDeploymentCmd)
	teamsCmd.AddCommand(teamsNotifyCmd)
}

func runTeamsNotifyDevComplete(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	url := teams.DevOpsWebhookURL(cfg)
	if url == "" {
		return fmt.Errorf("no DevOps webhook: set teams.devops_webhook in config or GOCLI_TEAMS_DEVOPS_WEBHOOK")
	}
	msg := strings.TrimSpace(teamsNotifyMessage)
	if msg == "" {
		msg = "Development work is complete and ready for the DevOps team."
	}
	if err := teams.PostMessage(context.Background(), url, msg); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, ui.OK("✓ Sent Teams message (DevOps)"))
	return nil
}

func runTeamsNotifyDeployment(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	url := teams.DeployWebhookURL(cfg)
	if url == "" {
		return fmt.Errorf("no deploy webhook: set teams.deploy_webhook in config or GOCLI_TEAMS_DEPLOY_WEBHOOK")
	}
	msg := strings.TrimSpace(teamsNotifyMessage)
	if msg == "" {
		msg = "Deployment completed successfully."
	}
	if err := teams.PostMessage(context.Background(), url, msg); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, ui.OK("✓ Sent Teams message (deployment)"))
	return nil
}
