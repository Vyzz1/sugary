package usecase

import (
	"context"
	"testing"
	"time"

	"sugary/internal/domain"
)

func TestEditMealMetaOnlyNoReanalyze(t *testing.T) {
	t.Parallel()

	analyzeCalled := false
	uc := NewEditMeal(
		stubMealRepository{
			getByIDFn: func(ctx context.Context, mealID int64) (domain.Meal, error) {
				return domain.Meal{
					ID:         mealID,
					DishName:   "Pho",
					MealType:   domain.MealTypeBreakfast,
					RecordedAt: time.Date(2026, 5, 22, 7, 0, 0, 0, time.UTC),
				}, nil
			},
			updateMetaFn: func(ctx context.Context, mealID int64, mealType string, recordedAt time.Time) (domain.Meal, error) {
				return domain.Meal{ID: mealID, MealType: mealType, RecordedAt: recordedAt}, nil
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
				analyzeCalled = true
				return domain.Nutrition{}, nil
			},
		},
	)

	newType := "lunch"
	newTime := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	meal, err := uc.Execute(context.Background(), domain.EditMealInput{
		MealID:     1,
		MealType:   &newType,
		RecordedAt: &newTime,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if analyzeCalled {
		t.Fatalf("AI should not be called for meal_type/recorded_at changes only")
	}
	if meal.MealType != domain.MealTypeLunch {
		t.Fatalf("expected meal_type lunch, got %s", meal.MealType)
	}
}

func TestEditMealDishNameReanalyzes(t *testing.T) {
	t.Parallel()

	analyzeCalled := false
	uc := NewEditMeal(
		stubMealRepository{
			getByIDFn: func(ctx context.Context, mealID int64) (domain.Meal, error) {
				return domain.Meal{
					ID:         mealID,
					DishName:   "Pho",
					MealType:   domain.MealTypeLunch,
					RecordedAt: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
				}, nil
			},
			updateWithAIFn: func(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
				return meal, nil
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
				analyzeCalled = true
				return domain.Nutrition{EstimatedSugarGrams: 10}, nil
			},
		},
	)

	newDish := "Com tam"
	_, err := uc.Execute(context.Background(), domain.EditMealInput{
		MealID:   1,
		DishName: &newDish,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !analyzeCalled {
		t.Fatalf("AI should be called when dish_name changes")
	}
}
