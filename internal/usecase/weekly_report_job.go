package usecase

import (
	"context"
	"time"

	"go.uber.org/zap"

	"sugary/internal/domain"
	"sugary/internal/platform/timeutil"
)

type weeklyReportCompiler interface {
	Execute(ctx context.Context, day time.Time) (domain.WeeklyReport, error)
}

type WeeklyReportJob struct {
	compiler weeklyReportCompiler
	location *time.Location
	now      func() time.Time
}

func NewWeeklyReportJob(compiler weeklyReportCompiler, timezone string) (WeeklyReportJob, error) {
	location, err := timeutil.ResolveLocation(timezone)
	if err != nil {
		return WeeklyReportJob{}, err
	}

	return WeeklyReportJob{
		compiler: compiler,
		location: location,
		now:      time.Now,
	}, nil
}

func (j WeeklyReportJob) Name() string {
	return "weekly_report"
}

func (j WeeklyReportJob) Run(ctx context.Context) error {
	startedAt := time.Now()
	now := j.now().In(j.location)
	targetWeekStart := timeutil.StartOfWeek(now).AddDate(0, 0, -7)

	report, err := j.compiler.Execute(ctx, targetWeekStart)
	if err != nil {
		zap.L().Error("weekly_report_job_failed",
			zap.String("job_name", j.Name()),
			zap.String("timezone", j.location.String()),
			zap.String("week_start_date", targetWeekStart.Format(time.DateOnly)),
			zap.Duration("duration", time.Since(startedAt)),
			zap.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
			zap.Error(err),
		)
		return err
	}

	duration := time.Since(startedAt)
	zap.L().Info("weekly_report_job_completed",
		zap.String("job_name", j.Name()),
		zap.String("timezone", j.location.String()),
		zap.String("week_start_date", targetWeekStart.Format(time.DateOnly)),
		zap.String("week_end_date", targetWeekStart.AddDate(0, 0, 6).Format(time.DateOnly)),
		zap.Int("meal_count", report.MealCount),
		zap.Float64("total_sugar_grams", report.TotalSugarGrams),
		zap.Duration("duration", duration),
		zap.Int64("duration_ms", duration.Milliseconds()),
	)
	return nil
}
