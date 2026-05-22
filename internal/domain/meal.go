package domain

import (
	"context"
	"strings"
	"time"
)

const (
	MealTypeBreakfast   = "breakfast"
	MealTypeLunch       = "lunch"
	MealTypeDinner      = "dinner"
	MealTypeSnack       = "snack"
	MealTypeUnspecified = "unspecified"
)

type Meal struct {
	ID             int64      `json:"id"`
	DishName       string     `json:"dish_name"`
	MealType       string     `json:"meal_type"`
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
	MealType   string
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

func IsValidMealType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case MealTypeBreakfast, MealTypeLunch, MealTypeDinner, MealTypeSnack, MealTypeUnspecified:
		return true
	default:
		return false
	}
}
