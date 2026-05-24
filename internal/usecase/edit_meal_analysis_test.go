package usecase

import (
	"context"
	"errors"
	"testing"

	"sugary/internal/domain"
)

func TestEditMealAnalysisExecute(t *testing.T) {
	t.Parallel()

	uc := NewEditMealAnalysis(stubMealRepository{
		updateAnalysisFn: func(ctx context.Context, mealID int64, nutrition domain.Nutrition) (domain.Meal, error) {
			if mealID != 3 {
				t.Fatalf("expected mealID 3, got %d", mealID)
			}
			return domain.Meal{ID: mealID, Analysis: &nutrition, IsUserEdited: true}, nil
		},
	})

	got, err := uc.Execute(context.Background(), domain.EditMealAnalysisInput{
		MealID: 3,
		Nutrition: domain.Nutrition{
			EstimatedSugarGrams:   21,
			EstimatedCarbsGrams:   33,
			EstimatedProteinGrams: 9,
			EstimatedCalories:     380,
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !got.IsUserEdited {
		t.Fatalf("expected is_user_edited true")
	}
}

func TestEditMealAnalysisExecuteValidation(t *testing.T) {
	t.Parallel()
	uc := NewEditMealAnalysis(stubMealRepository{})

	_, err := uc.Execute(context.Background(), domain.EditMealAnalysisInput{MealID: 0})
	if !errors.Is(err, domain.ErrInvalidMealInput) {
		t.Fatalf("expected ErrInvalidMealInput, got %v", err)
	}
}
