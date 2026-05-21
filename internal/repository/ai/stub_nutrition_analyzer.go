package ai

import (
	"context"
	"strings"

	"sugary/internal/domain"
)

type StubNutritionAnalyzer struct{}

func NewStubNutritionAnalyzer() StubNutritionAnalyzer {
	return StubNutritionAnalyzer{}
}

func (StubNutritionAnalyzer) AnalyzeMeal(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
	_ = ctx

	name := strings.ToLower(input.DishName)
	switch {
	case strings.Contains(name, "milk tea"), strings.Contains(name, "cake"), strings.Contains(name, "soda"):
		return domain.Nutrition{
			EstimatedSugarGrams: 35,
			EstimatedCalories:   420,
			RiskLevel:           "high",
			Notes:               []string{"high-sugar drink or dessert pattern detected"},
		}, nil
	case strings.Contains(name, "pho"), strings.Contains(name, "salad"), strings.Contains(name, "egg"):
		return domain.Nutrition{
			EstimatedSugarGrams: 5,
			EstimatedCalories:   320,
			RiskLevel:           "low",
			Notes:               []string{"generally lower added sugar unless sweet sauces are included"},
		}, nil
	default:
		return domain.Nutrition{
			EstimatedSugarGrams: 18,
			EstimatedCalories:   380,
			RiskLevel:           "medium",
			Notes:               []string{"estimated from dish name; image-aware AI can refine this later"},
		}, nil
	}
}
