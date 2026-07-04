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

func TestLoadAIFallbackConfig(t *testing.T) {
	t.Setenv("AI_PROVIDER", "gemini")
	t.Setenv("AI_FALLBACK_ENABLED", "true")
	t.Setenv("AI_FALLBACK_PROVIDER", "huggingface")

	cfg := Load()

	if cfg.AIProvider != "gemini" {
		t.Fatalf("expected primary provider gemini, got %q", cfg.AIProvider)
	}
	if !cfg.AIFallbackEnabled {
		t.Fatal("expected AI fallback enabled")
	}
	if cfg.AIFallbackProvider != "huggingface" {
		t.Fatalf("expected fallback provider huggingface, got %q", cfg.AIFallbackProvider)
	}
}

func TestLoadAIFallbackDefaults(t *testing.T) {
	t.Setenv("AI_PROVIDER", "")
	t.Setenv("AI_FALLBACK_ENABLED", "")
	t.Setenv("AI_FALLBACK_PROVIDER", "")

	cfg := Load()

	if cfg.AIProvider != "gemini" {
		t.Fatalf("expected default provider gemini, got %q", cfg.AIProvider)
	}
	if cfg.AIFallbackEnabled {
		t.Fatal("expected AI fallback disabled by default")
	}
	if cfg.AIFallbackProvider != "" {
		t.Fatalf("expected empty fallback provider by default, got %q", cfg.AIFallbackProvider)
	}
}
