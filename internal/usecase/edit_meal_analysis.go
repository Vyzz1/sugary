package usecase

import (
	"context"

	"sugary/internal/domain"
)

type EditMealAnalysis struct {
	mealRepository domain.MealRepository
}

func NewEditMealAnalysis(mealRepository domain.MealRepository) EditMealAnalysis {
	return EditMealAnalysis{mealRepository: mealRepository}
}

func (uc EditMealAnalysis) Execute(ctx context.Context, input domain.EditMealAnalysisInput) (domain.Meal, error) {
	if input.MealID <= 0 {
		return domain.Meal{}, domain.ErrInvalidMealInput
	}
	if input.EstimatedSugarGrams < 0 || input.EstimatedCarbsGrams < 0 || input.EstimatedProteinGrams < 0 || input.EstimatedCalories < 0 {
		return domain.Meal{}, domain.ErrInvalidNutrition
	}

	return uc.mealRepository.UpdateAnalysis(ctx, input.MealID, domain.Nutrition{
		EstimatedSugarGrams:   input.EstimatedSugarGrams,
		EstimatedCarbsGrams:   input.EstimatedCarbsGrams,
		EstimatedProteinGrams: input.EstimatedProteinGrams,
		EstimatedCalories:     input.EstimatedCalories,
	})
}
