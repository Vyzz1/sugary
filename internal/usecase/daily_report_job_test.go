package usecase

import (
	"context"
	"testing"
	"time"

	"sugary/internal/domain"
)

type stubDailyReportCompiler struct {
	executeFn func(ctx context.Context, day time.Time) (domain.DailyReport, error)
}

func (s stubDailyReportCompiler) Execute(ctx context.Context, day time.Time) (domain.DailyReport, error) {
	return s.executeFn(ctx, day)
}

func TestDailyReportJobRunUsesPreviousLocalDay(t *testing.T) {
	var gotDay time.Time
	job, err := NewDailyReportJob(stubDailyReportCompiler{
		executeFn: func(ctx context.Context, day time.Time) (domain.DailyReport, error) {
			gotDay = day
			return domain.DailyReport{}, nil
		},
	}, "Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatalf("NewDailyReportJob() error = %v", err)
	}

	job.now = func() time.Time {
		return time.Date(2026, 5, 29, 0, 5, 0, 0, time.UTC)
	}

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	expected := time.Date(2026, 5, 28, 0, 0, 0, 0, job.location)
	if !gotDay.Equal(expected) {
		t.Fatalf("Run() day = %v, want %v", gotDay, expected)
	}
}
