package usecase

import (
	"context"
	"testing"

	"sugary/internal/domain"
)

func TestListMealsExecuteNormalizesFilter(t *testing.T) {
	t.Parallel()

	uc := NewListMeals(
		stubMealRepository{
			listFn: func(ctx context.Context, filter domain.MealListFilter) ([]domain.Meal, int64, error) {
				if filter.Query != "tea" {
					t.Fatalf("expected trimmed query, got %q", filter.Query)
				}
				if filter.MealType != domain.MealTypeSnack {
					t.Fatalf("expected normalized meal_type snack, got %q", filter.MealType)
				}
				if filter.Page != 1 {
					t.Fatalf("expected default page 1, got %d", filter.Page)
				}
				if filter.PageSize != 20 {
					t.Fatalf("expected default page_size 20, got %d", filter.PageSize)
				}
				if filter.SortBy != "estimated_sugar_grams" {
					t.Fatalf("expected sort_by estimated_sugar_grams, got %q", filter.SortBy)
				}
				if filter.SortType != "asc" {
					t.Fatalf("expected sort_type asc, got %q", filter.SortType)
				}
				return []domain.Meal{}, 0, nil
			},
		},
	)

	_, _, _, err := uc.Execute(context.Background(), domain.MealListFilter{
		Query:    " tea ",
		MealType: "SNACK",
		SortBy:   "estimated_sugar_grams",
		SortType: "ASC",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestListMealsExecuteFallsBackToDefaults(t *testing.T) {
	t.Parallel()

	uc := NewListMeals(
		stubMealRepository{
			listFn: func(ctx context.Context, filter domain.MealListFilter) ([]domain.Meal, int64, error) {
				if filter.SortBy != "recorded_at" {
					t.Fatalf("expected default sort_by recorded_at, got %q", filter.SortBy)
				}
				if filter.SortType != "desc" {
					t.Fatalf("expected default sort_type desc, got %q", filter.SortType)
				}
				if filter.PageSize != 100 {
					t.Fatalf("expected capped page_size 100, got %d", filter.PageSize)
				}
				return []domain.Meal{}, 0, nil
			},
		},
	)

	_, _, _, err := uc.Execute(context.Background(), domain.MealListFilter{
		PageSize: 999,
		SortBy:   "bad_column",
		SortType: "bad_type",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
