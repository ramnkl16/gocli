package email

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/yourorg/gocli/internal/config"
	"github.com/yourorg/gocli/internal/secrets"
)

const (
	envSMTPHost     = "GOCLI_EMAIL_SMTP_HOST"
	envSMTPPort     = "GOCLI_EMAIL_SMTP_PORT"
	envFrom         = "GOCLI_EMAIL_FROM"
	envSMTPUser     = "GOCLI_EMAIL_SMTP_USER"
	envDevOpsTo     = "GOCLI_EMAIL_DEVOPS_TO"
	envDeployTo     = "GOCLI_EMAIL_DEPLOY_TO"
	envSMTPPassword = "GOCLI_SMTP_PASSWORD"
)

const defaultSMTPPort = 587

// DevOpsTo returns the dev-complete recipient: env overrides config.
func DevOpsTo(c *config.Config) string {
	if v := strings.TrimSpace(os.Getenv(envDevOpsTo)); v != "" {
		return v
	}
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.Email.DevOpsTo)
}

// DeployTo returns the deployment-complete recipient: env overrides config.
func DeployTo(c *config.Config) string {
	if v := strings.TrimSpace(os.Getenv(envDeployTo)); v != "" {
		return v
	}
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.Email.DeployTo)
}

// SMTPSettings resolves SMTP connection settings from config, env, and keyring.
func ResolveSMTPSettings(c *config.Config) (SMTPSettings, error) {
	var s SMTPSettings
	if c != nil {
		s.Host = strings.TrimSpace(c.Email.SMTPHost)
		s.Port = c.Email.SMTPPort
		s.From = strings.TrimSpace(c.Email.From)
		s.User = strings.TrimSpace(c.Email.SMTPUser)
	}
	if v := strings.TrimSpace(os.Getenv(envSMTPHost)); v != "" {
		s.Host = v
	}
	if v := strings.TrimSpace(os.Getenv(envSMTPPort)); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			return SMTPSettings{}, fmt.Errorf("%s: invalid port %q", envSMTPPort, v)
		}
		s.Port = port
	}
	if v := strings.TrimSpace(os.Getenv(envFrom)); v != "" {
		s.From = v
	}
	if v := strings.TrimSpace(os.Getenv(envSMTPUser)); v != "" {
		s.User = v
	}
	if s.Port <= 0 {
		s.Port = defaultSMTPPort
	}
	password, err := secrets.Get(secrets.KeySMTPPassword)
	if err != nil {
		return SMTPSettings{}, err
	}
	if v := strings.TrimSpace(os.Getenv(envSMTPPassword)); v != "" {
		password = v
	}
	s.Password = password
	return s, nil
}
