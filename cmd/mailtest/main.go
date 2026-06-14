package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"sugary/internal/config"
	"sugary/internal/domain"
	"sugary/internal/repository/mail"
)

func main() {
	reportType := flag.String("type", "daily", "report type to send: daily or weekly")
	date := flag.String("date", time.Now().Format(time.DateOnly), "daily report date in YYYY-MM-DD")
	weekStart := flag.String("week-start", startOfWeek(time.Now()).Format(time.DateOnly), "weekly report start date in YYYY-MM-DD")
	source := flag.String("source", "gemini", "AI source to show in the email: fallback, gemini, or huggingface")
	flag.Parse()

	cfg := config.Load()
	if !cfg.Brevo.Enabled {
		log.Fatal("BREVO_ENABLED must be true")
	}
	if len(cfg.Brevo.ReportEmails) == 0 {
		log.Fatal("BREVO_REPORT_RECIPIENTS must contain at least one email")
	}

	sender, err := mail.NewBrevoReportEmailSender(cfg.Brevo)
	if err != nil {
		log.Fatalf("create brevo sender: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch *reportType {
	case "daily":
		reportDate, err := time.Parse(time.DateOnly, *date)
		if err != nil {
			log.Fatalf("invalid -date: %v", err)
		}
		if err := sender.SendDailyReport(ctx, sampleDailyReport(reportDate, *source)); err != nil {
			log.Fatalf("send daily report email: %v", err)
		}
		fmt.Fprintf(os.Stdout, "sent daily report test email for %s to %v\n", reportDate.Format(time.DateOnly), cfg.Brevo.ReportEmails)
	case "weekly":
		start, err := time.Parse(time.DateOnly, *weekStart)
		if err != nil {
			log.Fatalf("invalid -week-start: %v", err)
		}
		if err := sender.SendWeeklyReport(ctx, sampleWeeklyReport(start, *source)); err != nil {
			log.Fatalf("send weekly report email: %v", err)
		}
		fmt.Fprintf(os.Stdout, "sent weekly report test email for %s to %v\n", start.Format(time.DateOnly), cfg.Brevo.ReportEmails)
	default:
		log.Fatalf("unsupported -type %q; use daily or weekly", *reportType)
	}
}

func sampleDailyReport(date time.Time, source string) domain.DailyReport {
	source, status := normalizeSourceAndStatus(source)
	return domain.DailyReport{
		Date:              date,
		MealCount:         3,
		TotalSugarGrams:   42.5,
		AverageSugarGrams: 14.2,
		HighestRiskLevel:  "medium",
		Summary:           "This is a test daily report email from Sugary. Your estimated sugar intake stayed moderate, with one drink contributing most of the total.",
		AIInsightSource:   source,
		AIInsightStatus:   status,
		AIInsights: domain.DailyReportAIInsights{
			Summary: "This is a test daily report email from Sugary.",
			Recommendations: []string{
				"Choose unsweetened drinks tomorrow.",
				"Pair higher-carb meals with protein when possible.",
			},
			PatternSignals: []string{
				"Sweet drinks contributed the largest sugar share.",
			},
		},
	}
}

func sampleWeeklyReport(weekStart time.Time, source string) domain.WeeklyReport {
	source, status := normalizeSourceAndStatus(source)
	breakdown := make([]domain.WeeklyReportDaily, 0, 7)
	for i := 0; i < 7; i++ {
		date := weekStart.AddDate(0, 0, i)
		breakdown = append(breakdown, domain.WeeklyReportDaily{
			Date:              date,
			MealCount:         2 + i%2,
			AnalyzedMealCount: 2,
			TotalSugarGrams:   18 + float64(i*3),
			AverageSugarGrams: 9 + float64(i),
			HighestRiskLevel:  []string{"low", "medium", "medium", "high", "low", "medium", "low"}[i],
		})
	}

	return domain.WeeklyReport{
		WeekStartDate:     weekStart,
		WeekEndDate:       weekStart.AddDate(0, 0, 6),
		CreatedAt:         time.Now().UTC(),
		MealCount:         18,
		AnalyzedMealCount: 14,
		TotalSugarGrams:   189,
		AverageSugarGrams: 13.5,
		HighestRiskLevel:  "high",
		Summary:           "This is a test weekly report email from Sugary. Sugar intake was mostly moderate, with one higher-risk day driven by sweet drinks or dessert.",
		DailyBreakdown:    breakdown,
		AIInsightSource:   source,
		AIInsightStatus:   status,
		AIInsights: domain.WeeklyReportAIInsights{
			Summary: "This is a test weekly report email from Sugary.",
			Recommendations: []string{
				"Plan lower-sugar drink options for the next week.",
				"Watch days with multiple snacks or desserts.",
			},
			PatternSignals: []string{
				"Mid-week intake peaked compared with the rest of the week.",
			},
		},
	}
}

func normalizeSourceAndStatus(source string) (string, string) {
	switch source {
	case "fallback":
		return "fallback", "fallback"
	case "huggingface", "huggie":
		return "huggingface", "completed"
	default:
		return "gemini", "completed"
	}
}

func startOfWeek(value time.Time) time.Time {
	day := time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
	offset := (int(day.Weekday()) + 6) % 7
	return day.AddDate(0, 0, -offset)
}
