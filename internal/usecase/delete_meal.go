package usecase

import (
	"context"

	"sugary/internal/domain"
)

type DeleteMeal struct {
	mealRepository domain.MealRepository
}

func NewDeleteMeal(mealRepository domain.MealRepository) DeleteMeal {
	return DeleteMeal{mealRepository: mealRepository}
}

func (uc DeleteMeal) Execute(ctx context.Context, mealID int64) error {
	if mealID <= 0 {
		return domain.ErrInvalidMealInput
	}
	return uc.mealRepository.SoftDelete(ctx, mealID)
}
