package config

import "testing"

func TestLoadBrevoConfig(t *testing.T) {
	t.Setenv("BREVO_ENABLED", "true")
	t.Setenv("BREVO_API_KEY", "brevo-key")
	t.Setenv("BREVO_API_URL", "https://example.com/email")
	t.Setenv("BREVO_SENDER_EMAIL", "reports@example.com")
	t.Setenv("BREVO_SENDER_NAME", "Reports")
	t.Setenv("BREVO_REPORT_RECIPIENTS", "one@example.com, two@example.com ,,three@example.com")

	cfg := Load()

	if !cfg.Brevo.Enabled {
		t.Fatal("expected Brevo to be enabled")
	}
	if cfg.Brevo.APIKey != "brevo-key" {
		t.Fatalf("expected API key, got %q", cfg.Brevo.APIKey)
	}
	if cfg.Brevo.APIURL != "https://example.com/email" {
		t.Fatalf("expected custom API URL, got %q", cfg.Brevo.APIURL)
	}
	if cfg.Brevo.SenderEmail != "reports@example.com" {
		t.Fatalf("expected sender email, got %q", cfg.Brevo.SenderEmail)
	}
	if cfg.Brevo.SenderName != "Reports" {
		t.Fatalf("expected sender name, got %q", cfg.Brevo.SenderName)
	}
	if len(cfg.Brevo.ReportEmails) != 3 {
		t.Fatalf("expected 3 recipients, got %d", len(cfg.Brevo.ReportEmails))
	}
	if cfg.Brevo.ReportEmails[1] != "two@example.com" {
		t.Fatalf("expected trimmed recipient, got %q", cfg.Brevo.ReportEmails[1])
	}
}

func TestLoadBrevoDefaults(t *testing.T) {
	t.Setenv("BREVO_ENABLED", "")
	t.Setenv("BREVO_API_KEY", "")
	t.Setenv("BREVO_API_URL", "")
	t.Setenv("BREVO_SENDER_EMAIL", "")
	t.Setenv("BREVO_SENDER_NAME", "")
	t.Setenv("BREVO_REPORT_RECIPIENTS", "")

	cfg := Load()

	if cfg.Brevo.Enabled {
		t.Fatal("expected Brevo to be disabled by default")
	}
	if cfg.Brevo.APIURL != "https://api.brevo.com/v3/smtp/email" {
		t.Fatalf("expected default API URL, got %q", cfg.Brevo.APIURL)
	}
	if cfg.Brevo.SenderName != "Sugary" {
		t.Fatalf("expected default sender name, got %q", cfg.Brevo.SenderName)
	}
	if len(cfg.Brevo.ReportEmails) != 0 {
		t.Fatalf("expected no recipients, got %d", len(cfg.Brevo.ReportEmails))
	}
}
