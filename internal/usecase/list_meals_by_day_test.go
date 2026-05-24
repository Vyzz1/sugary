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
		listByDayFn: func(ctx context.Context, day time.Time) ([]domain.Meal, error) {
			if !day.Equal(expectedDay) {
				t.Fatalf("expected normalized day %s, got %s", expectedDay, day)
			}
			return []domain.Meal{{ID: 1}}, nil
		},
	})

	meals, normalizedDay, err := uc.Execute(context.Background(), inputDay)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(meals) != 1 {
		t.Fatalf("expected 1 meal, got %d", len(meals))
	}
	if !normalizedDay.Equal(expectedDay) {
		t.Fatalf("expected normalizedDay %s, got %s", expectedDay, normalizedDay)
	}
}
