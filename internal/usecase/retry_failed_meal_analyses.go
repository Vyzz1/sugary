package usecase

import (
	"context"
	"time"

	"go.uber.org/zap"

	"sugary/internal/domain"
)

type RetryFailedMealAnalyses struct {
	mealRepository    domain.MealRepository
	nutritionAnalyzer domain.NutritionAnalyzer
	publisher         MealAnalysisPublisher
	maxRetryCount     int32
	cooldown          time.Duration
	batchSize         int32
	now               func() time.Time
}

func NewRetryFailedMealAnalyses(
	mealRepository domain.MealRepository,
	nutritionAnalyzer domain.NutritionAnalyzer,
	maxRetryCount int32,
	cooldown time.Duration,
	batchSize int32,
) RetryFailedMealAnalyses {
	return RetryFailedMealAnalyses{
		mealRepository:    mealRepository,
		nutritionAnalyzer: nutritionAnalyzer,
		maxRetryCount:     maxRetryCount,
		cooldown:          cooldown,
		batchSize:         batchSize,
		now:               time.Now,
	}
}

func (uc RetryFailedMealAnalyses) WithPublisher(pub MealAnalysisPublisher) RetryFailedMealAnalyses {
	uc.publisher = pub
	return uc
}

func (uc RetryFailedMealAnalyses) Execute(ctx context.Context) (int, error) {
	before := uc.now().UTC().Add(-uc.cooldown)
	meals, err := uc.mealRepository.ListRetryableFailed(ctx, domain.RetryableFailedMealsFilter{
		Before:        before,
		Limit:         uc.batchSize,
		MaxRetryCount: uc.maxRetryCount,
	})
	if err != nil {
		return 0, err
	}

	if len(meals) == 0 {
		return 0, nil
	}

	runner := logMealAnalysisRunner(uc.mealRepository, uc.nutritionAnalyzer, uc.publisher)
	retried := 0

	for _, meal := range meals {
		queuedMeal, err := uc.mealRepository.RetryFailedAnalysis(ctx, meal.ID)
		if err != nil {
			if err == domain.ErrMealNotFound {
				zap.L().Info("meal_analysis_retry_skip_missing",
					zap.Int64("meal_id", meal.ID),
				)
				continue
			}
			return retried, err
		}

		retried++
		runner.run(ctx, queuedMeal)
	}

	return retried, nil
}
