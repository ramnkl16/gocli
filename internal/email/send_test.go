package email

import (
	"context"
	"testing"

	"github.com/yourorg/gocli/internal/config"
)

func TestSendPlainText_Validation(t *testing.T) {
	t.Parallel()
	s := SMTPSettings{Host: "smtp.example.com", Port: 587, From: "a@b.com"}
	ctx := context.Background()

	cases := []struct {
		name string
		to   string
		sub  string
		body string
	}{
		{"empty to", "", "sub", "body"},
		{"empty subject", "x@y.com", "", "body"},
		{"empty body", "x@y.com", "sub", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := SendPlainText(ctx, s, tc.to, tc.sub, tc.body)
			if err == nil {
				t.Fatal("want error")
			}
		})
	}
}

func TestResolveSMTPSettings_EnvOverrides(t *testing.T) {
	t.Setenv(envSMTPHost, "mail.corp")
	t.Setenv(envSMTPPort, "2525")
	t.Setenv(envFrom, "bot@corp")
	t.Setenv(envSMTPUser, "bot@corp")
	t.Setenv(envSMTPPassword, "secret")

	cfg := configWithEmail()
	s, err := ResolveSMTPSettings(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	if s.Host != "mail.corp" || s.Port != 2525 || s.From != "bot@corp" || s.Password != "secret" {
		t.Fatalf("settings=%+v", s)
	}
}

func configWithEmail() config.Config {
	return config.Config{
		Email: config.EmailConfig{
			SMTPHost: "old",
			SMTPPort: 587,
			From:     "old@corp",
		},
	}
}
