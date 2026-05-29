package usecase

import (
	"context"
	"time"

	"go.uber.org/zap"

	"sugary/internal/domain"
	"sugary/internal/platform/timeutil"
)

type dailyReportCompiler interface {
	Execute(ctx context.Context, day time.Time) (domain.DailyReport, error)
}

type DailyReportJob struct {
	compiler dailyReportCompiler
	location *time.Location
	now      func() time.Time
}

func NewDailyReportJob(compiler dailyReportCompiler, timezone string) (DailyReportJob, error) {
	location, err := timeutil.ResolveLocation(timezone)
	if err != nil {
		return DailyReportJob{}, err
	}

	return DailyReportJob{
		compiler: compiler,
		location: location,
		now:      time.Now,
	}, nil
}

func (j DailyReportJob) Name() string {
	return "daily_report"
}

func (j DailyReportJob) Run(ctx context.Context) error {
	startedAt := time.Now()
	now := j.now().In(j.location)
	targetDay := timeutil.StartOfDay(now).AddDate(0, 0, -1)

	report, err := j.compiler.Execute(ctx, targetDay)
	if err != nil {
		zap.L().Error("daily_report_job_failed",
			zap.String("job_name", j.Name()),
			zap.String("timezone", j.location.String()),
			zap.String("report_date", targetDay.Format(time.DateOnly)),
			zap.Duration("duration", time.Since(startedAt)),
			zap.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
			zap.Error(err),
		)
		return err
	}

	duration := time.Since(startedAt)
	zap.L().Info("daily_report_job_completed",
		zap.String("job_name", j.Name()),
		zap.String("timezone", j.location.String()),
		zap.String("report_date", targetDay.Format(time.DateOnly)),
		zap.Int("meal_count", report.MealCount),
		zap.Float64("total_sugar_grams", report.TotalSugarGrams),
		zap.Duration("duration", duration),
		zap.Int64("duration_ms", duration.Milliseconds()),
	)
	return nil
}
