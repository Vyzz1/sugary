package usecase

import (
	"context"
	"time"

	"sugary/internal/domain"
)

type ListMealsByDay struct {
	mealRepository domain.MealRepository
}

func NewListMealsByDay(mealRepository domain.MealRepository) ListMealsByDay {
	return ListMealsByDay{mealRepository: mealRepository}
}

func (uc ListMealsByDay) Execute(ctx context.Context, day time.Time) ([]domain.Meal, time.Time, error) {
	if day.IsZero() {
		day = time.Now().UTC()
	}

	normalizedDay := startOfDayUTC(day)
	meals, err := uc.mealRepository.ListByDay(ctx, normalizedDay)
	if err != nil {
		return nil, time.Time{}, err
	}

	return meals, normalizedDay, nil
}
