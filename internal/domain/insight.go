package domain

import (
	"context"
	"time"
)

const (
	InsightRange7D  = "7d"
	InsightRange30D = "30d"
	InsightRange90D = "90d"
)

type InsightResponse struct {
	Range    InsightRange    `json:"range"`
	Summary  InsightSummary  `json:"summary"`
	Trend    InsightTrend    `json:"trend"`
	Charts   InsightCharts   `json:"charts"`
	Patterns InsightPatterns `json:"patterns"`
}

type InsightRange struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Days      int    `json:"days"`
	RangeType string `json:"range_type"`
}

type InsightSummary struct {
	TotalSugarGrams     float64            `json:"total_sugar_grams"`
	AverageSugarPerDay  float64            `json:"average_sugar_per_day"`
	AverageSugarPerMeal float64            `json:"average_sugar_per_meal"`
	TotalMeals          int64              `json:"total_meals"`
	HighRiskMeals       int64              `json:"high_risk_meals"`
	HighRiskDays        int64              `json:"high_risk_days"`
	WorstDay            *InsightDaySummary `json:"worst_day"`
	BestDay             *InsightDaySummary `json:"best_day"`
}

type InsightDaySummary struct {
	Date            string  `json:"date"`
	TotalSugarGrams float64 `json:"total_sugar_grams"`
	MealCount       int64   `json:"meal_count"`
	RiskLevel       string  `json:"risk_level"`
}

type InsightTrend struct {
	ComparisonLabel string             `json:"comparison_label"`
	CurrentPeriod   InsightPeriodRange `json:"current_period"`
	PreviousPeriod  InsightPeriodRange `json:"previous_period"`
	Sugar           TrendAverageMetric `json:"sugar"`
	HighRiskMeals   TrendCountMetric   `json:"high_risk_meals"`
	MealCount       TrendCountMetric   `json:"meal_count"`
}

type InsightPeriodRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type TrendAverageMetric struct {
	CurrentAverageDailyGrams  float64 `json:"current_avg_daily_grams"`
	PreviousAverageDailyGrams float64 `json:"previous_avg_daily_grams"`
	Direction                 string  `json:"direction"`
	Percent                   float64 `json:"percent"`
}

type TrendCountMetric struct {
	CurrentCount  int64   `json:"current_count"`
	PreviousCount int64   `json:"previous_count"`
	Direction     string  `json:"direction"`
	Percent       float64 `json:"percent"`
}

type InsightCharts struct {
	DailySugar        []DailySugarPoint       `json:"daily_sugar"`
	MealTypeBreakdown []MealTypeBreakdown     `json:"meal_type_breakdown"`
	RiskDistribution  []RiskDistributionPoint `json:"risk_distribution"`
	WeeklySugar       []WeeklySugarPoint      `json:"weekly_sugar"`
	SugarVsCalories   []SugarVsCaloriesPoint  `json:"sugar_vs_calories"`
}

type DailySugarPoint struct {
	Date              string  `json:"date"`
	TotalSugarGrams   float64 `json:"total_sugar_grams"`
	TotalCalories     int64   `json:"total_calories"`
	TotalCarbsGrams   float64 `json:"total_carbs_grams"`
	TotalProteinGrams float64 `json:"total_protein_grams"`
	MealCount         int64   `json:"meal_count"`
	RiskLevel         string  `json:"risk_level"`
	TargetGrams       float64 `json:"target_grams"`
}

type MealTypeBreakdown struct {
	MealType          string  `json:"meal_type"`
	TotalSugarGrams   float64 `json:"total_sugar_grams"`
	MealCount         int64   `json:"meal_count"`
	AverageSugarGrams float64 `json:"average_sugar_grams"`
}

type RiskDistributionPoint struct {
	RiskLevel  string  `json:"risk_level"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

type WeeklySugarPoint struct {
	Week            string  `json:"week"`
	TotalSugarGrams float64 `json:"total_sugar_grams"`
	AveragePerDay   float64 `json:"average_per_day"`
	MealCount       int64   `json:"meal_count"`
	HighRiskMeals   int64   `json:"high_risk_meals"`
}

type SugarVsCaloriesPoint struct {
	MealID     int64     `json:"meal_id"`
	DishName   string    `json:"dish_name"`
	SugarGrams float64   `json:"sugar_grams"`
	Calories   int       `json:"calories"`
	RiskLevel  string    `json:"risk_level"`
	RecordedAt time.Time `json:"recorded_at"`
}

type InsightPatterns struct {
	TopSugarMeals  []TopSugarMeal      `json:"top_sugar_meals"`
	TopSugarDishes []TopSugarDish      `json:"top_sugar_dishes"`
	WorstMealType  *WorstMealTypePoint `json:"worst_meal_type"`
}

type TopSugarMeal struct {
	MealID     int64     `json:"meal_id"`
	DishName   string    `json:"dish_name"`
	SugarGrams float64   `json:"sugar_grams"`
	Calories   int       `json:"calories"`
	MealType   string    `json:"meal_type"`
	RiskLevel  string    `json:"risk_level"`
	RecordedAt time.Time `json:"recorded_at"`
}

type TopSugarDish struct {
	DishName          string  `json:"dish_name"`
	TimesLogged       int64   `json:"times_logged"`
	TotalSugarGrams   float64 `json:"total_sugar_grams"`
	AverageSugarGrams float64 `json:"average_sugar_grams"`
}

type WorstMealTypePoint struct {
	MealType          string  `json:"meal_type"`
	TotalSugarGrams   float64 `json:"total_sugar_grams"`
	AverageSugarGrams float64 `json:"average_sugar_grams"`
}

type InsightPeriod struct {
	RangeType     string
	Days          int
	From          time.Time
	To            time.Time
	PreviousFrom  time.Time
	PreviousTo    time.Time
	FromInclusive time.Time
	ToExclusive   time.Time
	PrevInclusive time.Time
	PrevExclusive time.Time
	Timezone      string
}

type InsightPeriodFilter struct {
	FromInclusive time.Time
	ToExclusive   time.Time
	Timezone      string
}

type InsightPeriodStats struct {
	TotalSugarGrams float64
	MealCount       int64
	HighRiskMeals   int64
}

type InsightRepository interface {
	GetPeriodStats(ctx context.Context, filter InsightPeriodFilter) (InsightPeriodStats, error)
	GetDailySugar(ctx context.Context, filter InsightPeriodFilter) ([]DailySugarPoint, error)
	GetMealTypeBreakdown(ctx context.Context, filter InsightPeriodFilter) ([]MealTypeBreakdown, error)
	GetRiskDistribution(ctx context.Context, filter InsightPeriodFilter) ([]RiskDistributionPoint, error)
	GetWeeklySugar(ctx context.Context, filter InsightPeriodFilter) ([]WeeklySugarPoint, error)
	GetSugarVsCalories(ctx context.Context, filter InsightPeriodFilter) ([]SugarVsCaloriesPoint, error)
	GetTopSugarMeals(ctx context.Context, filter InsightPeriodFilter) ([]TopSugarMeal, error)
	GetTopSugarDishes(ctx context.Context, filter InsightPeriodFilter) ([]TopSugarDish, error)
}
