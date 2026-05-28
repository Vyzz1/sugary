package usecase

import (
	"context"
	"strings"
	"time"

	"sugary/internal/domain"
)

// LogMeal records a new meal.
//
// When a publisher is configured (via WithPublisher), the AI nutrition
// analysis runs asynchronously: the meal is saved immediately with
// analysis_status = "processing" and the result is pushed via WebSocket
// once the goroutine finishes (with up to analysisMaxRetries attempts).
//
// When no publisher is set (publisher == nil), the analysis still runs
// in the background goroutine; only the broadcast step is skipped.
type LogMeal struct {
	mealRepository    domain.MealRepository
	nutritionAnalyzer domain.NutritionAnalyzer
	publisher         MealAnalysisPublisher // optional – nil is safe
}

func NewLogMeal(mealRepository domain.MealRepository, nutritionAnalyzer domain.NutritionAnalyzer) LogMeal {
	return LogMeal{
		mealRepository:    mealRepository,
		nutritionAnalyzer: nutritionAnalyzer,
	}
}

// WithPublisher returns a copy of LogMeal that broadcasts AI results via pub.
func (uc LogMeal) WithPublisher(pub MealAnalysisPublisher) LogMeal {
	uc.publisher = pub
	return uc
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

	// Save immediately with status = "processing" — no AI call yet.
	meal := domain.Meal{
		DishName:       dishName,
		MealType:       mealType,
		ImageURL:       input.ImageURL,
		RecordedAt:     recordedAt,
		AnalysisStatus: domain.AnalysisStatusProcessing,
		// Analysis is nil; the DB columns default to 0 / empty string.
	}

	saved, err := uc.mealRepository.Create(ctx, meal)
	if err != nil {
		return domain.Meal{}, err
	}

	// Spawn background goroutine — AI analysis with retry.
	// Uses a detached context so HTTP request cancellation does not abort it.
	go uc.runAnalysisWithRetry(context.Background(), saved)

	return saved, nil
}

// runAnalysisWithRetry calls the AI analyzer up to analysisMaxRetries times
// with exponential back-off (1 s → 2 s → 4 s). On success it updates the DB
// and broadcasts the completed meal. On final failure it marks the meal as
// "failed" and broadcasts an error event.
func (uc LogMeal) runAnalysisWithRetry(ctx context.Context, meal domain.Meal) {
	logMealAnalysisRunner(uc.mealRepository, uc.nutritionAnalyzer, uc.publisher).run(ctx, meal)
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
