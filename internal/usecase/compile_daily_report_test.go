package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"sugary/internal/domain"
)

type stubDailyReportRepository struct {
	saveFn     func(ctx context.Context, report domain.DailyReport) error
	getByDayFn func(ctx context.Context, day time.Time) (domain.DailyReport, bool, error)
}

type stubDailyReportInterpreter struct {
	generateInsightsFn func(ctx context.Context, input domain.GenerateDailyReportSummaryInput) (domain.DailyReportAIInsights, error)
}

func (s stubDailyReportRepository) Save(ctx context.Context, report domain.DailyReport) error {
	return s.saveFn(ctx, report)
}

func (s stubDailyReportRepository) GetByDay(ctx context.Context, day time.Time) (domain.DailyReport, bool, error) {
	return s.getByDayFn(ctx, day)
}

func (s stubDailyReportInterpreter) GenerateInsights(ctx context.Context, input domain.GenerateDailyReportSummaryInput) (domain.DailyReportAIInsights, error) {
	return s.generateInsightsFn(ctx, input)
}

type capturingDailyReportPublisher struct {
	ch chan []byte
}

func (p *capturingDailyReportPublisher) Broadcast(msg []byte) {
	select {
	case p.ch <- msg:
	default:
	}
}

func TestCompileDailyReportExecute(t *testing.T) {
	t.Parallel()

	location, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatalf("expected valid location, got %v", err)
	}

	day := time.Date(2026, 5, 21, 15, 0, 0, 0, location)

	uc := NewCompileDailyReport(
		stubMealRepository{
			createFn: func(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
				return meal, nil
			},
			listByDayFn: func(ctx context.Context, filter domain.MealsByDayFilter) ([]domain.Meal, error) {
				if filter.Day.Hour() != 0 {
					t.Fatalf("expected day to be normalized, got %v", filter.Day)
				}
				if filter.Day.Location().String() != location.String() {
					t.Fatalf("expected timezone %q, got %q", location.String(), filter.Day.Location().String())
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
		stubDailyReportInterpreter{
			generateInsightsFn: func(ctx context.Context, input domain.GenerateDailyReportSummaryInput) (domain.DailyReportAIInsights, error) {
				return domain.DailyReportAIInsights{
					Summary:         "AI summary",
					Recommendations: []string{"Use less sweet drinks"},
				}, nil
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
	if report.Summary != "AI summary" {
		t.Fatalf("expected AI summary, got %q", report.Summary)
	}
	if report.AIInsights.Summary != "AI summary" {
		t.Fatalf("expected AI insights summary, got %q", report.AIInsights.Summary)
	}
	expectedStoredDate := time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC)
	if !report.Date.Equal(expectedStoredDate) {
		t.Fatalf("expected stored report date %s, got %s", expectedStoredDate, report.Date)
	}
}

func TestCompileDailyReportExecuteSkipsUnanalyzedMealsInAverage(t *testing.T) {
	t.Parallel()

	day := time.Date(2026, 5, 21, 15, 0, 0, 0, time.FixedZone("ICT", 7*60*60))

	uc := NewCompileDailyReport(
		stubMealRepository{
			createFn: func(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
				return meal, nil
			},
			listByDayFn: func(ctx context.Context, filter domain.MealsByDayFilter) ([]domain.Meal, error) {
				return []domain.Meal{
					{
						DishName: "Milk tea",
						Analysis: &domain.Nutrition{
							EstimatedSugarGrams: 30,
							RiskLevel:           "high",
						},
					},
					{
						DishName: "Unknown",
						Analysis: nil,
					},
				}, nil
			},
		},
		stubDailyReportRepository{
			saveFn: func(ctx context.Context, report domain.DailyReport) error { return nil },
			getByDayFn: func(ctx context.Context, day time.Time) (domain.DailyReport, bool, error) {
				return domain.DailyReport{}, false, nil
			},
		},
		stubDailyReportInterpreter{
			generateInsightsFn: func(ctx context.Context, input domain.GenerateDailyReportSummaryInput) (domain.DailyReportAIInsights, error) {
				return domain.DailyReportAIInsights{}, errors.New("ai unavailable")
			},
		},
	)

	report, err := uc.Execute(context.Background(), day)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if report.MealCount != 2 {
		t.Fatalf("expected meal count 2, got %d", report.MealCount)
	}
	if report.TotalSugarGrams != 30 {
		t.Fatalf("expected total sugar 30, got %v", report.TotalSugarGrams)
	}
	if report.AverageSugarGrams != 30 {
		t.Fatalf("expected average sugar 30, got %v", report.AverageSugarGrams)
	}
	if report.HighestRiskLevel != "high" {
		t.Fatalf("expected highest risk high, got %q", report.HighestRiskLevel)
	}
	if report.Summary == "" {
		t.Fatalf("expected fallback summary")
	}
	if report.AIInsights.Summary == "" {
		t.Fatalf("expected fallback ai insights summary")
	}
}

func TestCompileDailyReportExecuteNoAnalyzedMeals(t *testing.T) {
	t.Parallel()

	day := time.Date(2026, 5, 21, 15, 0, 0, 0, time.FixedZone("ICT", 7*60*60))

	uc := NewCompileDailyReport(
		stubMealRepository{
			createFn: func(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
				return meal, nil
			},
			listByDayFn: func(ctx context.Context, filter domain.MealsByDayFilter) ([]domain.Meal, error) {
				return []domain.Meal{
					{DishName: "Meal 1"},
					{DishName: "Meal 2"},
				}, nil
			},
		},
		stubDailyReportRepository{
			saveFn: func(ctx context.Context, report domain.DailyReport) error { return nil },
			getByDayFn: func(ctx context.Context, day time.Time) (domain.DailyReport, bool, error) {
				return domain.DailyReport{}, false, nil
			},
		},
		nil,
	)

	report, err := uc.Execute(context.Background(), day)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if report.TotalSugarGrams != 0 {
		t.Fatalf("expected total sugar 0, got %v", report.TotalSugarGrams)
	}
	if report.AverageSugarGrams != 0 {
		t.Fatalf("expected average sugar 0, got %v", report.AverageSugarGrams)
	}
	if report.HighestRiskLevel != "unknown" {
		t.Fatalf("expected highest risk unknown, got %q", report.HighestRiskLevel)
	}
	if report.AIInsights.Summary == "" {
		t.Fatalf("expected fallback ai insights summary")
	}
}

func TestCompileDailyReportExecuteBroadcastsAfterSuccessfulSave(t *testing.T) {
	t.Parallel()

	broadcastCh := make(chan []byte, 1)
	saveCalled := false

	uc := NewCompileDailyReport(
		stubMealRepository{
			listByDayFn: func(ctx context.Context, filter domain.MealsByDayFilter) ([]domain.Meal, error) {
				return []domain.Meal{
					{
						DishName: "Milk tea",
						Analysis: &domain.Nutrition{
							EstimatedSugarGrams: 35,
							RiskLevel:           "high",
						},
					},
				}, nil
			},
		},
		stubDailyReportRepository{
			saveFn: func(ctx context.Context, report domain.DailyReport) error {
				saveCalled = true
				return nil
			},
			getByDayFn: func(ctx context.Context, day time.Time) (domain.DailyReport, bool, error) {
				return domain.DailyReport{}, false, nil
			},
		},
		nil,
	).WithPublisher(&capturingDailyReportPublisher{ch: broadcastCh})

	_, err := uc.Execute(context.Background(), time.Date(2026, 5, 21, 15, 0, 0, 0, time.FixedZone("ICT", 7*60*60)))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !saveCalled {
		t.Fatal("expected Save to be called before broadcast")
	}

	select {
	case msg := <-broadcastCh:
		var push dailyReportPush
		if err := json.Unmarshal(msg, &push); err != nil {
			t.Fatalf("expected valid JSON broadcast, got %v", err)
		}
		if push.Type != "daily_report" {
			t.Fatalf("expected type daily_report, got %q", push.Type)
		}
		if push.Status != "completed" {
			t.Fatalf("expected status completed, got %q", push.Status)
		}
		if push.Data == nil {
			t.Fatal("expected report payload in broadcast")
		}
	default:
		t.Fatal("expected broadcast after successful save")
	}
}

func TestCompileDailyReportExecuteDoesNotBroadcastWhenSaveFails(t *testing.T) {
	t.Parallel()

	broadcastCh := make(chan []byte, 1)

	uc := NewCompileDailyReport(
		stubMealRepository{
			listByDayFn: func(ctx context.Context, filter domain.MealsByDayFilter) ([]domain.Meal, error) {
				return []domain.Meal{
					{
						DishName: "Milk tea",
						Analysis: &domain.Nutrition{
							EstimatedSugarGrams: 35,
							RiskLevel:           "high",
						},
					},
				}, nil
			},
		},
		stubDailyReportRepository{
			saveFn: func(ctx context.Context, report domain.DailyReport) error {
				return errors.New("save failed")
			},
			getByDayFn: func(ctx context.Context, day time.Time) (domain.DailyReport, bool, error) {
				return domain.DailyReport{}, false, nil
			},
		},
		nil,
	).WithPublisher(&capturingDailyReportPublisher{ch: broadcastCh})

	_, err := uc.Execute(context.Background(), time.Date(2026, 5, 21, 15, 0, 0, 0, time.FixedZone("ICT", 7*60*60)))
	if err == nil {
		t.Fatal("expected save error")
	}

	select {
	case msg := <-broadcastCh:
		t.Fatalf("expected no broadcast when save fails, got %s", string(msg))
	default:
	}
}

func TestCompileDailyReportExecuteNoMealsBroadcastsAfterSuccessfulSave(t *testing.T) {
	t.Parallel()

	broadcastCh := make(chan []byte, 1)

	uc := NewCompileDailyReport(
		stubMealRepository{
			listByDayFn: func(ctx context.Context, filter domain.MealsByDayFilter) ([]domain.Meal, error) {
				return []domain.Meal{}, nil
			},
		},
		stubDailyReportRepository{
			saveFn: func(ctx context.Context, report domain.DailyReport) error {
				return nil
			},
			getByDayFn: func(ctx context.Context, day time.Time) (domain.DailyReport, bool, error) {
				return domain.DailyReport{}, false, nil
			},
		},
		nil,
	).WithPublisher(&capturingDailyReportPublisher{ch: broadcastCh})

	_, err := uc.Execute(context.Background(), time.Date(2026, 5, 21, 15, 0, 0, 0, time.FixedZone("ICT", 7*60*60)))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	select {
	case msg := <-broadcastCh:
		var push dailyReportPush
		if err := json.Unmarshal(msg, &push); err != nil {
			t.Fatalf("expected valid JSON broadcast, got %v", err)
		}
		if push.Data == nil {
			t.Fatal("expected report payload in no-meals broadcast")
		}
		if push.Data.MealCount != 0 {
			t.Fatalf("expected meal_count 0, got %d", push.Data.MealCount)
		}
	default:
		t.Fatal("expected broadcast for no-meals report")
	}
}
