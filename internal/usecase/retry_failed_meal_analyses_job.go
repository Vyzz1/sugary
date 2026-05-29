package usecase

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type failedMealAnalysisRetrier interface {
	Execute(ctx context.Context) (int, error)
}

type RetryFailedMealAnalysesJob struct {
	retrier failedMealAnalysisRetrier
}

func NewRetryFailedMealAnalysesJob(retrier failedMealAnalysisRetrier) RetryFailedMealAnalysesJob {
	return RetryFailedMealAnalysesJob{retrier: retrier}
}

func (j RetryFailedMealAnalysesJob) Name() string {
	return "retry_failed_meal_analyses"
}

func (j RetryFailedMealAnalysesJob) Run(ctx context.Context) error {
	startedAt := time.Now()
	retried, err := j.retrier.Execute(ctx)
	if err != nil {
		duration := time.Since(startedAt)
		zap.L().Error("retry_failed_meal_analyses_job_failed",
			zap.String("job_name", j.Name()),
			zap.Int64("duration_ms", duration.Milliseconds()),
			zap.Error(err),
		)
		return err
	}

	duration := time.Since(startedAt)
	zap.L().Info("retry_failed_meal_analyses_job_completed",
		zap.String("job_name", j.Name()),
		zap.Int("retried_count", retried),
		zap.Int64("duration_ms", duration.Milliseconds()),
	)
	return nil
}
