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

type WeeklyReport struct {
	WeekStartDate     time.Time              `json:"week_start_date"`
	WeekEndDate       time.Time              `json:"week_end_date"`
	CreatedAt         time.Time              `json:"created_at"`
	MealCount         int                    `json:"meal_count"`
	AnalyzedMealCount int                    `json:"analyzed_meal_count"`
	TotalSugarGrams   float64                `json:"total_sugar_grams"`
	AverageSugarGrams float64                `json:"average_sugar_grams"`
	HighestRiskLevel  string                 `json:"highest_risk_level"`
	Summary           string                 `json:"summary"`
	DailyBreakdown    []WeeklyReportDaily    `json:"daily_breakdown"`
	AIInsightSource   string                 `json:"ai_insight_source"`
	AIInsightStatus   string                 `json:"ai_insight_status"`
	AIInsights        WeeklyReportAIInsights `json:"ai_insights"`
}

type GenerateDailyReportSummaryInput struct {
	Report            DailyReport
	AnalyzedMealCount int
	Meals             []Meal
}

type GenerateWeeklyReportSummaryInput struct {
	Report WeeklyReport
	Meals  []Meal
}

type DailyReportAIInsights struct {
	Summary         string                      `json:"summary"`
	TopContributors []DailyReportTopContributor `json:"top_contributors"`
	Recommendations []string                    `json:"recommendations"`
	PatternSignals  []string                    `json:"pattern_signals"`
}

type WeeklyReportAIInsights struct {
	Summary         string                       `json:"summary"`
	TopContributors []WeeklyReportTopContributor `json:"top_contributors"`
	Recommendations []string                     `json:"recommendations"`
	PatternSignals  []string                     `json:"pattern_signals"`
}

type DailyReportTopContributor struct {
	DishName            string  `json:"dish_name"`
	MealType            string  `json:"meal_type"`
	EstimatedSugarGrams float64 `json:"estimated_sugar_grams"`
	RiskLevel           string  `json:"risk_level"`
}

type WeeklyReportTopContributor struct {
	DishName            string  `json:"dish_name"`
	MealType            string  `json:"meal_type"`
	EstimatedSugarGrams float64 `json:"estimated_sugar_grams"`
	RiskLevel           string  `json:"risk_level"`
}

type WeeklyReportDaily struct {
	Date              time.Time `json:"date"`
	MealCount         int       `json:"meal_count"`
	AnalyzedMealCount int       `json:"analyzed_meal_count"`
	TotalSugarGrams   float64   `json:"total_sugar_grams"`
	AverageSugarGrams float64   `json:"average_sugar_grams"`
	HighestRiskLevel  string    `json:"highest_risk_level"`
}

type DailyReportRepository interface {
	Save(ctx context.Context, report DailyReport) error
	GetByDay(ctx context.Context, day time.Time) (DailyReport, bool, error)
}

type WeeklyReportRepository interface {
	Save(ctx context.Context, report WeeklyReport) error
	GetByWeekStart(ctx context.Context, weekStart time.Time) (WeeklyReport, bool, error)
}

type DailyReportInterpreter interface {
	GenerateInsights(ctx context.Context, input GenerateDailyReportSummaryInput) (DailyReportAIInsights, error)
}

type WeeklyReportInterpreter interface {
	GenerateInsights(ctx context.Context, input GenerateWeeklyReportSummaryInput) (WeeklyReportAIInsights, error)
}
