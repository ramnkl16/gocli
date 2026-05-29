package teams

import (
	"os"
	"strings"

	"github.com/yourorg/gocli/internal/config"
)

const (
	envDevOpsWebhook = "GOCLI_TEAMS_DEVOPS_WEBHOOK"
	envDeployWebhook = "GOCLI_TEAMS_DEPLOY_WEBHOOK"
)

// DevOpsWebhookURL returns the DevOps / dev-complete channel webhook: env overrides config.
func DevOpsWebhookURL(c *config.Config) string {
	if v := strings.TrimSpace(os.Getenv(envDevOpsWebhook)); v != "" {
		return v
	}
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.Teams.DevOpsWebhook)
}

// DeployWebhookURL returns the deployment-complete channel webhook: env overrides config.
func DeployWebhookURL(c *config.Config) string {
	if v := strings.TrimSpace(os.Getenv(envDeployWebhook)); v != "" {
		return v
	}
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.Teams.DeployWebhook)
}
