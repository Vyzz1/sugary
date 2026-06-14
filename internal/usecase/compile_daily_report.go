package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"sugary/internal/domain"
	"sugary/internal/platform/timeutil"

	"go.uber.org/zap"
)

type DailyReportPublisher interface {
	Broadcast(msg []byte)
}

type aiInsightProvider interface {
	AIInsightProviderName() string
}

type dailyReportPush struct {
	Type   string              `json:"type"`
	Status string              `json:"status"`
	Data   *domain.DailyReport `json:"data,omitempty"`
	Error  *analysisPushErr    `json:"error,omitempty"`
}

type CompileDailyReport struct {
	mealRepository        domain.MealRepository
	dailyReportRepository domain.DailyReportRepository
	interpreter           domain.DailyReportInterpreter
	publisher             DailyReportPublisher
	emailSender           domain.ReportEmailSender
}

func NewCompileDailyReport(
	mealRepository domain.MealRepository,
	dailyReportRepository domain.DailyReportRepository,
	interpreter domain.DailyReportInterpreter,
) CompileDailyReport {
	return CompileDailyReport{
		mealRepository:        mealRepository,
		dailyReportRepository: dailyReportRepository,
		interpreter:           interpreter,
	}
}

func (uc CompileDailyReport) WithPublisher(publisher DailyReportPublisher) CompileDailyReport {
	uc.publisher = publisher
	return uc
}

func (uc CompileDailyReport) WithEmailSender(sender domain.ReportEmailSender) CompileDailyReport {
	uc.emailSender = sender
	return uc
}

func (uc CompileDailyReport) broadcastReport(msg dailyReportPush) {
	if uc.publisher == nil {
		return
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		zap.L().Error("daily_report_push_marshal_failed",
			zap.String("job_name", "daily_report"),
			zap.Time("report_date", msg.Data.Date),
			zap.Error(err),
		)
		return
	}

	uc.publisher.Broadcast(payload)
}

func (uc CompileDailyReport) saveAndBroadcast(ctx context.Context, report domain.DailyReport) error {
	if err := uc.dailyReportRepository.Save(ctx, report); err != nil {
		return err
	}

	if uc.emailSender != nil {
		if err := uc.emailSender.SendDailyReport(ctx, report); err != nil {
			zap.L().Warn("daily_report_email_send_failed",
				zap.String("report_date", report.Date.Format(time.DateOnly)),
				zap.Error(err),
			)
		}
	}

	uc.broadcastReport(dailyReportPush{
		Type:   "daily_report",
		Status: "completed",
		Data:   &report,
	})

	return nil
}

func (uc CompileDailyReport) Execute(ctx context.Context, day time.Time) (domain.DailyReport, error) {
	if day.IsZero() {
		return domain.DailyReport{}, domain.ErrInvalidDate
	}

	day = timeutil.StartOfDay(day)

	existing, found, err := uc.dailyReportRepository.GetByDay(ctx, timeutil.CanonicalUTCDate(day))
	if err != nil {
		return domain.DailyReport{}, err
	}
	if found && hasCompletedAIInsights(existing.AIInsightSource, existing.AIInsightStatus) {
		zap.L().Info("daily_report_compile_skipped_existing_ai_completed",
			zap.String("report_date", existing.Date.Format(time.DateOnly)),
			zap.String("ai_insight_source", existing.AIInsightSource),
			zap.String("ai_insight_status", existing.AIInsightStatus),
		)
		return existing, nil
	}
	if found {
		zap.L().Info("daily_report_compile_reprocessing_existing_fallback",
			zap.String("report_date", existing.Date.Format(time.DateOnly)),
			zap.String("ai_insight_source", existing.AIInsightSource),
			zap.String("ai_insight_status", existing.AIInsightStatus),
		)
	}

	meals, err := uc.mealRepository.ListByDay(ctx, domain.MealsByDayFilter{
		Day:  day,
		Sort: defaultRecentMealsSort,
	})
	if err != nil {
		return domain.DailyReport{}, err
	}

	report := domain.DailyReport{
		Date:            timeutil.CanonicalUTCDate(day),
		AIInsightSource: "fallback",
		AIInsightStatus: "fallback",
	}

	if len(meals) == 0 {
		report.HighestRiskLevel = "unknown"
		report.Summary = fallbackDailyReportSummary(report, 0)
		report.AIInsights = fallbackDailyReportAIInsights(report.Summary)
		if err := uc.saveAndBroadcast(ctx, report); err != nil {
			return domain.DailyReport{}, err
		}
		return report, nil
	}

	var totalSugar float64
	highestRisk := "unknown"
	analyzedMealCount := 0

	for _, meal := range meals {
		report.MealCount++
		if meal.Analysis == nil {
			continue
		}

		analyzedMealCount++
		totalSugar += meal.Analysis.EstimatedSugarGrams
		if highestRisk == "unknown" || compareRisk(meal.Analysis.RiskLevel, highestRisk) > 0 {
			highestRisk = meal.Analysis.RiskLevel
		}
	}

	report.TotalSugarGrams = totalSugar
	report.HighestRiskLevel = highestRisk
	if analyzedMealCount > 0 {
		report.AverageSugarGrams = totalSugar / float64(analyzedMealCount)
	}
	report.Summary = fallbackDailyReportSummary(report, analyzedMealCount)
	report.AIInsights = fallbackDailyReportAIInsights(report.Summary)

	if analyzedMealCount > 0 && uc.interpreter != nil {
		insights, err := uc.interpreter.GenerateInsights(ctx, domain.GenerateDailyReportSummaryInput{
			Report:            report,
			AnalyzedMealCount: analyzedMealCount,
			Meals:             meals,
		})
		if err == nil && strings.TrimSpace(insights.Summary) != "" {
			insights.Summary = strings.TrimSpace(insights.Summary)
			report.Summary = insights.Summary
			report.AIInsights = insights
			report.AIInsightSource = aiInsightSourceName(uc.interpreter)
			report.AIInsightStatus = "completed"
		} else if err != nil {
			zap.L().Warn("daily_report_ai_fallback_used",
				zap.String("report_date", report.Date.Format("2006-01-02")),
				zap.Int("meal_count", report.MealCount),
				zap.Error(err),
			)
		}
	}

	if err := uc.saveAndBroadcast(ctx, report); err != nil {
		return domain.DailyReport{}, err
	}

	return report, nil
}

func fallbackDailyReportSummary(report domain.DailyReport, analyzedMealCount int) string {
	if report.MealCount == 0 {
		return "No meals were recorded for the selected day."
	}
	if analyzedMealCount == 0 {
		return fmt.Sprintf(
			"%d meals logged, but nutrition analysis is not available yet.",
			report.MealCount,
		)
	}
	return fmt.Sprintf(
		"%d meals logged (%d analyzed). Estimated sugar intake %.1fg. Highest meal risk: %s.",
		report.MealCount,
		analyzedMealCount,
		report.TotalSugarGrams,
		report.HighestRiskLevel,
	)
}

func fallbackDailyReportAIInsights(summary string) domain.DailyReportAIInsights {
	return domain.DailyReportAIInsights{
		Summary:         summary,
		TopContributors: []domain.DailyReportTopContributor{},
		Recommendations: []string{},
		PatternSignals:  []string{},
	}
}

func hasCompletedAIInsights(source string, status string) bool {
	return strings.TrimSpace(strings.ToLower(status)) == "completed" &&
		strings.TrimSpace(strings.ToLower(source)) != "" &&
		strings.TrimSpace(strings.ToLower(source)) != "fallback"
}

func aiInsightSourceName(interpreter any) string {
	if provider, ok := interpreter.(aiInsightProvider); ok {
		source := strings.TrimSpace(strings.ToLower(provider.AIInsightProviderName()))
		if source != "" {
			return source
		}
	}
	return "gemini"
}

func compareRisk(left string, right string) int {
	return riskScore(left) - riskScore(right)
}

func riskScore(level string) int {
	switch level {
	case "high":
		return 3
	case "medium":
		return 2
	default:
		return 1
	}
}
