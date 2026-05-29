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

	meals, err := uc.mealRepository.ListByDay(ctx, domain.MealsByDayFilter{
		Day:  day,
		Sort: defaultRecentMealsSort,
	})
	if err != nil {
		return domain.DailyReport{}, err
	}

	report := domain.DailyReport{
		Date: timeutil.CanonicalUTCDate(day),
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
