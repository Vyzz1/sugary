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
	IsUserEdited   bool       `json:"is_user_edited"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
	Analysis       *Nutrition `json:"analysis,omitempty"`
}

type Nutrition struct {
	EstimatedSugarGrams   float64  `json:"estimated_sugar_grams"`
	EstimatedCarbsGrams   float64  `json:"estimated_carbs_grams"`
	EstimatedProteinGrams float64  `json:"estimated_protein_grams"`
	EstimatedCalories     int      `json:"estimated_calories"`
	RiskLevel             string   `json:"risk_level"`
	Notes                 []string `json:"notes"`
}

type LogMealInput struct {
	SourceMealID *int64
	DishName     string
	MealType     string
	ImageURL     *string
	RecordedAt   time.Time
}

type EditMealInput struct {
	MealID     int64
	DishName   *string
	MealType   *string
	ImageURL   *string
	RecordedAt *time.Time
}

type EditMealAnalysisInput struct {
	MealID int64
	Nutrition
}

type RecentMealsFilter struct {
	Query    string
	Sort     string
	Page     int32
	PageSize int32
}

type MealsByDayFilter struct {
	Day  time.Time
	Sort string
}

type MealRepository interface {
	Create(ctx context.Context, meal Meal) (Meal, error)
	ListByDay(ctx context.Context, filter MealsByDayFilter) ([]Meal, error)
	ListRecentDistinct(ctx context.Context, filter RecentMealsFilter) ([]Meal, int64, error)
	GetByID(ctx context.Context, mealID int64) (Meal, error)
	UpdateMeta(ctx context.Context, mealID int64, mealType string, recordedAt time.Time) (Meal, error)
	UpdateWithAnalysis(ctx context.Context, meal Meal) (Meal, error)
	UpdateAnalysis(ctx context.Context, mealID int64, nutrition Nutrition) (Meal, error)
	SoftDelete(ctx context.Context, mealID int64) error
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
