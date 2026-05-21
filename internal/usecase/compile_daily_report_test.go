package usecase

import (
	"context"
	"testing"
	"time"

	"sugary/internal/domain"
)

type stubDailyReportRepository struct {
	saveFn     func(ctx context.Context, report domain.DailyReport) error
	getByDayFn func(ctx context.Context, day time.Time) (domain.DailyReport, bool, error)
}

func (s stubDailyReportRepository) Save(ctx context.Context, report domain.DailyReport) error {
	return s.saveFn(ctx, report)
}

func (s stubDailyReportRepository) GetByDay(ctx context.Context, day time.Time) (domain.DailyReport, bool, error) {
	return s.getByDayFn(ctx, day)
}

func TestCompileDailyReportExecute(t *testing.T) {
	t.Parallel()

	day := time.Date(2026, 5, 21, 15, 0, 0, 0, time.UTC)

	uc := NewCompileDailyReport(
		stubMealRepository{
			createFn: func(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
				return meal, nil
			},
			listByDayFn: func(ctx context.Context, requested time.Time) ([]domain.Meal, error) {
				if requested.Hour() != 0 {
					t.Fatalf("expected day to be normalized, got %v", requested)
				}
				return []domain.Meal{
					{
						DishName: "Milk tea",
						Analysis: &domain.Nutrition{
							EstimatedSugarGrams: 35,
							RiskLevel:           "high",
						},
					},
					{
						DishName: "Pho",
						Analysis: &domain.Nutrition{
							EstimatedSugarGrams: 6,
							RiskLevel:           "low",
						},
					},
				}, nil
			},
		},
		stubDailyReportRepository{
			saveFn: func(ctx context.Context, report domain.DailyReport) error {
				if report.MealCount != 2 {
					t.Fatalf("expected meal count 2, got %d", report.MealCount)
				}
				if report.HighestRiskLevel != "high" {
					t.Fatalf("expected high risk, got %q", report.HighestRiskLevel)
				}
				return nil
			},
			getByDayFn: func(ctx context.Context, day time.Time) (domain.DailyReport, bool, error) {
				return domain.DailyReport{}, false, nil
			},
		},
	)

	report, err := uc.Execute(context.Background(), day)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if report.TotalSugarGrams != 41 {
		t.Fatalf("expected total sugar 41, got %v", report.TotalSugarGrams)
	}
	if report.AverageSugarGrams != 20.5 {
		t.Fatalf("expected average sugar 20.5, got %v", report.AverageSugarGrams)
	}
}
