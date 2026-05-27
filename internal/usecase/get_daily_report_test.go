package usecase

import (
	"context"
	"testing"
	"time"

	"sugary/internal/domain"
)

func TestGetDailyReportExecute(t *testing.T) {
	t.Parallel()

	expectedDay := time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC)
	expectedReport := domain.DailyReport{
		Date:              expectedDay,
		MealCount:         2,
		TotalSugarGrams:   41,
		AverageSugarGrams: 20.5,
		HighestRiskLevel:  "high",
		Summary:           "2 meals logged.",
	}

	uc := NewGetDailyReport(stubDailyReportRepository{
		saveFn: func(ctx context.Context, report domain.DailyReport) error {
			return nil
		},
		getByDayFn: func(ctx context.Context, day time.Time) (domain.DailyReport, bool, error) {
			if !day.Equal(expectedDay) {
				t.Fatalf("expected normalized day %v, got %v", expectedDay, day)
			}
			return expectedReport, true, nil
		},
	})

	location, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatalf("expected valid location, got %v", err)
	}

	report, found, err := uc.Execute(context.Background(), time.Date(2026, 5, 21, 12, 0, 0, 0, location))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !found {
		t.Fatal("expected report to be found")
	}
	if report.MealCount != expectedReport.MealCount {
		t.Fatalf("expected meal count %d, got %d", expectedReport.MealCount, report.MealCount)
	}
}
