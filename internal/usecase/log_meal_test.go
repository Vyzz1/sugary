package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"sugary/internal/domain"
)

type stubMealRepository struct {
	createFn    func(ctx context.Context, meal domain.Meal) (domain.Meal, error)
	listByDayFn func(ctx context.Context, day time.Time) ([]domain.Meal, error)
}

func (s stubMealRepository) Create(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
	return s.createFn(ctx, meal)
}

func (s stubMealRepository) ListByDay(ctx context.Context, day time.Time) ([]domain.Meal, error) {
	return s.listByDayFn(ctx, day)
}

type stubNutritionAnalyzer struct {
	analyzeFn func(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error)
}

func (s stubNutritionAnalyzer) AnalyzeMeal(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
	return s.analyzeFn(ctx, input)
}

func TestLogMealExecute(t *testing.T) {
	t.Parallel()

	expectedNutrition := domain.Nutrition{
		EstimatedSugarGrams: 21.5,
		EstimatedCalories:   420,
		RiskLevel:           "medium",
		Notes:               []string{"sweet sauce likely contributes added sugar"},
	}

	recordedAt := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)

	uc := NewLogMeal(
		stubMealRepository{
			createFn: func(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
				if meal.DishName != "Banh mi" {
					t.Fatalf("expected trimmed dish name, got %q", meal.DishName)
				}
				if meal.MealType != domain.MealTypeUnspecified {
					t.Fatalf("expected default meal_type %q, got %q", domain.MealTypeUnspecified, meal.MealType)
				}
				if meal.Analysis == nil || meal.Analysis.RiskLevel != "medium" {
					t.Fatalf("expected analyzed meal, got %+v", meal.Analysis)
				}
				meal.ID = 1
				return meal, nil
			},
			listByDayFn: func(ctx context.Context, day time.Time) ([]domain.Meal, error) {
				return nil, nil
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
				if input.DishName != "Banh mi" {
					t.Fatalf("expected analyzer input Banh mi, got %q", input.DishName)
				}
				return expectedNutrition, nil
			},
		},
	)

	got, err := uc.Execute(context.Background(), domain.LogMealInput{
		DishName:   "  Banh mi  ",
		RecordedAt: recordedAt,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.ID != 1 {
		t.Fatalf("expected meal ID to be assigned, got %d", got.ID)
	}
	if got.Analysis == nil || got.Analysis.EstimatedSugarGrams != expectedNutrition.EstimatedSugarGrams {
		t.Fatalf("expected analysis %+v, got %+v", expectedNutrition, got.Analysis)
	}
}

func TestLogMealExecuteKeepsProvidedMealType(t *testing.T) {
	t.Parallel()

	uc := NewLogMeal(
		stubMealRepository{
			createFn: func(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
				return meal, nil
			},
			listByDayFn: func(ctx context.Context, day time.Time) ([]domain.Meal, error) {
				return nil, nil
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
				return domain.Nutrition{RiskLevel: "low"}, nil
			},
		},
	)

	got, err := uc.Execute(context.Background(), domain.LogMealInput{
		DishName: "Pho",
		MealType: "LUNCH",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.MealType != domain.MealTypeLunch {
		t.Fatalf("expected meal_type %q, got %q", domain.MealTypeLunch, got.MealType)
	}
}

func TestLogMealExecuteReturnsValidationError(t *testing.T) {
	t.Parallel()

	uc := NewLogMeal(
		stubMealRepository{
			createFn: func(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
				return domain.Meal{}, nil
			},
			listByDayFn: func(ctx context.Context, day time.Time) ([]domain.Meal, error) {
				return nil, nil
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
				return domain.Nutrition{}, nil
			},
		},
	)

	_, err := uc.Execute(context.Background(), domain.LogMealInput{
		DishName: " ",
	})
	if !errors.Is(err, domain.ErrInvalidMealInput) {
		t.Fatalf("expected error %v, got %v", domain.ErrInvalidMealInput, err)
	}
}
