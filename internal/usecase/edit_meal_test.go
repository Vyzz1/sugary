package usecase

import (
	"context"
	"errors"
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

	analyzeCalled := make(chan struct{}, 1)
	updateResultCalled := make(chan struct{}, 1)
	broadcastCh := make(chan []byte, 1)
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
			updateForReanalysisFn: func(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
				meal.AnalysisStatus = domain.AnalysisStatusProcessing
				return meal, nil
			},
			updateAnalysisResultFn: func(ctx context.Context, mealID int64, nutrition domain.Nutrition) (domain.Meal, error) {
				updateResultCalled <- struct{}{}
				return domain.Meal{
					ID:             mealID,
					DishName:       "Com tam",
					AnalysisStatus: domain.AnalysisStatusCompleted,
					Analysis:       &nutrition,
				}, nil
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
				analyzeCalled <- struct{}{}
				return domain.Nutrition{EstimatedSugarGrams: 10}, nil
			},
		},
	).WithPublisher(&capturingPublisher{ch: broadcastCh})

	newDish := "Com tam"
	meal, err := uc.Execute(context.Background(), domain.EditMealInput{
		MealID:   1,
		DishName: &newDish,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if meal.AnalysisStatus != domain.AnalysisStatusProcessing {
		t.Fatalf("expected processing status, got %q", meal.AnalysisStatus)
	}

	select {
	case <-analyzeCalled:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected async AI call when dish_name changes")
	}

	select {
	case <-updateResultCalled:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected async analysis result update")
	}

	select {
	case <-broadcastCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected websocket broadcast after async reanalysis")
	}
}

func TestEditMealRunAnalysisWithRetryDeletedMealSkipsFailureBroadcast(t *testing.T) {
	t.Parallel()

	broadcastCh := make(chan []byte, 1)
	attempts := 0

	uc := NewEditMeal(
		stubMealRepository{
			updateAnalysisStatusFn: func(ctx context.Context, mealID int64, status string) error {
				return domain.ErrMealNotFound
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
				attempts++
				return domain.Nutrition{}, errors.New("persistent error")
			},
		},
	).WithPublisher(&capturingPublisher{ch: broadcastCh})

	uc.runAnalysisWithRetry(context.Background(), domain.Meal{
		ID:             9,
		DishName:       "Tra sua",
		AnalysisStatus: domain.AnalysisStatusProcessing,
	})

	if attempts != analysisMaxRetries {
		t.Fatalf("expected %d analyzer calls, got %d", analysisMaxRetries, attempts)
	}

	select {
	case msg := <-broadcastCh:
		t.Fatalf("expected no broadcast for deleted meal, got %s", string(msg))
	default:
	}
}
