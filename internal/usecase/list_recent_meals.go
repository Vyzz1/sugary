package usecase

import (
	"context"
	"strings"

	"sugary/internal/domain"
)

const (
	defaultRecentMealsPageSize int32 = 20
	defaultRecentMealsPage     int32 = 1
	maxRecentMealsPageSize     int32 = 100
	defaultRecentMealsSort           = "created_desc"
)

type ListRecentMeals struct {
	mealRepository domain.MealRepository
}

func NewListRecentMeals(mealRepository domain.MealRepository) ListRecentMeals {
	return ListRecentMeals{mealRepository: mealRepository}
}

func (uc ListRecentMeals) Execute(ctx context.Context, filter domain.RecentMealsFilter) ([]domain.Meal, int64, domain.RecentMealsFilter, error) {
	if filter.Page <= 0 {
		filter.Page = defaultRecentMealsPage
	}
	if filter.PageSize <= 0 {
		filter.PageSize = defaultRecentMealsPageSize
	}
	if filter.PageSize > maxRecentMealsPageSize {
		filter.PageSize = maxRecentMealsPageSize
	}

	filter.Query = strings.TrimSpace(filter.Query)
	filter.Sort = normalizeRecentMealsSort(filter.Sort)

	meals, total, err := uc.mealRepository.ListRecentDistinct(ctx, filter)
	return meals, total, filter, err
}

func normalizeRecentMealsSort(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "created_asc", "name_asc", "name_desc":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return defaultRecentMealsSort
	}
}
