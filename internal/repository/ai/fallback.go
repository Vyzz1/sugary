package ai

import (
	"context"
	"strings"
	"sync"

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
	primary      domain.DailyReportInterpreter
	fallback     domain.DailyReportInterpreter
	mu           sync.RWMutex
	lastProvider string
}

type FallbackWeeklyReportInterpreter struct {
	primary      domain.WeeklyReportInterpreter
	fallback     domain.WeeklyReportInterpreter
	mu           sync.RWMutex
	lastProvider string
}

func NewFallbackDailyReportInterpreter(primary domain.DailyReportInterpreter, fallback domain.DailyReportInterpreter) *FallbackDailyReportInterpreter {
	return &FallbackDailyReportInterpreter{primary: primary, fallback: fallback}
}

func NewFallbackWeeklyReportInterpreter(primary domain.WeeklyReportInterpreter, fallback domain.WeeklyReportInterpreter) *FallbackWeeklyReportInterpreter {
	return &FallbackWeeklyReportInterpreter{primary: primary, fallback: fallback}
}

func (i *FallbackDailyReportInterpreter) AIInsightProviderName() string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	if i.lastProvider != "" {
		return i.lastProvider
	}
	return "fallback"
}

func (i *FallbackWeeklyReportInterpreter) AIInsightProviderName() string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	if i.lastProvider != "" {
		return i.lastProvider
	}
	return "fallback"
}

func (i *FallbackDailyReportInterpreter) GenerateInsights(ctx context.Context, input domain.GenerateDailyReportSummaryInput) (domain.DailyReportAIInsights, error) {
	insights, err := i.primary.GenerateInsights(ctx, input)
	if err == nil {
		i.setLastProvider(providerName(i.primary))
		return insights, nil
	}

	zap.L().Warn("daily_report_ai_primary_failed_fallback_used", zap.Error(err))
	insights, err = i.fallback.GenerateInsights(ctx, input)
	if err == nil {
		i.setLastProvider(providerName(i.fallback))
	}
	return insights, err
}

func (i *FallbackWeeklyReportInterpreter) GenerateInsights(ctx context.Context, input domain.GenerateWeeklyReportSummaryInput) (domain.WeeklyReportAIInsights, error) {
	insights, err := i.primary.GenerateInsights(ctx, input)
	if err == nil {
		i.setLastProvider(providerName(i.primary))
		return insights, nil
	}

	zap.L().Warn("weekly_report_ai_primary_failed_fallback_used", zap.Error(err))
	insights, err = i.fallback.GenerateInsights(ctx, input)
	if err == nil {
		i.setLastProvider(providerName(i.fallback))
	}
	return insights, err
}

func (i *FallbackDailyReportInterpreter) setLastProvider(provider string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.lastProvider = provider
}

func (i *FallbackWeeklyReportInterpreter) setLastProvider(provider string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.lastProvider = provider
}

func providerName(interpreter any) string {
	provider, ok := interpreter.(interface {
		AIInsightProviderName() string
	})
	if !ok {
		return "gemini"
	}
	name := strings.TrimSpace(strings.ToLower(provider.AIInsightProviderName()))
	if name == "" {
		return "gemini"
	}
	return name
}
