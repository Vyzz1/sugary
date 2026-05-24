package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"sugary/internal/domain"
)

type stubMealRepository struct {
	createFn             func(ctx context.Context, meal domain.Meal) (domain.Meal, error)
	listByDayFn          func(ctx context.Context, day time.Time) ([]domain.Meal, error)
	listRecentDistinctFn func(ctx context.Context, filter domain.RecentMealsFilter) ([]domain.Meal, int64, error)
	getByIDFn            func(ctx context.Context, mealID int64) (domain.Meal, error)
	updateMetaFn         func(ctx context.Context, mealID int64, mealType string, recordedAt time.Time) (domain.Meal, error)
	updateWithAIFn       func(ctx context.Context, meal domain.Meal) (domain.Meal, error)
	updateAnalysisFn     func(ctx context.Context, mealID int64, nutrition domain.Nutrition) (domain.Meal, error)
	softDeleteFn         func(ctx context.Context, mealID int64) error
}

func (s stubMealRepository) Create(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
	return s.createFn(ctx, meal)
}

func (s stubMealRepository) ListByDay(ctx context.Context, day time.Time) ([]domain.Meal, error) {
	return s.listByDayFn(ctx, day)
}

func (s stubMealRepository) ListRecentDistinct(ctx context.Context, filter domain.RecentMealsFilter) ([]domain.Meal, int64, error) {
	if s.listRecentDistinctFn == nil {
		return nil, 0, nil
	}
	return s.listRecentDistinctFn(ctx, filter)
}

func (s stubMealRepository) UpdateAnalysis(ctx context.Context, mealID int64, nutrition domain.Nutrition) (domain.Meal, error) {
	if s.updateAnalysisFn == nil {
		return domain.Meal{}, nil
	}
	return s.updateAnalysisFn(ctx, mealID, nutrition)
}

func (s stubMealRepository) GetByID(ctx context.Context, mealID int64) (domain.Meal, error) {
	if s.getByIDFn == nil {
		return domain.Meal{}, domain.ErrMealNotFound
	}
	return s.getByIDFn(ctx, mealID)
}

func (s stubMealRepository) UpdateMeta(ctx context.Context, mealID int64, mealType string, recordedAt time.Time) (domain.Meal, error) {
	if s.updateMetaFn == nil {
		return domain.Meal{}, nil
	}
	return s.updateMetaFn(ctx, mealID, mealType, recordedAt)
}

func (s stubMealRepository) UpdateWithAnalysis(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
	if s.updateWithAIFn == nil {
		return domain.Meal{}, nil
	}
	return s.updateWithAIFn(ctx, meal)
}

func (s stubMealRepository) SoftDelete(ctx context.Context, mealID int64) error {
	if s.softDeleteFn == nil {
		return nil
	}
	return s.softDeleteFn(ctx, mealID)
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

func TestLogMealExecuteClonesSourceMealWithoutAnalyzer(t *testing.T) {
	t.Parallel()

	analyzeCalled := false
	sourceID := int64(7)
	recordedAt := time.Date(2026, 5, 23, 12, 30, 0, 0, time.UTC)

	uc := NewLogMeal(
		stubMealRepository{
			getByIDFn: func(ctx context.Context, mealID int64) (domain.Meal, error) {
				if mealID != sourceID {
					t.Fatalf("expected source meal ID %d, got %d", sourceID, mealID)
				}
				imageURL := "https://example.com/milk-tea.jpg"
				return domain.Meal{
					ID:             sourceID,
					DishName:       "Milk tea",
					MealType:       domain.MealTypeSnack,
					ImageURL:       &imageURL,
					RecordedAt:     time.Date(2026, 5, 22, 15, 0, 0, 0, time.UTC),
					AnalysisStatus: "completed",
					IsUserEdited:   true,
					Analysis: &domain.Nutrition{
						EstimatedSugarGrams:   35,
						EstimatedCarbsGrams:   54,
						EstimatedProteinGrams: 6,
						EstimatedCalories:     420,
						RiskLevel:             "high",
						Notes:                 []string{"copied estimate"},
					},
				}, nil
			},
			createFn: func(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
				if meal.DishName != "Milk tea" {
					t.Fatalf("expected cloned dish name, got %q", meal.DishName)
				}
				if meal.MealType != domain.MealTypeLunch {
					t.Fatalf("expected overridden meal_type lunch, got %q", meal.MealType)
				}
				if !meal.RecordedAt.Equal(recordedAt) {
					t.Fatalf("expected provided recorded_at, got %s", meal.RecordedAt)
				}
				if meal.Analysis == nil || meal.Analysis.EstimatedSugarGrams != 35 {
					t.Fatalf("expected copied analysis, got %+v", meal.Analysis)
				}
				if !meal.IsUserEdited {
					t.Fatalf("expected cloned is_user_edited flag")
				}
				meal.ID = 8
				return meal, nil
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
				analyzeCalled = true
				return domain.Nutrition{}, nil
			},
		},
	)

	got, err := uc.Execute(context.Background(), domain.LogMealInput{
		SourceMealID: &sourceID,
		MealType:     "LUNCH",
		RecordedAt:   recordedAt,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.ID != 8 {
		t.Fatalf("expected new meal ID 8, got %d", got.ID)
	}
	if analyzeCalled {
		t.Fatalf("expected clone flow not to call analyzer")
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
