package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"sugary/internal/domain"
	"sugary/internal/platform/timeutil"
)

type weeklyReportPush struct {
	Type   string               `json:"type"`
	Status string               `json:"status"`
	Data   *domain.WeeklyReport `json:"data,omitempty"`
	Error  *analysisPushErr     `json:"error,omitempty"`
}

type CompileWeeklyReport struct {
	mealRepository         domain.MealRepository
	weeklyReportRepository domain.WeeklyReportRepository
	interpreter            domain.WeeklyReportInterpreter
	publisher              DailyReportPublisher
	now                    func() time.Time
}

func NewCompileWeeklyReport(
	mealRepository domain.MealRepository,
	weeklyReportRepository domain.WeeklyReportRepository,
	interpreter domain.WeeklyReportInterpreter,
) CompileWeeklyReport {
	return CompileWeeklyReport{
		mealRepository:         mealRepository,
		weeklyReportRepository: weeklyReportRepository,
		interpreter:            interpreter,
		now:                    time.Now,
	}
}

func (uc CompileWeeklyReport) WithPublisher(publisher DailyReportPublisher) CompileWeeklyReport {
	uc.publisher = publisher
	return uc
}

func (uc CompileWeeklyReport) broadcastReport(msg weeklyReportPush) {
	if uc.publisher == nil {
		return
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		zap.L().Error("weekly_report_push_marshal_failed",
			zap.String("job_name", "weekly_report"),
			zap.Error(err),
		)
		return
	}

	uc.publisher.Broadcast(payload)
}

func (uc CompileWeeklyReport) saveAndBroadcast(ctx context.Context, report domain.WeeklyReport) error {
	if err := uc.weeklyReportRepository.Save(ctx, report); err != nil {
		return err
	}

	uc.broadcastReport(weeklyReportPush{
		Type:   "weekly_report",
		Status: "completed",
		Data:   &report,
	})

	return nil
}

func (uc CompileWeeklyReport) Execute(ctx context.Context, day time.Time) (domain.WeeklyReport, error) {
	if day.IsZero() {
		return domain.WeeklyReport{}, domain.ErrInvalidDate
	}

	weekStart := timeutil.StartOfWeek(day)
	weekEndExclusive := weekStart.AddDate(0, 0, 7)
	weekEndDate := weekStart.AddDate(0, 0, 6)

	existing, found, err := uc.weeklyReportRepository.GetByWeekStart(ctx, weekStart)
	if err != nil {
		return domain.WeeklyReport{}, err
	}
	if found && hasCompletedAIInsights(existing.AIInsightSource, existing.AIInsightStatus) {
		zap.L().Info("weekly_report_compile_skipped_existing_ai_completed",
			zap.String("week_start_date", existing.WeekStartDate.Format(time.DateOnly)),
			zap.String("week_end_date", existing.WeekEndDate.Format(time.DateOnly)),
			zap.String("ai_insight_source", existing.AIInsightSource),
			zap.String("ai_insight_status", existing.AIInsightStatus),
		)
		return existing, nil
	}
	if found {
		zap.L().Info("weekly_report_compile_reprocessing_existing_fallback",
			zap.String("week_start_date", existing.WeekStartDate.Format(time.DateOnly)),
			zap.String("week_end_date", existing.WeekEndDate.Format(time.DateOnly)),
			zap.String("ai_insight_source", existing.AIInsightSource),
			zap.String("ai_insight_status", existing.AIInsightStatus),
		)
	}

	createdAt := uc.now().UTC()
	if found && !existing.CreatedAt.IsZero() {
		createdAt = existing.CreatedAt
	}

	report := domain.WeeklyReport{
		WeekStartDate:    timeutil.CanonicalUTCDate(weekStart),
		WeekEndDate:      timeutil.CanonicalUTCDate(weekEndDate),
		CreatedAt:        createdAt,
		HighestRiskLevel: "unknown",
		DailyBreakdown:   emptyWeeklyBreakdown(weekStart),
		AIInsightSource:  "fallback",
		AIInsightStatus:  "fallback",
	}

	meals, err := uc.listWeekMeals(ctx, weekStart, weekEndExclusive)
	if err != nil {
		return domain.WeeklyReport{}, err
	}

	if len(meals) == 0 {
		report.Summary = fallbackWeeklyReportSummary(report)
		report.AIInsights = fallbackWeeklyReportAIInsights(report.Summary)
		if err := uc.saveAndBroadcast(ctx, report); err != nil {
			return domain.WeeklyReport{}, err
		}
		return report, nil
	}

	dailyByDate := make(map[string]*domain.WeeklyReportDaily, len(report.DailyBreakdown))
	for i := range report.DailyBreakdown {
		daily := &report.DailyBreakdown[i]
		dailyByDate[daily.Date.Format(time.DateOnly)] = daily
	}

	for _, meal := range meals {
		report.MealCount++
		localDate := timeutil.CanonicalUTCDate(meal.RecordedAt.In(weekStart.Location())).Format(time.DateOnly)
		daily := dailyByDate[localDate]
		if daily == nil {
			continue
		}
		daily.MealCount++

		if meal.Analysis == nil {
			continue
		}

		report.AnalyzedMealCount++
		report.TotalSugarGrams += meal.Analysis.EstimatedSugarGrams
		if report.HighestRiskLevel == "unknown" || compareRisk(meal.Analysis.RiskLevel, report.HighestRiskLevel) > 0 {
			report.HighestRiskLevel = meal.Analysis.RiskLevel
		}

		daily.AnalyzedMealCount++
		daily.TotalSugarGrams += meal.Analysis.EstimatedSugarGrams
		if daily.HighestRiskLevel == "unknown" || compareRisk(meal.Analysis.RiskLevel, daily.HighestRiskLevel) > 0 {
			daily.HighestRiskLevel = meal.Analysis.RiskLevel
		}
	}

	if report.AnalyzedMealCount > 0 {
		report.AverageSugarGrams = report.TotalSugarGrams / float64(report.AnalyzedMealCount)
	}
	for i := range report.DailyBreakdown {
		if report.DailyBreakdown[i].AnalyzedMealCount > 0 {
			report.DailyBreakdown[i].AverageSugarGrams = report.DailyBreakdown[i].TotalSugarGrams / float64(report.DailyBreakdown[i].AnalyzedMealCount)
		}
	}

	report.Summary = fallbackWeeklyReportSummary(report)
	report.AIInsights = fallbackWeeklyReportAIInsights(report.Summary)

	if report.AnalyzedMealCount > 0 && uc.interpreter != nil {
		insights, err := uc.interpreter.GenerateInsights(ctx, domain.GenerateWeeklyReportSummaryInput{
			Report: report,
			Meals:  meals,
		})
		if err == nil && strings.TrimSpace(insights.Summary) != "" {
			insights.Summary = strings.TrimSpace(insights.Summary)
			report.Summary = insights.Summary
			report.AIInsights = insights
			report.AIInsightSource = "gemini"
			report.AIInsightStatus = "completed"
		} else if err != nil {
			zap.L().Warn("weekly_report_ai_fallback_used",
				zap.String("week_start_date", report.WeekStartDate.Format(time.DateOnly)),
				zap.Int("meal_count", report.MealCount),
				zap.Error(err),
			)
		}
	}

	if err := uc.saveAndBroadcast(ctx, report); err != nil {
		return domain.WeeklyReport{}, err
	}

	return report, nil
}

func (uc CompileWeeklyReport) listWeekMeals(ctx context.Context, start time.Time, end time.Time) ([]domain.Meal, error) {
	const pageSize = int32(100)

	meals := make([]domain.Meal, 0)
	for page := int32(1); ; page++ {
		batch, total, err := uc.mealRepository.List(ctx, domain.MealListFilter{
			StartAt:  &start,
			EndAt:    &end,
			Page:     page,
			PageSize: pageSize,
			SortBy:   defaultMealsSortBy,
			SortType: "asc",
		})
		if err != nil {
			return nil, err
		}
		meals = append(meals, batch...)
		if len(batch) == 0 || int64(len(meals)) >= total {
			break
		}
	}

	return meals, nil
}

func emptyWeeklyBreakdown(weekStart time.Time) []domain.WeeklyReportDaily {
	breakdown := make([]domain.WeeklyReportDaily, 0, 7)
	for i := 0; i < 7; i++ {
		breakdown = append(breakdown, domain.WeeklyReportDaily{
			Date:             timeutil.CanonicalUTCDate(weekStart.AddDate(0, 0, i)),
			HighestRiskLevel: "unknown",
		})
	}
	return breakdown
}

func fallbackWeeklyReportSummary(report domain.WeeklyReport) string {
	if report.MealCount == 0 {
		return "No meals were recorded for the selected week."
	}
	if report.AnalyzedMealCount == 0 {
		return fmt.Sprintf(
			"%d meals logged this week, but nutrition analysis is not available yet.",
			report.MealCount,
		)
	}
	return fmt.Sprintf(
		"%d meals logged this week (%d analyzed). Estimated sugar intake %.1fg. Highest meal risk: %s.",
		report.MealCount,
		report.AnalyzedMealCount,
		report.TotalSugarGrams,
		report.HighestRiskLevel,
	)
}

func fallbackWeeklyReportAIInsights(summary string) domain.WeeklyReportAIInsights {
	return domain.WeeklyReportAIInsights{
		Summary:         summary,
		TopContributors: []domain.WeeklyReportTopContributor{},
		Recommendations: []string{},
		PatternSignals:  []string{},
	}
}
