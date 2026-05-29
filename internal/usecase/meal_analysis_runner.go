package usecase

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"sugary/internal/domain"
)

// MealAnalysisPublisher broadcasts async AI analysis results to connected clients.
// The hub in platform/hub implements this interface.
// Broadcast receives a fully serialized JSON message so the usecase stays
// decoupled from WebSocket framing details.
type MealAnalysisPublisher interface {
	Broadcast(msg []byte)
}

// mealAnalysisPush is the JSON envelope sent over WebSocket when AI analysis
// completes or fails.
type mealAnalysisPush struct {
	Type   string           `json:"type"`
	Status string           `json:"status"`
	Data   *domain.Meal     `json:"data,omitempty"`
	Error  *analysisPushErr `json:"error,omitempty"`
}

type analysisPushErr struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	MealID  int64  `json:"meal_id"`
}

const (
	analysisMaxRetries = 3
	analysisBaseDelay  = time.Second
)

type mealAnalysisRunner struct {
	mealRepository    domain.MealRepository
	nutritionAnalyzer domain.NutritionAnalyzer
	publisher         MealAnalysisPublisher
	logs              mealAnalysisRunnerLogs
}

type mealAnalysisRunnerLogs struct {
	attemptCompleted   string
	retry              string
	failed             string
	statusSkipDeleted  string
	statusUpdateFailed string
	updateFailed       string
	completed          string
	marshalFailed      string
}

func (r mealAnalysisRunner) run(ctx context.Context, meal domain.Meal) {
	input := domain.AnalyzeMealInput{
		DishName: meal.DishName,
		ImageURL: meal.ImageURL,
	}
	jobStartedAt := time.Now()

	var (
		nutrition domain.Nutrition
		lastErr   error
	)

	for attempt := 0; attempt < analysisMaxRetries; attempt++ {
		if attempt > 0 {
			backoff := analysisBaseDelay * (1 << (attempt - 1))
			time.Sleep(backoff)
		}

		if _, err := r.mealRepository.GetByID(ctx, meal.ID); err != nil {
			if err == domain.ErrMealNotFound {
				zap.L().Info(r.logs.statusSkipDeleted, zap.Int64("meal_id", meal.ID))
				return
			}
			lastErr = err
			break
		}

		attemptStartedAt := time.Now()
		nutrition, lastErr = r.nutritionAnalyzer.AnalyzeMeal(ctx, input)
		attemptLatency := time.Since(attemptStartedAt)
		if lastErr == nil {
			zap.L().Info(r.logs.attemptCompleted,
				zap.Int64("meal_id", meal.ID),
				zap.Int("attempt", attempt+1),
				zap.Duration("ai_latency", attemptLatency),
				zap.Int64("ai_latency_ms", attemptLatency.Milliseconds()),
			)
			break
		}

		zap.L().Warn(r.logs.retry,
			zap.Int64("meal_id", meal.ID),
			zap.Int("attempt", attempt+1),
			zap.Duration("ai_latency", attemptLatency),
			zap.Int64("ai_latency_ms", attemptLatency.Milliseconds()),
			zap.Error(lastErr),
		)
	}

	if lastErr != nil {
		totalLatency := time.Since(jobStartedAt)
		zap.L().Error(r.logs.failed,
			zap.Int64("meal_id", meal.ID),
			zap.Duration("total_latency", totalLatency),
			zap.Int64("total_latency_ms", totalLatency.Milliseconds()),
			zap.Error(lastErr),
		)
		if err := r.mealRepository.UpdateAnalysisStatus(ctx, meal.ID, domain.AnalysisStatusFailed); err != nil {
			if err == domain.ErrMealNotFound {
				zap.L().Info(r.logs.statusSkipDeleted, zap.Int64("meal_id", meal.ID))
				return
			}
			zap.L().Error(r.logs.statusUpdateFailed,
				zap.Int64("meal_id", meal.ID),
				zap.Error(err),
			)
			return
		}
		r.broadcast(mealAnalysisPush{
			Type:   "meal_analysis",
			Status: domain.AnalysisStatusFailed,
			Error: &analysisPushErr{
				Code:    "analysis_failed",
				Message: "AI analysis failed after " + itoa(analysisMaxRetries) + " retries: " + lastErr.Error(),
				MealID:  meal.ID,
			},
		})
		return
	}

	completed, err := r.mealRepository.UpdateAnalysisResult(ctx, meal.ID, nutrition)
	if err != nil {
		zap.L().Error(r.logs.updateFailed,
			zap.Int64("meal_id", meal.ID),
			zap.Error(err),
		)
		return
	}

	totalLatency := time.Since(jobStartedAt)
	zap.L().Info(r.logs.completed,
		zap.Int64("meal_id", meal.ID),
		zap.Float64("sugar_grams", nutrition.EstimatedSugarGrams),
		zap.String("risk_level", nutrition.RiskLevel),
		zap.Duration("total_latency", totalLatency),
		zap.Int64("total_latency_ms", totalLatency.Milliseconds()),
	)

	r.broadcast(mealAnalysisPush{
		Type:   "meal_analysis",
		Status: domain.AnalysisStatusCompleted,
		Data:   &completed,
	})
}

func (r mealAnalysisRunner) broadcast(msg mealAnalysisPush) {
	if r.publisher == nil {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		zap.L().Error(r.logs.marshalFailed, zap.Error(err))
		return
	}

	r.publisher.Broadcast(data)
}

func newMealAnalysisRunner(
	mealRepository domain.MealRepository,
	nutritionAnalyzer domain.NutritionAnalyzer,
	publisher MealAnalysisPublisher,
	logs mealAnalysisRunnerLogs,
) mealAnalysisRunner {
	return mealAnalysisRunner{
		mealRepository:    mealRepository,
		nutritionAnalyzer: nutritionAnalyzer,
		publisher:         publisher,
		logs:              logs,
	}
}

func logMealAnalysisRunner(
	mealRepository domain.MealRepository,
	nutritionAnalyzer domain.NutritionAnalyzer,
	publisher MealAnalysisPublisher,
) mealAnalysisRunner {
	return newMealAnalysisRunner(mealRepository, nutritionAnalyzer, publisher, mealAnalysisRunnerLogs{
		attemptCompleted:   "meal_analysis_attempt_completed",
		retry:              "meal_analysis_retry",
		failed:             "meal_analysis_failed",
		statusSkipDeleted:  "meal_analysis_status_skip_deleted",
		statusUpdateFailed: "meal_analysis_status_update_failed",
		updateFailed:       "meal_analysis_update_failed",
		completed:          "meal_analysis_completed",
		marshalFailed:      "meal_analysis_marshal_failed",
	})
}

func editMealAnalysisRunner(
	mealRepository domain.MealRepository,
	nutritionAnalyzer domain.NutritionAnalyzer,
	publisher MealAnalysisPublisher,
) mealAnalysisRunner {
	return newMealAnalysisRunner(mealRepository, nutritionAnalyzer, publisher, mealAnalysisRunnerLogs{
		attemptCompleted:   "meal_reanalysis_attempt_completed",
		retry:              "meal_reanalysis_retry",
		failed:             "meal_reanalysis_failed",
		statusSkipDeleted:  "meal_reanalysis_status_skip_deleted",
		statusUpdateFailed: "meal_reanalysis_status_update_failed",
		updateFailed:       "meal_reanalysis_update_failed",
		completed:          "meal_reanalysis_completed",
		marshalFailed:      "meal_reanalysis_marshal_failed",
	})
}

// itoa is a minimal int-to-string helper to avoid importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
