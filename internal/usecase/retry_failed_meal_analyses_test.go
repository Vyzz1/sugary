package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"sugary/internal/domain"
)

func TestRetryFailedMealAnalysesExecute(t *testing.T) {
	t.Parallel()

	retryCalled := false
	updateResultCalled := false

	uc := NewRetryFailedMealAnalyses(
		stubMealRepository{
			listRetryableFailedFn: func(ctx context.Context, filter domain.RetryableFailedMealsFilter) ([]domain.Meal, error) {
				if filter.MaxRetryCount != 5 {
					t.Fatalf("expected max retry count 5, got %d", filter.MaxRetryCount)
				}
				return []domain.Meal{
					{
						ID:             7,
						DishName:       "Milk tea",
						AnalysisStatus: domain.AnalysisStatusFailed,
					},
				}, nil
			},
			getByIDFn: func(ctx context.Context, mealID int64) (domain.Meal, error) {
				return domain.Meal{ID: mealID, DishName: "Milk tea"}, nil
			},
			retryFailedAnalysisFn: func(ctx context.Context, mealID int64) (domain.Meal, error) {
				retryCalled = true
				return domain.Meal{
					ID:             mealID,
					DishName:       "Milk tea",
					AnalysisStatus: domain.AnalysisStatusProcessing,
				}, nil
			},
			updateAnalysisResultFn: func(ctx context.Context, mealID int64, nutrition domain.Nutrition) (domain.Meal, error) {
				updateResultCalled = true
				return domain.Meal{
					ID:             mealID,
					AnalysisStatus: domain.AnalysisStatusCompleted,
					Analysis:       &nutrition,
				}, nil
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
				return domain.Nutrition{RiskLevel: "medium"}, nil
			},
		},
		5,
		15*time.Minute,
		10,
	)

	retried, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if retried != 1 {
		t.Fatalf("expected retried count 1, got %d", retried)
	}
	if !retryCalled {
		t.Fatal("expected RetryFailedAnalysis to be called")
	}
	if !updateResultCalled {
		t.Fatal("expected UpdateAnalysisResult to be called")
	}
}

func TestRetryFailedMealAnalysesExecuteSkipsMissingMeals(t *testing.T) {
	t.Parallel()

	uc := NewRetryFailedMealAnalyses(
		stubMealRepository{
			listRetryableFailedFn: func(ctx context.Context, filter domain.RetryableFailedMealsFilter) ([]domain.Meal, error) {
				return []domain.Meal{
					{ID: 9, DishName: "Tea", AnalysisStatus: domain.AnalysisStatusFailed},
				}, nil
			},
			retryFailedAnalysisFn: func(ctx context.Context, mealID int64) (domain.Meal, error) {
				return domain.Meal{}, domain.ErrMealNotFound
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
				t.Fatal("expected analyzer not to run for missing meal")
				return domain.Nutrition{}, nil
			},
		},
		5,
		15*time.Minute,
		10,
	)

	retried, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if retried != 0 {
		t.Fatalf("expected retried count 0, got %d", retried)
	}
}

func TestRetryFailedMealAnalysesExecuteReturnsListError(t *testing.T) {
	t.Parallel()

	uc := NewRetryFailedMealAnalyses(
		stubMealRepository{
			listRetryableFailedFn: func(ctx context.Context, filter domain.RetryableFailedMealsFilter) ([]domain.Meal, error) {
				return nil, errors.New("db down")
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
				return domain.Nutrition{}, nil
			},
		},
		5,
		15*time.Minute,
		10,
	)

	_, err := uc.Execute(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
