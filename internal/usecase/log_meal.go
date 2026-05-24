package usecase

import (
	"context"
	"strings"
	"time"

	"sugary/internal/domain"
)

type LogMeal struct {
	mealRepository    domain.MealRepository
	nutritionAnalyzer domain.NutritionAnalyzer
}

func NewLogMeal(mealRepository domain.MealRepository, nutritionAnalyzer domain.NutritionAnalyzer) LogMeal {
	return LogMeal{
		mealRepository:    mealRepository,
		nutritionAnalyzer: nutritionAnalyzer,
	}
}

func (uc LogMeal) Execute(ctx context.Context, input domain.LogMealInput) (domain.Meal, error) {
	recordedAt := input.RecordedAt
	if recordedAt.IsZero() {
		recordedAt = time.Now().UTC()
	}

	if input.SourceMealID != nil {
		return uc.cloneMeal(ctx, *input.SourceMealID, input.MealType, recordedAt)
	}

	dishName := strings.TrimSpace(input.DishName)
	if dishName == "" {
		return domain.Meal{}, domain.ErrInvalidMealInput
	}

	mealType := strings.TrimSpace(strings.ToLower(input.MealType))
	if mealType == "" {
		mealType = domain.MealTypeUnspecified
	}

	nutrition, err := uc.nutritionAnalyzer.AnalyzeMeal(ctx, domain.AnalyzeMealInput{
		DishName: dishName,
		ImageURL: input.ImageURL,
	})
	if err != nil {
		return domain.Meal{}, err
	}

	meal := domain.Meal{
		DishName:       dishName,
		MealType:       mealType,
		ImageURL:       input.ImageURL,
		RecordedAt:     recordedAt,
		AnalysisStatus: "completed",
		Analysis:       &nutrition,
	}

	return uc.mealRepository.Create(ctx, meal)
}

func (uc LogMeal) cloneMeal(ctx context.Context, sourceMealID int64, mealType string, recordedAt time.Time) (domain.Meal, error) {
	if sourceMealID <= 0 {
		return domain.Meal{}, domain.ErrInvalidMealInput
	}

	source, err := uc.mealRepository.GetByID(ctx, sourceMealID)
	if err != nil {
		return domain.Meal{}, err
	}
	if source.Analysis == nil {
		return domain.Meal{}, domain.ErrInvalidMealInput
	}

	normalizedMealType := strings.TrimSpace(strings.ToLower(mealType))
	if normalizedMealType == "" {
		normalizedMealType = source.MealType
	}
	if !domain.IsValidMealType(normalizedMealType) {
		return domain.Meal{}, domain.ErrInvalidMealInput
	}

	nutrition := *source.Analysis
	meal := domain.Meal{
		DishName:       source.DishName,
		MealType:       normalizedMealType,
		ImageURL:       source.ImageURL,
		RecordedAt:     recordedAt,
		AnalysisStatus: source.AnalysisStatus,
		IsUserEdited:   source.IsUserEdited,
		Analysis:       &nutrition,
	}

	return uc.mealRepository.Create(ctx, meal)
}
