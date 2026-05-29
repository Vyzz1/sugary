package domain

import (
	"context"
	"time"
)

type DailyReport struct {
	Date              time.Time             `json:"date"`
	MealCount         int                   `json:"meal_count"`
	TotalSugarGrams   float64               `json:"total_sugar_grams"`
	AverageSugarGrams float64               `json:"average_sugar_grams"`
	HighestRiskLevel  string                `json:"highest_risk_level"`
	Summary           string                `json:"summary"`
	AIInsightSource   string                `json:"ai_insight_source"`
	AIInsightStatus   string                `json:"ai_insight_status"`
	AIInsights        DailyReportAIInsights `json:"ai_insights"`
}

type GenerateDailyReportSummaryInput struct {
	Report            DailyReport
	AnalyzedMealCount int
	Meals             []Meal
}

type DailyReportAIInsights struct {
	Summary         string                      `json:"summary"`
	TopContributors []DailyReportTopContributor `json:"top_contributors"`
	Recommendations []string                    `json:"recommendations"`
	PatternSignals  []string                    `json:"pattern_signals"`
}

type DailyReportTopContributor struct {
	DishName            string  `json:"dish_name"`
	MealType            string  `json:"meal_type"`
	EstimatedSugarGrams float64 `json:"estimated_sugar_grams"`
	RiskLevel           string  `json:"risk_level"`
}

type DailyReportRepository interface {
	Save(ctx context.Context, report DailyReport) error
	GetByDay(ctx context.Context, day time.Time) (DailyReport, bool, error)
}

type DailyReportInterpreter interface {
	GenerateInsights(ctx context.Context, input GenerateDailyReportSummaryInput) (DailyReportAIInsights, error)
}
