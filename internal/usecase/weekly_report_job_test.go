package usecase

import (
	"context"
	"testing"
	"time"

	"sugary/internal/domain"
)

type stubWeeklyReportCompiler struct {
	executeFn func(ctx context.Context, day time.Time) (domain.WeeklyReport, error)
}

func (s stubWeeklyReportCompiler) Execute(ctx context.Context, day time.Time) (domain.WeeklyReport, error) {
	return s.executeFn(ctx, day)
}

func TestWeeklyReportJobRunUsesPreviousCompletedWeekOnMonday(t *testing.T) {
	t.Parallel()

	var gotDay time.Time
	job, err := NewWeeklyReportJob(stubWeeklyReportCompiler{
		executeFn: func(ctx context.Context, day time.Time) (domain.WeeklyReport, error) {
			gotDay = day
			return domain.WeeklyReport{}, nil
		},
	}, "Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatalf("NewWeeklyReportJob() error = %v", err)
	}

	job.now = func() time.Time {
		return time.Date(2026, 6, 14, 17, 10, 0, 0, time.UTC)
	}

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	expected := time.Date(2026, 6, 8, 0, 0, 0, 0, job.location)
	if !gotDay.Equal(expected) {
		t.Fatalf("Run() day = %v, want %v", gotDay, expected)
	}
}

func TestWeeklyReportJobRunUsesPreviousCompletedWeekMidweek(t *testing.T) {
	t.Parallel()

	var gotDay time.Time
	job, err := NewWeeklyReportJob(stubWeeklyReportCompiler{
		executeFn: func(ctx context.Context, day time.Time) (domain.WeeklyReport, error) {
			gotDay = day
			return domain.WeeklyReport{}, nil
		},
	}, "Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatalf("NewWeeklyReportJob() error = %v", err)
	}

	job.now = func() time.Time {
		return time.Date(2026, 6, 17, 12, 0, 0, 0, job.location)
	}

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	expected := time.Date(2026, 6, 8, 0, 0, 0, 0, job.location)
	if !gotDay.Equal(expected) {
		t.Fatalf("Run() day = %v, want %v", gotDay, expected)
	}
}

func TestNewWeeklyReportJobInvalidTimezone(t *testing.T) {
	t.Parallel()

	if _, err := NewWeeklyReportJob(stubWeeklyReportCompiler{}, "Invalid/Timezone"); err == nil {
		t.Fatal("expected invalid timezone error")
	}
}
