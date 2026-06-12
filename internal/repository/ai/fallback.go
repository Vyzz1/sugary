package ai

import (
	"context"

	"go.uber.org/zap"

	"sugary/internal/domain"
)

type FallbackNutritionAnalyzer struct {
	primary  domain.NutritionAnalyzer
	fallback domain.NutritionAnalyzer
}

func NewFallbackNutritionAnalyzer(primary domain.NutritionAnalyzer, fallback domain.NutritionAnalyzer) FallbackNutritionAnalyzer {
	return FallbackNutritionAnalyzer{primary: primary, fallback: fallback}
}

func (a FallbackNutritionAnalyzer) AnalyzeMeal(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
	nutrition, err := a.primary.AnalyzeMeal(ctx, input)
	if err == nil {
		return nutrition, nil
	}

	zap.L().Warn("nutrition_ai_primary_failed_fallback_used", zap.Error(err))
	return a.fallback.AnalyzeMeal(ctx, input)
}

type FallbackDailyReportInterpreter struct {
	primary  domain.DailyReportInterpreter
	fallback domain.DailyReportInterpreter
}

type FallbackWeeklyReportInterpreter struct {
	primary  domain.WeeklyReportInterpreter
	fallback domain.WeeklyReportInterpreter
}

func NewFallbackDailyReportInterpreter(primary domain.DailyReportInterpreter, fallback domain.DailyReportInterpreter) FallbackDailyReportInterpreter {
	return FallbackDailyReportInterpreter{primary: primary, fallback: fallback}
}

func NewFallbackWeeklyReportInterpreter(primary domain.WeeklyReportInterpreter, fallback domain.WeeklyReportInterpreter) FallbackWeeklyReportInterpreter {
	return FallbackWeeklyReportInterpreter{primary: primary, fallback: fallback}
}

func (i FallbackDailyReportInterpreter) GenerateInsights(ctx context.Context, input domain.GenerateDailyReportSummaryInput) (domain.DailyReportAIInsights, error) {
	insights, err := i.primary.GenerateInsights(ctx, input)
	if err == nil {
		return insights, nil
	}

	zap.L().Warn("daily_report_ai_primary_failed_fallback_used", zap.Error(err))
	return i.fallback.GenerateInsights(ctx, input)
}

func (i FallbackWeeklyReportInterpreter) GenerateInsights(ctx context.Context, input domain.GenerateWeeklyReportSummaryInput) (domain.WeeklyReportAIInsights, error) {
	insights, err := i.primary.GenerateInsights(ctx, input)
	if err == nil {
		return insights, nil
	}

	zap.L().Warn("weekly_report_ai_primary_failed_fallback_used", zap.Error(err))
	return i.fallback.GenerateInsights(ctx, input)
}
