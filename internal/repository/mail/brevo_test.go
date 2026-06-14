package mail

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sugary/internal/config"
	"sugary/internal/domain"
)

func TestBrevoReportEmailSenderSendsDailyReport(t *testing.T) {
	t.Parallel()

	var got brevoEmailRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("api-key") != "brevo-key" {
			t.Fatalf("expected api-key header")
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	sender, err := NewBrevoReportEmailSender(config.BrevoConfig{
		Enabled:      true,
		APIKey:       "brevo-key",
		APIURL:       server.URL,
		SenderEmail:  "reports@example.com",
		SenderName:   "Sugary",
		ReportEmails: []string{"user@example.com"},
	})
	if err != nil {
		t.Fatalf("expected no constructor error, got %v", err)
	}

	err = sender.SendDailyReport(context.Background(), domain.DailyReport{
		Date:              time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC),
		MealCount:         2,
		TotalSugarGrams:   41,
		AverageSugarGrams: 20.5,
		HighestRiskLevel:  "high",
		Summary:           "Daily summary",
		AIInsightSource:   "gemini",
		AIInsightStatus:   "completed",
		AIInsights: domain.DailyReportAIInsights{
			Recommendations: []string{"Drink water"},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Sender.Email != "reports@example.com" {
		t.Fatalf("expected sender email, got %q", got.Sender.Email)
	}
	if len(got.To) != 1 || got.To[0].Email != "user@example.com" {
		t.Fatalf("expected recipient, got %+v", got.To)
	}
	if got.Subject != "Sugary daily report - 2026-06-12" {
		t.Fatalf("expected daily subject, got %q", got.Subject)
	}
	if !strings.Contains(got.HTMLContent, "Daily summary") ||
		!strings.Contains(got.HTMLContent, "41.0g") ||
		!strings.Contains(got.HTMLContent, "gemini") ||
		!strings.Contains(got.HTMLContent, "completed") {
		t.Fatalf("expected rendered report content, got %q", got.HTMLContent)
	}
}

func TestBrevoReportEmailSenderSendsWeeklyReport(t *testing.T) {
	t.Parallel()

	var got brevoEmailRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender, err := NewBrevoReportEmailSender(config.BrevoConfig{
		Enabled:      true,
		APIKey:       "brevo-key",
		APIURL:       server.URL,
		SenderEmail:  "reports@example.com",
		SenderName:   "Sugary",
		ReportEmails: []string{"user@example.com"},
	})
	if err != nil {
		t.Fatalf("expected no constructor error, got %v", err)
	}

	err = sender.SendWeeklyReport(context.Background(), domain.WeeklyReport{
		WeekStartDate:     time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC),
		WeekEndDate:       time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC),
		MealCount:         3,
		AnalyzedMealCount: 2,
		TotalSugarGrams:   41,
		AverageSugarGrams: 20.5,
		HighestRiskLevel:  "high",
		Summary:           "Weekly summary",
		AIInsightSource:   "huggingface",
		AIInsightStatus:   "completed",
		DailyBreakdown: []domain.WeeklyReportDaily{
			{
				Date:              time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC),
				MealCount:         1,
				AnalyzedMealCount: 1,
				TotalSugarGrams:   20,
				HighestRiskLevel:  "medium",
			},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Subject != "Sugary weekly report - 2026-06-08 to 2026-06-14" {
		t.Fatalf("expected weekly subject, got %q", got.Subject)
	}
	if !strings.Contains(got.HTMLContent, "Weekly summary") ||
		!strings.Contains(got.HTMLContent, "2026-06-08") ||
		!strings.Contains(got.HTMLContent, "huggingface") ||
		!strings.Contains(got.HTMLContent, "completed") {
		t.Fatalf("expected rendered weekly content, got %q", got.HTMLContent)
	}
}

func TestBrevoReportEmailSenderReturnsErrorOnNon2xx(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	sender, err := NewBrevoReportEmailSender(config.BrevoConfig{
		Enabled:      true,
		APIKey:       "brevo-key",
		APIURL:       server.URL,
		SenderEmail:  "reports@example.com",
		ReportEmails: []string{"user@example.com"},
	})
	if err != nil {
		t.Fatalf("expected no constructor error, got %v", err)
	}

	err = sender.SendDailyReport(context.Background(), domain.DailyReport{Date: time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)})
	if err == nil {
		t.Fatal("expected non-2xx error")
	}
}

func TestBrevoReportEmailSenderNoopWhenDisabled(t *testing.T) {
	t.Parallel()

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	sender, err := NewBrevoReportEmailSender(config.BrevoConfig{
		Enabled:      false,
		APIKey:       "brevo-key",
		APIURL:       server.URL,
		SenderEmail:  "reports@example.com",
		ReportEmails: []string{"user@example.com"},
	})
	if err != nil {
		t.Fatalf("expected no constructor error, got %v", err)
	}

	err = sender.SendDailyReport(context.Background(), domain.DailyReport{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if called {
		t.Fatal("expected no HTTP request when disabled")
	}
}

func TestBrevoReportEmailSenderValidatesRequiredConfigWhenEnabled(t *testing.T) {
	t.Parallel()

	_, err := NewBrevoReportEmailSender(config.BrevoConfig{
		Enabled:      true,
		APIURL:       "https://example.com",
		SenderEmail:  "reports@example.com",
		ReportEmails: []string{"user@example.com"},
	})
	if err == nil {
		t.Fatal("expected missing API key error")
	}
}

func TestBrevoReportEmailSenderAllowsEnabledWithNoRecipients(t *testing.T) {
	t.Parallel()

	_, err := NewBrevoReportEmailSender(config.BrevoConfig{
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("expected enabled sender with no recipients to be no-op, got %v", err)
	}
}
