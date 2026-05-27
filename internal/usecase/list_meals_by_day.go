package usecase

import (
	"context"
	"time"

	"sugary/internal/domain"
	"sugary/internal/platform/timeutil"
)

type ListMealsByDay struct {
	mealRepository domain.MealRepository
}

func NewListMealsByDay(mealRepository domain.MealRepository) ListMealsByDay {
	return ListMealsByDay{mealRepository: mealRepository}
}

func (uc ListMealsByDay) Execute(ctx context.Context, filter domain.MealsByDayFilter) ([]domain.Meal, domain.MealsByDayFilter, error) {
	if filter.Day.IsZero() {
		filter.Day = time.Now()
	}

	filter.Day = timeutil.StartOfDay(filter.Day)
	filter.Sort = normalizeRecentMealsSort(filter.Sort)

	meals, err := uc.mealRepository.ListByDay(ctx, filter)
	if err != nil {
		return nil, domain.MealsByDayFilter{}, err
	}

	return meals, filter, nil
}
