package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"sugary/internal/domain"
)

type stubWeeklyReportRepository struct {
	saveFn           func(ctx context.Context, report domain.WeeklyReport) error
	getByWeekStartFn func(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error)
}

type stubWeeklyReportInterpreter struct {
	providerName       string
	generateInsightsFn func(ctx context.Context, input domain.GenerateWeeklyReportSummaryInput) (domain.WeeklyReportAIInsights, error)
}

func (s stubWeeklyReportRepository) Save(ctx context.Context, report domain.WeeklyReport) error {
	return s.saveFn(ctx, report)
}

func (s stubWeeklyReportRepository) GetByWeekStart(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error) {
	return s.getByWeekStartFn(ctx, weekStart)
}

func (s stubWeeklyReportInterpreter) GenerateInsights(ctx context.Context, input domain.GenerateWeeklyReportSummaryInput) (domain.WeeklyReportAIInsights, error) {
	return s.generateInsightsFn(ctx, input)
}

func (s stubWeeklyReportInterpreter) AIInsightProviderName() string {
	return s.providerName
}

func TestCompileWeeklyReportExecute(t *testing.T) {
	t.Parallel()

	location, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatalf("expected valid location, got %v", err)
	}

	createdAt := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)
	day := time.Date(2026, 6, 10, 12, 0, 0, 0, location)

	uc := NewCompileWeeklyReport(
		stubMealRepository{
			listFn: func(ctx context.Context, filter domain.MealListFilter) ([]domain.Meal, int64, error) {
				expectedStart := time.Date(2026, 6, 8, 0, 0, 0, 0, location)
				expectedEnd := time.Date(2026, 6, 15, 0, 0, 0, 0, location)
				if filter.StartAt == nil || !filter.StartAt.Equal(expectedStart) {
					t.Fatalf("expected week start %s, got %v", expectedStart, filter.StartAt)
				}
				if filter.EndAt == nil || !filter.EndAt.Equal(expectedEnd) {
					t.Fatalf("expected week end %s, got %v", expectedEnd, filter.EndAt)
				}
				return []domain.Meal{
					{
						DishName:   "Milk tea",
						MealType:   domain.MealTypeDrink,
						RecordedAt: time.Date(2026, 6, 9, 14, 0, 0, 0, location),
						Analysis: &domain.Nutrition{
							EstimatedSugarGrams: 35,
							RiskLevel:           "high",
						},
					},
					{
						DishName:   "Pho",
						MealType:   domain.MealTypeLunch,
						RecordedAt: time.Date(2026, 6, 10, 12, 0, 0, 0, location),
						Analysis: &domain.Nutrition{
							EstimatedSugarGrams: 6,
							RiskLevel:           "low",
						},
					},
					{
						DishName:   "Pending",
						RecordedAt: time.Date(2026, 6, 11, 12, 0, 0, 0, location),
					},
				}, 3, nil
			},
		},
		stubWeeklyReportRepository{
			saveFn: func(ctx context.Context, report domain.WeeklyReport) error {
				if report.CreatedAt != createdAt {
					t.Fatalf("expected created_at %s, got %s", createdAt, report.CreatedAt)
				}
				if len(report.DailyBreakdown) != 7 {
					t.Fatalf("expected 7 daily breakdown items, got %d", len(report.DailyBreakdown))
				}
				return nil
			},
			getByWeekStartFn: func(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error) {
				return domain.WeeklyReport{}, false, nil
			},
		},
		stubWeeklyReportInterpreter{
			generateInsightsFn: func(ctx context.Context, input domain.GenerateWeeklyReportSummaryInput) (domain.WeeklyReportAIInsights, error) {
				if input.Report.AnalyzedMealCount != 2 {
					t.Fatalf("expected analyzed meal count 2, got %d", input.Report.AnalyzedMealCount)
				}
				return domain.WeeklyReportAIInsights{
					Summary:         "AI weekly summary",
					Recommendations: []string{"Plan lower-sugar drinks next week"},
				}, nil
			},
		},
	)
	uc.now = func() time.Time { return createdAt }

	report, err := uc.Execute(context.Background(), day)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if report.WeekStartDate != time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("expected week_start_date 2026-06-08, got %s", report.WeekStartDate)
	}
	if report.WeekEndDate != time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("expected week_end_date 2026-06-14, got %s", report.WeekEndDate)
	}
	if report.MealCount != 3 {
		t.Fatalf("expected meal count 3, got %d", report.MealCount)
	}
	if report.AnalyzedMealCount != 2 {
		t.Fatalf("expected analyzed meal count 2, got %d", report.AnalyzedMealCount)
	}
	if report.TotalSugarGrams != 41 {
		t.Fatalf("expected total sugar 41, got %v", report.TotalSugarGrams)
	}
	if report.AverageSugarGrams != 20.5 {
		t.Fatalf("expected average sugar 20.5, got %v", report.AverageSugarGrams)
	}
	if report.HighestRiskLevel != "high" {
		t.Fatalf("expected highest risk high, got %q", report.HighestRiskLevel)
	}
	if report.Summary != "AI weekly summary" {
		t.Fatalf("expected AI weekly summary, got %q", report.Summary)
	}
	if report.AIInsightSource != "gemini" {
		t.Fatalf("expected ai_insight_source gemini, got %q", report.AIInsightSource)
	}
}

func TestCompileWeeklyReportExecuteKeepsExistingCreatedAt(t *testing.T) {
	t.Parallel()

	existingCreatedAt := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
	newNow := time.Date(2026, 6, 12, 8, 0, 0, 0, time.UTC)

	uc := NewCompileWeeklyReport(
		stubMealRepository{
			listFn: func(ctx context.Context, filter domain.MealListFilter) ([]domain.Meal, int64, error) {
				return []domain.Meal{}, 0, nil
			},
		},
		stubWeeklyReportRepository{
			saveFn: func(ctx context.Context, report domain.WeeklyReport) error {
				if report.CreatedAt != existingCreatedAt {
					t.Fatalf("expected existing created_at %s, got %s", existingCreatedAt, report.CreatedAt)
				}
				return nil
			},
			getByWeekStartFn: func(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error) {
				return domain.WeeklyReport{CreatedAt: existingCreatedAt}, true, nil
			},
		},
		nil,
	)
	uc.now = func() time.Time { return newNow }

	report, err := uc.Execute(context.Background(), time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if report.CreatedAt != existingCreatedAt {
		t.Fatalf("expected existing created_at %s, got %s", existingCreatedAt, report.CreatedAt)
	}
}

func TestCompileWeeklyReportExecuteUsesFallbackWhenAIFails(t *testing.T) {
	t.Parallel()

	uc := NewCompileWeeklyReport(
		stubMealRepository{
			listFn: func(ctx context.Context, filter domain.MealListFilter) ([]domain.Meal, int64, error) {
				return []domain.Meal{
					{
						DishName:   "Milk tea",
						RecordedAt: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC),
						Analysis: &domain.Nutrition{
							EstimatedSugarGrams: 30,
							RiskLevel:           "high",
						},
					},
				}, 1, nil
			},
		},
		stubWeeklyReportRepository{
			saveFn: func(ctx context.Context, report domain.WeeklyReport) error { return nil },
			getByWeekStartFn: func(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error) {
				return domain.WeeklyReport{}, false, nil
			},
		},
		stubWeeklyReportInterpreter{
			generateInsightsFn: func(ctx context.Context, input domain.GenerateWeeklyReportSummaryInput) (domain.WeeklyReportAIInsights, error) {
				return domain.WeeklyReportAIInsights{}, errors.New("ai unavailable")
			},
		},
	)

	report, err := uc.Execute(context.Background(), time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if report.AIInsightSource != "fallback" {
		t.Fatalf("expected fallback source, got %q", report.AIInsightSource)
	}
	if report.AIInsights.Summary == "" {
		t.Fatal("expected fallback insights summary")
	}
}

func TestCompileWeeklyReportExecuteBroadcastsAfterSuccessfulSave(t *testing.T) {
	t.Parallel()

	broadcastCh := make(chan []byte, 1)
	uc := NewCompileWeeklyReport(
		stubMealRepository{
			listFn: func(ctx context.Context, filter domain.MealListFilter) ([]domain.Meal, int64, error) {
				return []domain.Meal{}, 0, nil
			},
		},
		stubWeeklyReportRepository{
			saveFn: func(ctx context.Context, report domain.WeeklyReport) error { return nil },
			getByWeekStartFn: func(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error) {
				return domain.WeeklyReport{}, false, nil
			},
		},
		nil,
	).WithPublisher(&capturingDailyReportPublisher{ch: broadcastCh})

	_, err := uc.Execute(context.Background(), time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	select {
	case msg := <-broadcastCh:
		var push weeklyReportPush
		if err := json.Unmarshal(msg, &push); err != nil {
			t.Fatalf("expected valid JSON broadcast, got %v", err)
		}
		if push.Type != "weekly_report" {
			t.Fatalf("expected type weekly_report, got %q", push.Type)
		}
		if push.Data == nil {
			t.Fatal("expected weekly report payload")
		}
	default:
		t.Fatal("expected weekly report broadcast")
	}
}

func TestCompileWeeklyReportExecuteReturnsExistingWhenAICompleted(t *testing.T) {
	t.Parallel()

	expected := domain.WeeklyReport{
		WeekStartDate:   time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC),
		WeekEndDate:     time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC),
		Summary:         "Existing weekly AI summary",
		AIInsightSource: "gemini",
		AIInsightStatus: "completed",
	}

	uc := NewCompileWeeklyReport(
		stubMealRepository{
			listFn: func(ctx context.Context, filter domain.MealListFilter) ([]domain.Meal, int64, error) {
				t.Fatal("expected List not to be called")
				return nil, 0, nil
			},
		},
		stubWeeklyReportRepository{
			saveFn: func(ctx context.Context, report domain.WeeklyReport) error {
				t.Fatal("expected Save not to be called")
				return nil
			},
			getByWeekStartFn: func(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error) {
				return expected, true, nil
			},
		},
		stubWeeklyReportInterpreter{
			generateInsightsFn: func(ctx context.Context, input domain.GenerateWeeklyReportSummaryInput) (domain.WeeklyReportAIInsights, error) {
				t.Fatal("expected AI interpreter not to be called")
				return domain.WeeklyReportAIInsights{}, nil
			},
		},
	)

	report, err := uc.Execute(context.Background(), time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if report.Summary != expected.Summary {
		t.Fatalf("expected existing report summary %q, got %q", expected.Summary, report.Summary)
	}
}

func TestCompileWeeklyReportExecuteRecompilesExistingFallback(t *testing.T) {
	t.Parallel()

	listCalled := false
	saveCalled := false
	uc := NewCompileWeeklyReport(
		stubMealRepository{
			listFn: func(ctx context.Context, filter domain.MealListFilter) ([]domain.Meal, int64, error) {
				listCalled = true
				return []domain.Meal{}, 0, nil
			},
		},
		stubWeeklyReportRepository{
			saveFn: func(ctx context.Context, report domain.WeeklyReport) error {
				saveCalled = true
				return nil
			},
			getByWeekStartFn: func(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error) {
				return domain.WeeklyReport{
					AIInsightSource: "fallback",
					AIInsightStatus: "fallback",
				}, true, nil
			},
		},
		nil,
	)

	if _, err := uc.Execute(context.Background(), time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !listCalled {
		t.Fatal("expected fallback report to be recompiled")
	}
	if !saveCalled {
		t.Fatal("expected fallback report recompile to save")
	}
}

func TestCompileWeeklyReportExecuteSendsEmailAfterSuccessfulSave(t *testing.T) {
	t.Parallel()

	saveCalled := false
	emailCalled := false
	uc := NewCompileWeeklyReport(
		stubMealRepository{
			listFn: func(ctx context.Context, filter domain.MealListFilter) ([]domain.Meal, int64, error) {
				return []domain.Meal{}, 0, nil
			},
		},
		stubWeeklyReportRepository{
			saveFn: func(ctx context.Context, report domain.WeeklyReport) error {
				saveCalled = true
				return nil
			},
			getByWeekStartFn: func(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error) {
				return domain.WeeklyReport{}, false, nil
			},
		},
		nil,
	).WithEmailSender(stubReportEmailSender{
		sendWeeklyFn: func(ctx context.Context, report domain.WeeklyReport) error {
			if !saveCalled {
				t.Fatal("expected Save before email")
			}
			emailCalled = true
			return nil
		},
	})

	if _, err := uc.Execute(context.Background(), time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !emailCalled {
		t.Fatal("expected weekly report email")
	}
}

func TestCompileWeeklyReportExecuteIgnoresEmailFailure(t *testing.T) {
	t.Parallel()

	uc := NewCompileWeeklyReport(
		stubMealRepository{
			listFn: func(ctx context.Context, filter domain.MealListFilter) ([]domain.Meal, int64, error) {
				return []domain.Meal{}, 0, nil
			},
		},
		stubWeeklyReportRepository{
			saveFn: func(ctx context.Context, report domain.WeeklyReport) error {
				return nil
			},
			getByWeekStartFn: func(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error) {
				return domain.WeeklyReport{}, false, nil
			},
		},
		nil,
	).WithEmailSender(stubReportEmailSender{
		sendWeeklyFn: func(ctx context.Context, report domain.WeeklyReport) error {
			return errors.New("email failed")
		},
	})

	if _, err := uc.Execute(context.Background(), time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("expected email error not to fail compile, got %v", err)
	}
}

func TestCompileWeeklyReportExecuteDoesNotSendEmailWhenAICompletedSkip(t *testing.T) {
	t.Parallel()

	uc := NewCompileWeeklyReport(
		stubMealRepository{
			listFn: func(ctx context.Context, filter domain.MealListFilter) ([]domain.Meal, int64, error) {
				t.Fatal("expected List not to be called")
				return nil, 0, nil
			},
		},
		stubWeeklyReportRepository{
			saveFn: func(ctx context.Context, report domain.WeeklyReport) error {
				t.Fatal("expected Save not to be called")
				return nil
			},
			getByWeekStartFn: func(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error) {
				return domain.WeeklyReport{
					AIInsightSource: "gemini",
					AIInsightStatus: "completed",
				}, true, nil
			},
		},
		nil,
	).WithEmailSender(stubReportEmailSender{
		sendWeeklyFn: func(ctx context.Context, report domain.WeeklyReport) error {
			t.Fatal("expected email not to be sent")
			return nil
		},
	})

	if _, err := uc.Execute(context.Background(), time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCompileWeeklyReportExecuteUsesInterpreterProviderAsAISource(t *testing.T) {
	t.Parallel()

	uc := NewCompileWeeklyReport(
		stubMealRepository{
			listFn: func(ctx context.Context, filter domain.MealListFilter) ([]domain.Meal, int64, error) {
				return []domain.Meal{
					{
						DishName:   "Milk tea",
						RecordedAt: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC),
						Analysis: &domain.Nutrition{
							EstimatedSugarGrams: 35,
							RiskLevel:           "high",
						},
					},
				}, 1, nil
			},
		},
		stubWeeklyReportRepository{
			saveFn: func(ctx context.Context, report domain.WeeklyReport) error {
				return nil
			},
			getByWeekStartFn: func(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error) {
				return domain.WeeklyReport{}, false, nil
			},
		},
		stubWeeklyReportInterpreter{
			providerName: "huggingface",
			generateInsightsFn: func(ctx context.Context, input domain.GenerateWeeklyReportSummaryInput) (domain.WeeklyReportAIInsights, error) {
				return domain.WeeklyReportAIInsights{Summary: "AI weekly summary"}, nil
			},
		},
	)

	report, err := uc.Execute(context.Background(), time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if report.AIInsightSource != "huggingface" {
		t.Fatalf("expected ai source huggingface, got %q", report.AIInsightSource)
	}
}
