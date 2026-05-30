package usecase

import (
	"context"
	"strings"

	"sugary/internal/domain"
)

const (
	defaultMealsPageSize = int32(20)
	defaultMealsPage     = int32(1)
	maxMealsPageSize     = int32(100)
	defaultMealsSortBy   = "recorded_at"
	defaultMealsSortType = "desc"
)

var sortableMealColumns = []string{
	"recorded_at",
	"dish_name",
	"meal_type",
	"estimated_sugar_grams",
	"estimated_calories",
}

type ListMeals struct {
	mealRepository domain.MealRepository
}

func NewListMeals(mealRepository domain.MealRepository) ListMeals {
	return ListMeals{mealRepository: mealRepository}
}

func (uc ListMeals) Execute(ctx context.Context, filter domain.MealListFilter) ([]domain.Meal, int64, domain.MealListFilter, error) {
	if filter.Page <= 0 {
		filter.Page = defaultMealsPage
	}
	if filter.PageSize <= 0 {
		filter.PageSize = defaultMealsPageSize
	}
	if filter.PageSize > maxMealsPageSize {
		filter.PageSize = maxMealsPageSize
	}

	filter.Query = strings.TrimSpace(filter.Query)
	filter.MealType = strings.TrimSpace(strings.ToLower(filter.MealType))
	filter.SortBy = normalizeMealSortBy(filter.SortBy)
	filter.SortType = normalizeMealSortType(filter.SortType)

	meals, total, err := uc.mealRepository.List(ctx, filter)
	return meals, total, filter, err
}

func normalizeMealSortBy(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	for _, column := range sortableMealColumns {
		if value == column {
			return value
		}
	}
	return defaultMealsSortBy
}

func normalizeMealSortType(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "asc":
		return "asc"
	default:
		return defaultMealsSortType
	}
}
