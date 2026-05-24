package usecase

import (
	"context"
	"errors"
	"testing"

	"sugary/internal/domain"
)

func TestDeleteMealExecute(t *testing.T) {
	t.Parallel()

	called := false
	uc := NewDeleteMeal(stubMealRepository{
		softDeleteFn: func(ctx context.Context, mealID int64) error {
			called = true
			if mealID != 9 {
				t.Fatalf("expected mealID 9, got %d", mealID)
			}
			return nil
		},
	})

	if err := uc.Execute(context.Background(), 9); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !called {
		t.Fatalf("expected soft delete to be called")
	}
}

func TestDeleteMealExecuteValidation(t *testing.T) {
	t.Parallel()

	uc := NewDeleteMeal(stubMealRepository{})
	err := uc.Execute(context.Background(), 0)
	if !errors.Is(err, domain.ErrInvalidMealInput) {
		t.Fatalf("expected ErrInvalidMealInput, got %v", err)
	}
}
