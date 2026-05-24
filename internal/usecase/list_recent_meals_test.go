package usecase

import (
	"context"
	"testing"

	"sugary/internal/domain"
)

func TestListRecentMealsExecuteDefaultsLimit(t *testing.T) {
	t.Parallel()

	uc := NewListRecentMeals(stubMealRepository{
		listRecentDistinctFn: func(ctx context.Context, filter domain.RecentMealsFilter) ([]domain.Meal, int64, error) {
			if filter.PageSize != defaultRecentMealsPageSize {
				t.Fatalf("expected default page size %d, got %d", defaultRecentMealsPageSize, filter.PageSize)
			}
			if filter.Page != defaultRecentMealsPage {
				t.Fatalf("expected default page %d, got %d", defaultRecentMealsPage, filter.Page)
			}
			if filter.Sort != defaultRecentMealsSort {
				t.Fatalf("expected default sort %q, got %q", defaultRecentMealsSort, filter.Sort)
			}
			return []domain.Meal{{ID: 1}}, 1, nil
		},
	})

	meals, total, normalized, err := uc.Execute(context.Background(), domain.RecentMealsFilter{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(meals) != 1 {
		t.Fatalf("expected 1 meal, got %d", len(meals))
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if normalized.Sort != defaultRecentMealsSort {
		t.Fatalf("expected normalized sort %q, got %q", defaultRecentMealsSort, normalized.Sort)
	}
}

func TestListRecentMealsExecuteCapsLimit(t *testing.T) {
	t.Parallel()

	uc := NewListRecentMeals(stubMealRepository{
		listRecentDistinctFn: func(ctx context.Context, filter domain.RecentMealsFilter) ([]domain.Meal, int64, error) {
			if filter.PageSize != 100 {
				t.Fatalf("expected capped page size 100, got %d", filter.PageSize)
			}
			if filter.Sort != "name_desc" {
				t.Fatalf("expected sort name_desc, got %q", filter.Sort)
			}
			return nil, 0, nil
		},
	})

	_, _, _, err := uc.Execute(context.Background(), domain.RecentMealsFilter{
		PageSize: 1000,
		Sort:     "name_desc",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
