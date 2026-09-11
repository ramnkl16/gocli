package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yourorg/gocli/internal/config"
	"github.com/yourorg/gocli/internal/email"
	"github.com/yourorg/gocli/internal/ui"
)

var emailNotifyMessage string
var emailNotifySubject string

var emailCmd = &cobra.Command{
	Use:   "email",
	Short: "Email notifications (SMTP)",
	Long: `Send plain-text email notifications via SMTP.

Configure SMTP in ~/.gocli/config.yaml under email.*, or set GOCLI_EMAIL_SMTP_HOST,
GOCLI_EMAIL_FROM, GOCLI_EMAIL_DEVOPS_TO, GOCLI_EMAIL_DEPLOY_TO, and
GOCLI_SMTP_PASSWORD (env wins).`,
}

var emailNotifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Send a notification email",
}

var emailNotifyDevCompleteCmd = &cobra.Command{
	Use:   "dev-complete",
	Short: "Email the DevOps recipient (email.devops_to / GOCLI_EMAIL_DEVOPS_TO)",
	RunE:  runEmailNotifyDevComplete,
}

var emailNotifyDeploymentCmd = &cobra.Command{
	Use:   "deployment",
	Short: "Email the deployment recipient (email.deploy_to / GOCLI_EMAIL_DEPLOY_TO)",
	RunE:  runEmailNotifyDeployment,
}

func init() {
	for _, c := range []*cobra.Command{emailNotifyDevCompleteCmd, emailNotifyDeploymentCmd} {
		c.Flags().StringVarP(&emailNotifyMessage, "message", "m", "", "message body (default: generic text for the notification type)")
		c.Flags().StringVarP(&emailNotifySubject, "subject", "s", "", "email subject (default: generic subject for the notification type)")
	}
	emailNotifyCmd.AddCommand(emailNotifyDevCompleteCmd, emailNotifyDeploymentCmd)
	emailCmd.AddCommand(emailNotifyCmd)
}

func runEmailNotifyDevComplete(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	to := email.DevOpsTo(cfg)
	if to == "" {
		return fmt.Errorf("no DevOps recipient: set email.devops_to in config or GOCLI_EMAIL_DEVOPS_TO")
	}
	s, err := email.ResolveSMTPSettings(cfg)
	if err != nil {
		return err
	}
	subject := strings.TrimSpace(emailNotifySubject)
	if subject == "" {
		subject = "Development complete"
	}
	msg := strings.TrimSpace(emailNotifyMessage)
	if msg == "" {
		msg = "Development work is complete and ready for the DevOps team."
	}
	if err := email.SendPlainText(context.Background(), s, to, subject, msg); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, ui.OK("✓ Sent email (DevOps)"))
	return nil
}

func runEmailNotifyDeployment(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	to := email.DeployTo(cfg)
	if to == "" {
		return fmt.Errorf("no deploy recipient: set email.deploy_to in config or GOCLI_EMAIL_DEPLOY_TO")
	}
	s, err := email.ResolveSMTPSettings(cfg)
	if err != nil {
		return err
	}
	subject := strings.TrimSpace(emailNotifySubject)
	if subject == "" {
		subject = "Deployment complete"
	}
	msg := strings.TrimSpace(emailNotifyMessage)
	if msg == "" {
		msg = "Deployment completed successfully."
	}
	if err := email.SendPlainText(context.Background(), s, to, subject, msg); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, ui.OK("✓ Sent email (deployment)"))
	return nil
}
