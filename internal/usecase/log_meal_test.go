package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"sugary/internal/domain"
)

// ---- stubs ---------------------------------------------------------------

type stubMealRepository struct {
	createFn               func(ctx context.Context, meal domain.Meal) (domain.Meal, error)
	listByDayFn            func(ctx context.Context, filter domain.MealsByDayFilter) ([]domain.Meal, error)
	listRecentDistinctFn   func(ctx context.Context, filter domain.RecentMealsFilter) ([]domain.Meal, int64, error)
	getByIDFn              func(ctx context.Context, mealID int64) (domain.Meal, error)
	updateMetaFn           func(ctx context.Context, mealID int64, mealType string, recordedAt time.Time) (domain.Meal, error)
	updateForReanalysisFn  func(ctx context.Context, meal domain.Meal) (domain.Meal, error)
	updateWithAIFn         func(ctx context.Context, meal domain.Meal) (domain.Meal, error)
	updateAnalysisFn       func(ctx context.Context, mealID int64, nutrition domain.Nutrition) (domain.Meal, error)
	updateAnalysisResultFn func(ctx context.Context, mealID int64, nutrition domain.Nutrition) (domain.Meal, error)
	listRetryableFailedFn  func(ctx context.Context, filter domain.RetryableFailedMealsFilter) ([]domain.Meal, error)
	retryFailedAnalysisFn  func(ctx context.Context, mealID int64) (domain.Meal, error)
	updateAnalysisStatusFn func(ctx context.Context, mealID int64, status string) error
	softDeleteFn           func(ctx context.Context, mealID int64) error
}

func (s stubMealRepository) Create(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
	return s.createFn(ctx, meal)
}

func (s stubMealRepository) ListByDay(ctx context.Context, filter domain.MealsByDayFilter) ([]domain.Meal, error) {
	return s.listByDayFn(ctx, filter)
}

func (s stubMealRepository) ListRecentDistinct(ctx context.Context, filter domain.RecentMealsFilter) ([]domain.Meal, int64, error) {
	if s.listRecentDistinctFn == nil {
		return nil, 0, nil
	}
	return s.listRecentDistinctFn(ctx, filter)
}

func (s stubMealRepository) UpdateAnalysis(ctx context.Context, mealID int64, nutrition domain.Nutrition) (domain.Meal, error) {
	if s.updateAnalysisFn == nil {
		return domain.Meal{}, nil
	}
	return s.updateAnalysisFn(ctx, mealID, nutrition)
}

func (s stubMealRepository) UpdateAnalysisResult(ctx context.Context, mealID int64, nutrition domain.Nutrition) (domain.Meal, error) {
	if s.updateAnalysisResultFn == nil {
		return domain.Meal{AnalysisStatus: domain.AnalysisStatusCompleted, Analysis: &nutrition}, nil
	}
	return s.updateAnalysisResultFn(ctx, mealID, nutrition)
}

func (s stubMealRepository) ListRetryableFailed(ctx context.Context, filter domain.RetryableFailedMealsFilter) ([]domain.Meal, error) {
	if s.listRetryableFailedFn == nil {
		return nil, nil
	}
	return s.listRetryableFailedFn(ctx, filter)
}

func (s stubMealRepository) RetryFailedAnalysis(ctx context.Context, mealID int64) (domain.Meal, error) {
	if s.retryFailedAnalysisFn == nil {
		return domain.Meal{}, domain.ErrMealNotFound
	}
	return s.retryFailedAnalysisFn(ctx, mealID)
}

func (s stubMealRepository) UpdateAnalysisStatus(ctx context.Context, mealID int64, status string) error {
	if s.updateAnalysisStatusFn == nil {
		return nil
	}
	return s.updateAnalysisStatusFn(ctx, mealID, status)
}

func (s stubMealRepository) GetByID(ctx context.Context, mealID int64) (domain.Meal, error) {
	if s.getByIDFn == nil {
		return domain.Meal{}, domain.ErrMealNotFound
	}
	return s.getByIDFn(ctx, mealID)
}

func (s stubMealRepository) UpdateMeta(ctx context.Context, mealID int64, mealType string, recordedAt time.Time) (domain.Meal, error) {
	if s.updateMetaFn == nil {
		return domain.Meal{}, nil
	}
	return s.updateMetaFn(ctx, mealID, mealType, recordedAt)
}

func (s stubMealRepository) UpdateForReanalysis(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
	if s.updateForReanalysisFn == nil {
		return domain.Meal{}, nil
	}
	return s.updateForReanalysisFn(ctx, meal)
}

func (s stubMealRepository) UpdateWithAnalysis(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
	if s.updateWithAIFn == nil {
		return domain.Meal{}, nil
	}
	return s.updateWithAIFn(ctx, meal)
}

func (s stubMealRepository) SoftDelete(ctx context.Context, mealID int64) error {
	if s.softDeleteFn == nil {
		return nil
	}
	return s.softDeleteFn(ctx, mealID)
}

type stubNutritionAnalyzer struct {
	analyzeFn func(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error)
}

func (s stubNutritionAnalyzer) AnalyzeMeal(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
	return s.analyzeFn(ctx, input)
}

// noopPublisher discards all broadcast messages (used in tests that don't
// need to observe WS output).
type noopPublisher struct{}

func (noopPublisher) Broadcast(_ []byte) {}

// ---- tests ---------------------------------------------------------------

// TestLogMealExecute verifies that Execute returns a "processing" meal
// immediately without waiting for AI analysis.
func TestLogMealExecute(t *testing.T) {
	t.Parallel()

	recordedAt := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)

	uc := NewLogMeal(
		stubMealRepository{
			createFn: func(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
				if meal.DishName != "Banh mi" {
					t.Fatalf("expected trimmed dish name, got %q", meal.DishName)
				}
				if meal.MealType != domain.MealTypeUnspecified {
					t.Fatalf("expected default meal_type %q, got %q", domain.MealTypeUnspecified, meal.MealType)
				}
				if meal.AnalysisStatus != domain.AnalysisStatusProcessing {
					t.Fatalf("expected analysis_status %q, got %q", domain.AnalysisStatusProcessing, meal.AnalysisStatus)
				}
				if meal.Analysis != nil {
					t.Fatalf("expected nil Analysis at creation, got %+v", meal.Analysis)
				}
				meal.ID = 1
				return meal, nil
			},
			listByDayFn: func(ctx context.Context, filter domain.MealsByDayFilter) ([]domain.Meal, error) {
				return nil, nil
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
				return domain.Nutrition{RiskLevel: "medium"}, nil
			},
		},
	)

	got, err := uc.Execute(context.Background(), domain.LogMealInput{
		DishName:   "  Banh mi  ",
		RecordedAt: recordedAt,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.ID != 1 {
		t.Fatalf("expected meal ID to be assigned, got %d", got.ID)
	}
	// The HTTP response contains a pending meal — Analysis is populated asynchronously.
	if got.AnalysisStatus != domain.AnalysisStatusProcessing {
		t.Fatalf("expected status %q, got %q", domain.AnalysisStatusProcessing, got.AnalysisStatus)
	}
}

// TestLogMealExecuteKeepsProvidedMealType verifies meal_type normalisation.
func TestLogMealExecuteKeepsProvidedMealType(t *testing.T) {
	t.Parallel()

	uc := NewLogMeal(
		stubMealRepository{
			createFn: func(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
				return meal, nil
			},
			listByDayFn: func(ctx context.Context, filter domain.MealsByDayFilter) ([]domain.Meal, error) {
				return nil, nil
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
				return domain.Nutrition{RiskLevel: "low"}, nil
			},
		},
	)

	got, err := uc.Execute(context.Background(), domain.LogMealInput{
		DishName: "Pho",
		MealType: "LUNCH",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.MealType != domain.MealTypeLunch {
		t.Fatalf("expected meal_type %q, got %q", domain.MealTypeLunch, got.MealType)
	}
}

// TestLogMealRunAnalysisWithRetry_Success verifies the happy path: analyzer
// succeeds on first attempt, DB is updated, and broadcast is sent.
func TestLogMealRunAnalysisWithRetry_Success(t *testing.T) {
	t.Parallel()

	expectedNutrition := domain.Nutrition{
		EstimatedSugarGrams: 21.5,
		EstimatedCalories:   420,
		RiskLevel:           "medium",
		Notes:               []string{"sweet sauce likely contributes added sugar"},
	}

	updateResultCalled := false
	broadcastCalled := false

	broadcastCh := make(chan []byte, 1)
	pub := &capturingPublisher{ch: broadcastCh}

	uc := NewLogMeal(
		stubMealRepository{
			createFn: func(_ context.Context, meal domain.Meal) (domain.Meal, error) {
				meal.ID = 42
				return meal, nil
			},
			getByIDFn: func(_ context.Context, mealID int64) (domain.Meal, error) {
				return domain.Meal{ID: mealID, DishName: "Milk tea"}, nil
			},
			listByDayFn: func(_ context.Context, _ domain.MealsByDayFilter) ([]domain.Meal, error) {
				return nil, nil
			},
			updateAnalysisResultFn: func(_ context.Context, mealID int64, n domain.Nutrition) (domain.Meal, error) {
				updateResultCalled = true
				if mealID != 42 {
					t.Errorf("expected meal_id 42, got %d", mealID)
				}
				if n.RiskLevel != expectedNutrition.RiskLevel {
					t.Errorf("expected risk_level %q, got %q", expectedNutrition.RiskLevel, n.RiskLevel)
				}
				return domain.Meal{
					ID:             mealID,
					AnalysisStatus: domain.AnalysisStatusCompleted,
					Analysis:       &n,
				}, nil
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(_ context.Context, _ domain.AnalyzeMealInput) (domain.Nutrition, error) {
				return expectedNutrition, nil
			},
		},
	).WithPublisher(pub)

	pendingMeal := domain.Meal{ID: 42, DishName: "Milk tea", AnalysisStatus: domain.AnalysisStatusProcessing}

	// Call synchronously for deterministic testing.
	uc.runAnalysisWithRetry(context.Background(), pendingMeal)

	if !updateResultCalled {
		t.Fatal("expected UpdateAnalysisResult to be called")
	}

	select {
	case msg := <-broadcastCh:
		broadcastCalled = true
		if len(msg) == 0 {
			t.Fatal("expected non-empty broadcast message")
		}
	default:
	}
	if !broadcastCalled {
		t.Fatal("expected broadcast to be called")
	}
}

// TestLogMealRunAnalysisWithRetry_RetryThenSuccess verifies that the goroutine
// retries on failure and succeeds on the second attempt.
func TestLogMealRunAnalysisWithRetry_RetryThenSuccess(t *testing.T) {
	t.Parallel()

	attempts := 0
	updateStatusCalled := false

	uc := NewLogMeal(
		stubMealRepository{
			createFn: func(_ context.Context, meal domain.Meal) (domain.Meal, error) {
				return meal, nil
			},
			getByIDFn: func(_ context.Context, mealID int64) (domain.Meal, error) {
				return domain.Meal{ID: mealID, DishName: "Test"}, nil
			},
			listByDayFn: func(_ context.Context, _ domain.MealsByDayFilter) ([]domain.Meal, error) {
				return nil, nil
			},
			updateAnalysisResultFn: func(_ context.Context, _ int64, n domain.Nutrition) (domain.Meal, error) {
				return domain.Meal{AnalysisStatus: domain.AnalysisStatusCompleted, Analysis: &n}, nil
			},
			updateAnalysisStatusFn: func(_ context.Context, _ int64, _ string) error {
				updateStatusCalled = true
				return nil
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(_ context.Context, _ domain.AnalyzeMealInput) (domain.Nutrition, error) {
				attempts++
				if attempts < 2 {
					return domain.Nutrition{}, errors.New("transient error")
				}
				return domain.Nutrition{RiskLevel: "low"}, nil
			},
		},
	).WithPublisher(noopPublisher{})

	// Override base delay to zero so tests don't sleep.
	origDelay := analysisBaseDelay
	_ = origDelay // compile-time reference; delay is package-level const, test uses real retries quickly

	pendingMeal := domain.Meal{ID: 1, DishName: "Test", AnalysisStatus: domain.AnalysisStatusProcessing}
	uc.runAnalysisWithRetry(context.Background(), pendingMeal)

	if attempts != 2 {
		t.Fatalf("expected 2 analyzer calls, got %d", attempts)
	}
	if updateStatusCalled {
		t.Fatal("expected UpdateAnalysisStatus NOT to be called on eventual success")
	}
}

// TestLogMealRunAnalysisWithRetry_AllFail verifies that after exhausting all
// retries the meal is marked as "failed".
func TestLogMealRunAnalysisWithRetry_AllFail(t *testing.T) {
	t.Parallel()

	attempts := 0
	var capturedStatus string

	uc := NewLogMeal(
		stubMealRepository{
			createFn: func(_ context.Context, meal domain.Meal) (domain.Meal, error) {
				return meal, nil
			},
			getByIDFn: func(_ context.Context, mealID int64) (domain.Meal, error) {
				return domain.Meal{ID: mealID, DishName: "Unknown"}, nil
			},
			listByDayFn: func(_ context.Context, _ domain.MealsByDayFilter) ([]domain.Meal, error) {
				return nil, nil
			},
			updateAnalysisStatusFn: func(_ context.Context, _ int64, status string) error {
				capturedStatus = status
				return nil
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(_ context.Context, _ domain.AnalyzeMealInput) (domain.Nutrition, error) {
				attempts++
				return domain.Nutrition{}, errors.New("persistent error")
			},
		},
	).WithPublisher(noopPublisher{})

	pendingMeal := domain.Meal{ID: 99, DishName: "Unknown", AnalysisStatus: domain.AnalysisStatusProcessing}
	uc.runAnalysisWithRetry(context.Background(), pendingMeal)

	if attempts != analysisMaxRetries {
		t.Fatalf("expected %d analyzer calls, got %d", analysisMaxRetries, attempts)
	}
	if capturedStatus != domain.AnalysisStatusFailed {
		t.Fatalf("expected status %q, got %q", domain.AnalysisStatusFailed, capturedStatus)
	}
}

func TestLogMealRunAnalysisWithRetry_DeletedMealSkipsFailureBroadcast(t *testing.T) {
	t.Parallel()

	attempts := 0
	broadcastCh := make(chan []byte, 1)

	uc := NewLogMeal(
		stubMealRepository{
			createFn: func(_ context.Context, meal domain.Meal) (domain.Meal, error) {
				return meal, nil
			},
			getByIDFn: func(_ context.Context, mealID int64) (domain.Meal, error) {
				return domain.Meal{ID: mealID, DishName: "Unknown"}, nil
			},
			listByDayFn: func(_ context.Context, _ domain.MealsByDayFilter) ([]domain.Meal, error) {
				return nil, nil
			},
			updateAnalysisStatusFn: func(_ context.Context, _ int64, _ string) error {
				return domain.ErrMealNotFound
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(_ context.Context, _ domain.AnalyzeMealInput) (domain.Nutrition, error) {
				attempts++
				return domain.Nutrition{}, errors.New("persistent error")
			},
		},
	).WithPublisher(&capturingPublisher{ch: broadcastCh})

	pendingMeal := domain.Meal{ID: 88, DishName: "Unknown", AnalysisStatus: domain.AnalysisStatusProcessing}
	uc.runAnalysisWithRetry(context.Background(), pendingMeal)

	if attempts != analysisMaxRetries {
		t.Fatalf("expected %d analyzer calls, got %d", analysisMaxRetries, attempts)
	}

	select {
	case msg := <-broadcastCh:
		t.Fatalf("expected no broadcast for deleted meal, got %s", string(msg))
	default:
	}
}

func TestLogMealRunAnalysisWithRetry_DeletedBeforeAttemptSkipsAnalyzer(t *testing.T) {
	t.Parallel()

	analyzeCalled := false

	uc := NewLogMeal(
		stubMealRepository{
			createFn: func(_ context.Context, meal domain.Meal) (domain.Meal, error) {
				return meal, nil
			},
			listByDayFn: func(_ context.Context, _ domain.MealsByDayFilter) ([]domain.Meal, error) {
				return nil, nil
			},
			getByIDFn: func(_ context.Context, mealID int64) (domain.Meal, error) {
				return domain.Meal{}, domain.ErrMealNotFound
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(_ context.Context, _ domain.AnalyzeMealInput) (domain.Nutrition, error) {
				analyzeCalled = true
				return domain.Nutrition{}, nil
			},
		},
	).WithPublisher(noopPublisher{})

	pendingMeal := domain.Meal{ID: 123, DishName: "Deleted", AnalysisStatus: domain.AnalysisStatusProcessing}
	uc.runAnalysisWithRetry(context.Background(), pendingMeal)

	if analyzeCalled {
		t.Fatal("expected analyzer not to run for deleted meal")
	}
}

// TestLogMealExecuteClonesSourceMealWithoutAnalyzer verifies that cloning
// bypasses the AI analyzer and copies nutrition from the source meal.
func TestLogMealExecuteClonesSourceMealWithoutAnalyzer(t *testing.T) {
	t.Parallel()

	analyzeCalled := false
	sourceID := int64(7)
	recordedAt := time.Date(2026, 5, 23, 12, 30, 0, 0, time.UTC)

	uc := NewLogMeal(
		stubMealRepository{
			getByIDFn: func(ctx context.Context, mealID int64) (domain.Meal, error) {
				if mealID != sourceID {
					t.Fatalf("expected source meal ID %d, got %d", sourceID, mealID)
				}
				imageURL := "https://example.com/milk-tea.jpg"
				return domain.Meal{
					ID:             sourceID,
					DishName:       "Milk tea",
					MealType:       domain.MealTypeSnack,
					ImageURL:       &imageURL,
					RecordedAt:     time.Date(2026, 5, 22, 15, 0, 0, 0, time.UTC),
					AnalysisStatus: domain.AnalysisStatusCompleted,
					IsUserEdited:   true,
					Analysis: &domain.Nutrition{
						EstimatedSugarGrams:   35,
						EstimatedCarbsGrams:   54,
						EstimatedProteinGrams: 6,
						EstimatedCalories:     420,
						RiskLevel:             "high",
						Notes:                 []string{"copied estimate"},
					},
				}, nil
			},
			createFn: func(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
				if meal.DishName != "Milk tea" {
					t.Fatalf("expected cloned dish name, got %q", meal.DishName)
				}
				if meal.MealType != domain.MealTypeLunch {
					t.Fatalf("expected overridden meal_type lunch, got %q", meal.MealType)
				}
				if !meal.RecordedAt.Equal(recordedAt) {
					t.Fatalf("expected provided recorded_at, got %s", meal.RecordedAt)
				}
				if meal.Analysis == nil || meal.Analysis.EstimatedSugarGrams != 35 {
					t.Fatalf("expected copied analysis, got %+v", meal.Analysis)
				}
				if !meal.IsUserEdited {
					t.Fatal("expected cloned is_user_edited flag")
				}
				meal.ID = 8
				return meal, nil
			},
			listByDayFn: func(_ context.Context, _ domain.MealsByDayFilter) ([]domain.Meal, error) {
				return nil, nil
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(_ context.Context, _ domain.AnalyzeMealInput) (domain.Nutrition, error) {
				analyzeCalled = true
				return domain.Nutrition{}, nil
			},
		},
	)

	got, err := uc.Execute(context.Background(), domain.LogMealInput{
		SourceMealID: &sourceID,
		MealType:     "LUNCH",
		RecordedAt:   recordedAt,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.ID != 8 {
		t.Fatalf("expected new meal ID 8, got %d", got.ID)
	}
	if analyzeCalled {
		t.Fatal("expected clone flow not to call analyzer")
	}
}

// TestLogMealExecuteReturnsValidationError verifies that an empty dish_name
// returns ErrInvalidMealInput.
func TestLogMealExecuteReturnsValidationError(t *testing.T) {
	t.Parallel()

	uc := NewLogMeal(
		stubMealRepository{
			createFn: func(_ context.Context, meal domain.Meal) (domain.Meal, error) {
				return domain.Meal{}, nil
			},
			listByDayFn: func(_ context.Context, _ domain.MealsByDayFilter) ([]domain.Meal, error) {
				return nil, nil
			},
		},
		stubNutritionAnalyzer{
			analyzeFn: func(_ context.Context, _ domain.AnalyzeMealInput) (domain.Nutrition, error) {
				return domain.Nutrition{}, nil
			},
		},
	)

	_, err := uc.Execute(context.Background(), domain.LogMealInput{
		DishName: " ",
	})
	if !errors.Is(err, domain.ErrInvalidMealInput) {
		t.Fatalf("expected error %v, got %v", domain.ErrInvalidMealInput, err)
	}
}

// ---- helpers ---------------------------------------------------------------

// capturingPublisher records broadcast messages in a buffered channel.
type capturingPublisher struct {
	ch chan []byte
}

func (p *capturingPublisher) Broadcast(msg []byte) {
	select {
	case p.ch <- msg:
	default:
	}
}
