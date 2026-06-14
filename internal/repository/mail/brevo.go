package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"sugary/internal/config"
	"sugary/internal/domain"
)

type BrevoReportEmailSender struct {
	enabled     bool
	apiKey      string
	apiURL      string
	senderEmail string
	senderName  string
	recipients  []brevoContact
	client      *http.Client
}

type brevoContact struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type brevoEmailRequest struct {
	Sender      brevoContact   `json:"sender"`
	To          []brevoContact `json:"to"`
	Subject     string         `json:"subject"`
	HTMLContent string         `json:"htmlContent"`
}

func NewBrevoReportEmailSender(cfg config.BrevoConfig) (BrevoReportEmailSender, error) {
	recipients := make([]brevoContact, 0, len(cfg.ReportEmails))
	for _, email := range cfg.ReportEmails {
		email = strings.TrimSpace(email)
		if email != "" {
			recipients = append(recipients, brevoContact{Email: email})
		}
	}

	sender := BrevoReportEmailSender{
		enabled:     cfg.Enabled,
		apiKey:      strings.TrimSpace(cfg.APIKey),
		apiURL:      strings.TrimSpace(cfg.APIURL),
		senderEmail: strings.TrimSpace(cfg.SenderEmail),
		senderName:  strings.TrimSpace(cfg.SenderName),
		recipients:  recipients,
		client:      &http.Client{Timeout: 15 * time.Second},
	}
	if err := sender.validate(); err != nil {
		return BrevoReportEmailSender{}, err
	}
	return sender, nil
}

func (s BrevoReportEmailSender) validate() error {
	if !s.enabled || len(s.recipients) == 0 {
		return nil
	}
	if s.apiKey == "" {
		return fmt.Errorf("brevo api key is required when report email is enabled")
	}
	if s.apiURL == "" {
		return fmt.Errorf("brevo api url is required when report email is enabled")
	}
	if s.senderEmail == "" {
		return fmt.Errorf("brevo sender email is required when report email is enabled")
	}
	return nil
}

func (s BrevoReportEmailSender) SendDailyReport(ctx context.Context, report domain.DailyReport) error {
	if !s.shouldSend() {
		return nil
	}

	html, err := renderDailyReportEmail(report)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("Sugary daily report - %s", report.Date.Format(time.DateOnly))
	return s.send(ctx, subject, html)
}

func (s BrevoReportEmailSender) SendWeeklyReport(ctx context.Context, report domain.WeeklyReport) error {
	if !s.shouldSend() {
		return nil
	}

	html, err := renderWeeklyReportEmail(report)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("Sugary weekly report - %s to %s",
		report.WeekStartDate.Format(time.DateOnly),
		report.WeekEndDate.Format(time.DateOnly),
	)
	return s.send(ctx, subject, html)
}

func (s BrevoReportEmailSender) shouldSend() bool {
	return s.enabled &&
		s.apiKey != "" &&
		s.apiURL != "" &&
		s.senderEmail != "" &&
		len(s.recipients) > 0
}

func (s BrevoReportEmailSender) send(ctx context.Context, subject string, html string) error {
	payload, err := json.Marshal(brevoEmailRequest{
		Sender: brevoContact{
			Email: s.senderEmail,
			Name:  s.senderName,
		},
		To:          s.recipients,
		Subject:     subject,
		HTMLContent: html,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("api-key", s.apiKey)
	req.Header.Set("content-type", "application/json")

	start := time.Now()
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		zap.L().Warn("brevo_report_email_failed",
			zap.Int("status", resp.StatusCode),
			zap.String("subject", subject),
			zap.Int64("latency_ms", time.Since(start).Milliseconds()),
		)
		return fmt.Errorf("brevo request failed: status=%d body=%s", resp.StatusCode, string(raw))
	}

	zap.L().Info("brevo_report_email_sent",
		zap.String("subject", subject),
		zap.Int("recipient_count", len(s.recipients)),
		zap.Int64("latency_ms", time.Since(start).Milliseconds()),
	)
	return nil
}

type dailyReportEmailView struct {
	Date              string
	MealCount         int
	TotalSugarGrams   string
	AverageSugarGrams string
	HighestRiskLevel  string
	Summary           string
	AIInsightSource   string
	AIInsightStatus   string
	Recommendations   []string
	PatternSignals    []string
}

type weeklyReportEmailView struct {
	WeekStartDate     string
	WeekEndDate       string
	MealCount         int
	AnalyzedMealCount int
	TotalSugarGrams   string
	AverageSugarGrams string
	HighestRiskLevel  string
	Summary           string
	AIInsightSource   string
	AIInsightStatus   string
	Recommendations   []string
	PatternSignals    []string
	DailyBreakdown    []weeklyReportDailyView
}

type weeklyReportDailyView struct {
	Date              string
	MealCount         int
	AnalyzedMealCount int
	TotalSugarGrams   string
	HighestRiskLevel  string
}

func renderDailyReportEmail(report domain.DailyReport) (string, error) {
	var out bytes.Buffer
	err := dailyReportTemplate.Execute(&out, dailyReportEmailView{
		Date:              report.Date.Format(time.DateOnly),
		MealCount:         report.MealCount,
		TotalSugarGrams:   formatGrams(report.TotalSugarGrams),
		AverageSugarGrams: formatGrams(report.AverageSugarGrams),
		HighestRiskLevel:  report.HighestRiskLevel,
		Summary:           report.Summary,
		AIInsightSource:   report.AIInsightSource,
		AIInsightStatus:   report.AIInsightStatus,
		Recommendations:   report.AIInsights.Recommendations,
		PatternSignals:    report.AIInsights.PatternSignals,
	})
	return out.String(), err
}

func renderWeeklyReportEmail(report domain.WeeklyReport) (string, error) {
	breakdown := make([]weeklyReportDailyView, 0, len(report.DailyBreakdown))
	for _, day := range report.DailyBreakdown {
		breakdown = append(breakdown, weeklyReportDailyView{
			Date:              day.Date.Format(time.DateOnly),
			MealCount:         day.MealCount,
			AnalyzedMealCount: day.AnalyzedMealCount,
			TotalSugarGrams:   formatGrams(day.TotalSugarGrams),
			HighestRiskLevel:  day.HighestRiskLevel,
		})
	}

	var out bytes.Buffer
	err := weeklyReportTemplate.Execute(&out, weeklyReportEmailView{
		WeekStartDate:     report.WeekStartDate.Format(time.DateOnly),
		WeekEndDate:       report.WeekEndDate.Format(time.DateOnly),
		MealCount:         report.MealCount,
		AnalyzedMealCount: report.AnalyzedMealCount,
		TotalSugarGrams:   formatGrams(report.TotalSugarGrams),
		AverageSugarGrams: formatGrams(report.AverageSugarGrams),
		HighestRiskLevel:  report.HighestRiskLevel,
		Summary:           report.Summary,
		AIInsightSource:   report.AIInsightSource,
		AIInsightStatus:   report.AIInsightStatus,
		Recommendations:   report.AIInsights.Recommendations,
		PatternSignals:    report.AIInsights.PatternSignals,
		DailyBreakdown:    breakdown,
	})
	return out.String(), err
}

func formatGrams(value float64) string {
	return fmt.Sprintf("%.1fg", value)
}

var dailyReportTemplate = template.Must(template.New("daily_report_email").Parse(`<!doctype html>
<html>
<body style="font-family: Arial, sans-serif; color: #1f2937; line-height: 1.5;">
  <h1 style="margin-bottom: 4px;">Daily sugar report</h1>
  <p style="margin-top: 0; color: #6b7280;">{{.Date}}</p>
  <p>{{.Summary}}</p>
  <table cellpadding="8" cellspacing="0" style="border-collapse: collapse; border: 1px solid #e5e7eb;">
    <tr><td><strong>Meals</strong></td><td>{{.MealCount}}</td></tr>
    <tr><td><strong>Total sugar</strong></td><td>{{.TotalSugarGrams}}</td></tr>
    <tr><td><strong>Average sugar</strong></td><td>{{.AverageSugarGrams}}</td></tr>
    <tr><td><strong>Highest risk</strong></td><td>{{.HighestRiskLevel}}</td></tr>
    <tr><td><strong>AI source</strong></td><td>{{.AIInsightSource}}</td></tr>
    <tr><td><strong>AI status</strong></td><td>{{.AIInsightStatus}}</td></tr>
  </table>
  {{if .Recommendations}}
  <h2>Recommendations</h2>
  <ul>{{range .Recommendations}}<li>{{.}}</li>{{end}}</ul>
  {{end}}
  {{if .PatternSignals}}
  <h2>Pattern signals</h2>
  <ul>{{range .PatternSignals}}<li>{{.}}</li>{{end}}</ul>
  {{end}}
</body>
</html>`))

var weeklyReportTemplate = template.Must(template.New("weekly_report_email").Parse(`<!doctype html>
<html>
<body style="font-family: Arial, sans-serif; color: #1f2937; line-height: 1.5;">
  <h1 style="margin-bottom: 4px;">Weekly sugar report</h1>
  <p style="margin-top: 0; color: #6b7280;">{{.WeekStartDate}} to {{.WeekEndDate}}</p>
  <p>{{.Summary}}</p>
  <table cellpadding="8" cellspacing="0" style="border-collapse: collapse; border: 1px solid #e5e7eb;">
    <tr><td><strong>Meals</strong></td><td>{{.MealCount}}</td></tr>
    <tr><td><strong>Analyzed meals</strong></td><td>{{.AnalyzedMealCount}}</td></tr>
    <tr><td><strong>Total sugar</strong></td><td>{{.TotalSugarGrams}}</td></tr>
    <tr><td><strong>Average sugar</strong></td><td>{{.AverageSugarGrams}}</td></tr>
    <tr><td><strong>Highest risk</strong></td><td>{{.HighestRiskLevel}}</td></tr>
    <tr><td><strong>AI source</strong></td><td>{{.AIInsightSource}}</td></tr>
    <tr><td><strong>AI status</strong></td><td>{{.AIInsightStatus}}</td></tr>
  </table>
  {{if .DailyBreakdown}}
  <h2>Daily breakdown</h2>
  <table cellpadding="8" cellspacing="0" style="border-collapse: collapse; border: 1px solid #e5e7eb;">
    <tr><th align="left">Date</th><th align="left">Meals</th><th align="left">Analyzed</th><th align="left">Sugar</th><th align="left">Risk</th></tr>
    {{range .DailyBreakdown}}
    <tr><td>{{.Date}}</td><td>{{.MealCount}}</td><td>{{.AnalyzedMealCount}}</td><td>{{.TotalSugarGrams}}</td><td>{{.HighestRiskLevel}}</td></tr>
    {{end}}
  </table>
  {{end}}
  {{if .Recommendations}}
  <h2>Recommendations</h2>
  <ul>{{range .Recommendations}}<li>{{.}}</li>{{end}}</ul>
  {{end}}
  {{if .PatternSignals}}
  <h2>Pattern signals</h2>
  <ul>{{range .PatternSignals}}<li>{{.}}</li>{{end}}</ul>
  {{end}}
</body>
</html>`))
