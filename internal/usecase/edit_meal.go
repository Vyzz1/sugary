package usecase

import (
	"context"
	"strings"

	"sugary/internal/domain"
)

type EditMeal struct {
	mealRepository    domain.MealRepository
	nutritionAnalyzer domain.NutritionAnalyzer
}

func NewEditMeal(mealRepository domain.MealRepository, nutritionAnalyzer domain.NutritionAnalyzer) EditMeal {
	return EditMeal{
		mealRepository:    mealRepository,
		nutritionAnalyzer: nutritionAnalyzer,
	}
}

func (uc EditMeal) Execute(ctx context.Context, input domain.EditMealInput) (domain.Meal, error) {
	if input.MealID <= 0 {
		return domain.Meal{}, domain.ErrInvalidMealInput
	}

	current, err := uc.mealRepository.GetByID(ctx, input.MealID)
	if err != nil {
		return domain.Meal{}, err
	}

	mealType := current.MealType
	if input.MealType != nil {
		normalized := strings.TrimSpace(strings.ToLower(*input.MealType))
		if normalized == "" || !domain.IsValidMealType(normalized) {
			return domain.Meal{}, domain.ErrInvalidMealInput
		}
		mealType = normalized
	}

	recordedAt := current.RecordedAt
	if input.RecordedAt != nil {
		recordedAt = input.RecordedAt.UTC()
	}

	dishName := current.DishName
	if input.DishName != nil {
		trimmed := strings.TrimSpace(*input.DishName)
		if trimmed == "" {
			return domain.Meal{}, domain.ErrInvalidMealInput
		}
		dishName = trimmed
	}

	imageURL := current.ImageURL
	if input.ImageURL != nil {
		trimmed := strings.TrimSpace(*input.ImageURL)
		if trimmed == "" {
			imageURL = nil
		} else {
			imageURL = &trimmed
		}
	}

	changedDishOrImage := dishName != current.DishName || !sameNullableString(imageURL, current.ImageURL)
	changedMetaOnly := mealType != current.MealType || !recordedAt.Equal(current.RecordedAt.UTC())

	if !changedDishOrImage && !changedMetaOnly {
		return domain.Meal{}, domain.ErrNoMealChanges
	}

	if changedDishOrImage {
		nutrition, err := uc.nutritionAnalyzer.AnalyzeMeal(ctx, domain.AnalyzeMealInput{
			DishName: dishName,
			ImageURL: imageURL,
		})
		if err != nil {
			return domain.Meal{}, err
		}

		return uc.mealRepository.UpdateWithAnalysis(ctx, domain.Meal{
			ID:         input.MealID,
			DishName:   dishName,
			MealType:   mealType,
			ImageURL:   imageURL,
			RecordedAt: recordedAt,
			Analysis:   &nutrition,
		})
	}

	return uc.mealRepository.UpdateMeta(ctx, input.MealID, mealType, recordedAt)
}

func sameNullableString(left *string, right *string) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	return *left == *right
}
