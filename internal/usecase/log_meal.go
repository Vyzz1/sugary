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
	dishName := strings.TrimSpace(input.DishName)
	if dishName == "" {
		return domain.Meal{}, domain.ErrInvalidMealInput
	}

	recordedAt := input.RecordedAt
	if recordedAt.IsZero() {
		recordedAt = time.Now().UTC()
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
