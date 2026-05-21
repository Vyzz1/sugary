package domain

import (
	"context"
	"time"
)

type Meal struct {
	ID             int64      `json:"id"`
	DishName       string     `json:"dish_name"`
	ImageURL       *string    `json:"image_url,omitempty"`
	RecordedAt     time.Time  `json:"recorded_at"`
	AnalysisStatus string     `json:"analysis_status"`
	Analysis       *Nutrition `json:"analysis,omitempty"`
}

type Nutrition struct {
	EstimatedSugarGrams float64  `json:"estimated_sugar_grams"`
	EstimatedCalories   int      `json:"estimated_calories"`
	RiskLevel           string   `json:"risk_level"`
	Notes               []string `json:"notes"`
}

type LogMealInput struct {
	DishName   string
	ImageURL   *string
	RecordedAt time.Time
}

type MealRepository interface {
	Create(ctx context.Context, meal Meal) (Meal, error)
	ListByDay(ctx context.Context, day time.Time) ([]Meal, error)
}

type NutritionAnalyzer interface {
	AnalyzeMeal(ctx context.Context, input AnalyzeMealInput) (Nutrition, error)
}

type AnalyzeMealInput struct {
	DishName string
	ImageURL *string
}
