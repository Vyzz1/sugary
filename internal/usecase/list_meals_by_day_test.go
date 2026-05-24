package usecase

import (
	"context"
	"testing"
	"time"

	"sugary/internal/domain"
)

func TestListMealsByDayExecuteNormalizesDate(t *testing.T) {
	t.Parallel()

	inputDay := time.Date(2026, 5, 23, 15, 45, 0, 0, time.UTC)
	expectedDay := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)

	uc := NewListMealsByDay(stubMealRepository{
		listByDayFn: func(ctx context.Context, filter domain.MealsByDayFilter) ([]domain.Meal, error) {
			if !filter.Day.Equal(expectedDay) {
				t.Fatalf("expected normalized day %s, got %s", expectedDay, filter.Day)
			}
			if filter.Sort != defaultRecentMealsSort {
				t.Fatalf("expected default sort %q, got %q", defaultRecentMealsSort, filter.Sort)
			}
			return []domain.Meal{{ID: 1}}, nil
		},
	})

	meals, normalized, err := uc.Execute(context.Background(), domain.MealsByDayFilter{Day: inputDay})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(meals) != 1 {
		t.Fatalf("expected 1 meal, got %d", len(meals))
	}
	if !normalized.Day.Equal(expectedDay) {
		t.Fatalf("expected normalizedDay %s, got %s", expectedDay, normalized.Day)
	}
}
