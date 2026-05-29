package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"sugary/internal/domain"
	"sugary/internal/platform/timeutil"
	reposqlc "sugary/internal/repository/postgres/sqlc"
)

type MealRepository struct {
	queries *reposqlc.Queries
}

func NewMealRepository(queries *reposqlc.Queries) MealRepository {
	return MealRepository{
		queries: queries,
	}
}

func (r MealRepository) Create(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
	// Nutrition fields may be zero when analysis_status = 'processing'.
	var sugarGrams, carbsGrams, proteinGrams float64
	var calories int32
	var riskLevel, analysisNotes string
	if meal.Analysis != nil {
		sugarGrams = meal.Analysis.EstimatedSugarGrams
		carbsGrams = meal.Analysis.EstimatedCarbsGrams
		proteinGrams = meal.Analysis.EstimatedProteinGrams
		calories = int32(meal.Analysis.EstimatedCalories)
		riskLevel = meal.Analysis.RiskLevel
		analysisNotes = strings.Join(meal.Analysis.Notes, "\n")
	}

	params := reposqlc.CreateMealParams{
		DishName:              meal.DishName,
		MealType:              meal.MealType,
		ImageUrl:              meal.ImageURL,
		RecordedAt:            pgtype.Timestamptz{Time: meal.RecordedAt.UTC(), Valid: true},
		AnalysisStatus:        meal.AnalysisStatus,
		EstimatedSugarGrams:   sugarGrams,
		EstimatedCarbsGrams:   carbsGrams,
		EstimatedProteinGrams: proteinGrams,
		EstimatedCalories:     calories,
		RiskLevel:             riskLevel,
		AnalysisNotes:         analysisNotes,
		IsUserEdited:          meal.IsUserEdited,
	}

	row, err := r.queries.CreateMeal(ctx, params)
	if err != nil {
		return domain.Meal{}, err
	}

	return mapMealRow(row), nil
}

func (r MealRepository) ListByDay(ctx context.Context, filter domain.MealsByDayFilter) ([]domain.Meal, error) {
	dayStart, dayEnd := timeutil.DayBoundsUTC(filter.Day)

	rows, err := r.queries.ListMealsByDay(ctx, reposqlc.ListMealsByDayParams{
		DayStart: pgtype.Timestamptz{Time: dayStart, Valid: true},
		DayEnd:   pgtype.Timestamptz{Time: dayEnd, Valid: true},
		SortType: filter.Sort,
	})
	if err != nil {
		return nil, err
	}

	meals := make([]domain.Meal, 0, len(rows))
	for _, row := range rows {
		meals = append(meals, mapMealRow(row))
	}

	return meals, nil
}

func (r MealRepository) ListRecentDistinct(ctx context.Context, filter domain.RecentMealsFilter) ([]domain.Meal, int64, error) {
	total, err := r.queries.CountRecentDistinctMeals(ctx, filter.Query)
	if err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.PageSize
	rows, err := r.queries.ListRecentDistinctMeals(ctx, reposqlc.ListRecentDistinctMealsParams{
		QueryText:   filter.Query,
		SortType:    filter.Sort,
		LimitCount:  filter.PageSize,
		OffsetCount: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	meals := make([]domain.Meal, 0, len(rows))
	for _, row := range rows {
		meals = append(meals, mapRecentMealRow(row))
	}

	return meals, total, nil
}

func mapRecentMealRow(row reposqlc.ListRecentDistinctMealsRow) domain.Meal {
	return domain.Meal{
		ID:                    row.ID,
		DishName:              row.DishName,
		MealType:              row.MealType,
		ImageURL:              row.ImageUrl,
		RecordedAt:            row.RecordedAt.Time,
		AnalysisStatus:        row.AnalysisStatus,
		IsUserEdited:          row.IsUserEdited,
		AnalysisRetryCount:    row.AnalysisRetryCount,
		LastAnalysisAttemptAt: nullableTime(row.LastAnalysisAttemptAt),
		DeletedAt:             nullableTime(row.DeletedAt),
		Analysis: &domain.Nutrition{
			EstimatedSugarGrams:   row.EstimatedSugarGrams,
			EstimatedCarbsGrams:   row.EstimatedCarbsGrams,
			EstimatedProteinGrams: row.EstimatedProteinGrams,
			EstimatedCalories:     int(row.EstimatedCalories),
			RiskLevel:             row.RiskLevel,
			Notes:                 splitNotes(row.AnalysisNotes),
		},
	}
}

func (r MealRepository) GetByID(ctx context.Context, mealID int64) (domain.Meal, error) {
	row, err := r.queries.GetMealByID(ctx, mealID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Meal{}, domain.ErrMealNotFound
		}
		return domain.Meal{}, err
	}
	return mapMealRow(row), nil
}

func (r MealRepository) ListRetryableFailed(ctx context.Context, filter domain.RetryableFailedMealsFilter) ([]domain.Meal, error) {
	rows, err := r.queries.ListRetryableFailedMeals(ctx, reposqlc.ListRetryableFailedMealsParams{
		MaxRetryCount: filter.MaxRetryCount,
		BeforeTime:    pgtype.Timestamptz{Time: filter.Before.UTC(), Valid: true},
		LimitCount:    filter.Limit,
	})
	if err != nil {
		return nil, err
	}

	meals := make([]domain.Meal, 0, len(rows))
	for _, row := range rows {
		meals = append(meals, mapMealRow(row))
	}

	return meals, nil
}

func (r MealRepository) UpdateMeta(ctx context.Context, mealID int64, mealType string, recordedAt time.Time) (domain.Meal, error) {
	row, err := r.queries.UpdateMealMetaByID(ctx, reposqlc.UpdateMealMetaByIDParams{
		ID:         mealID,
		MealType:   mealType,
		RecordedAt: pgtype.Timestamptz{Time: recordedAt.UTC(), Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Meal{}, domain.ErrMealNotFound
		}
		return domain.Meal{}, err
	}
	return mapMealRow(row), nil
}

func (r MealRepository) UpdateForReanalysis(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
	row, err := r.queries.UpdateMealForReanalysisByID(ctx, reposqlc.UpdateMealForReanalysisByIDParams{
		ID:         meal.ID,
		DishName:   meal.DishName,
		MealType:   meal.MealType,
		ImageUrl:   meal.ImageURL,
		RecordedAt: pgtype.Timestamptz{Time: meal.RecordedAt.UTC(), Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Meal{}, domain.ErrMealNotFound
		}
		return domain.Meal{}, err
	}
	return mapMealRow(row), nil
}

func (r MealRepository) UpdateWithAnalysis(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
	row, err := r.queries.UpdateMealWithAnalysisByID(ctx, reposqlc.UpdateMealWithAnalysisByIDParams{
		ID:                    meal.ID,
		DishName:              meal.DishName,
		MealType:              meal.MealType,
		ImageUrl:              meal.ImageURL,
		RecordedAt:            pgtype.Timestamptz{Time: meal.RecordedAt.UTC(), Valid: true},
		EstimatedSugarGrams:   meal.Analysis.EstimatedSugarGrams,
		EstimatedCarbsGrams:   meal.Analysis.EstimatedCarbsGrams,
		EstimatedProteinGrams: meal.Analysis.EstimatedProteinGrams,
		EstimatedCalories:     int32(meal.Analysis.EstimatedCalories),
		RiskLevel:             meal.Analysis.RiskLevel,
		AnalysisNotes:         strings.Join(meal.Analysis.Notes, "\n"),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Meal{}, domain.ErrMealNotFound
		}
		return domain.Meal{}, err
	}
	return mapMealRow(row), nil
}

func (r MealRepository) UpdateAnalysis(ctx context.Context, mealID int64, nutrition domain.Nutrition) (domain.Meal, error) {
	row, err := r.queries.UpdateMealAnalysisByID(ctx, reposqlc.UpdateMealAnalysisByIDParams{
		ID:                    mealID,
		EstimatedSugarGrams:   nutrition.EstimatedSugarGrams,
		EstimatedCarbsGrams:   nutrition.EstimatedCarbsGrams,
		EstimatedProteinGrams: nutrition.EstimatedProteinGrams,
		EstimatedCalories:     int32(nutrition.EstimatedCalories),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Meal{}, domain.ErrMealNotFound
		}
		return domain.Meal{}, err
	}

	return mapMealRow(row), nil
}

// UpdateAnalysisResult sets all nutrition fields and marks the meal as 'completed'.
// Called by the async goroutine after AI analysis succeeds.
func (r MealRepository) UpdateAnalysisResult(ctx context.Context, mealID int64, nutrition domain.Nutrition) (domain.Meal, error) {
	row, err := r.queries.UpdateMealAnalysisResultByID(ctx, reposqlc.UpdateMealAnalysisResultByIDParams{
		ID:                    mealID,
		EstimatedSugarGrams:   nutrition.EstimatedSugarGrams,
		EstimatedCarbsGrams:   nutrition.EstimatedCarbsGrams,
		EstimatedProteinGrams: nutrition.EstimatedProteinGrams,
		EstimatedCalories:     int32(nutrition.EstimatedCalories),
		RiskLevel:             nutrition.RiskLevel,
		AnalysisNotes:         strings.Join(nutrition.Notes, "\n"),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Meal{}, domain.ErrMealNotFound
		}
		return domain.Meal{}, err
	}
	return mapMealRow(row), nil
}

func (r MealRepository) RetryFailedAnalysis(ctx context.Context, mealID int64) (domain.Meal, error) {
	row, err := r.queries.RetryFailedMealAnalysisByID(ctx, mealID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Meal{}, domain.ErrMealNotFound
		}
		return domain.Meal{}, err
	}
	return mapMealRow(row), nil
}

// UpdateAnalysisStatus updates only the analysis_status column.
// Called by the async goroutine when all retries are exhausted.
func (r MealRepository) UpdateAnalysisStatus(ctx context.Context, mealID int64, status string) error {
	affected, err := r.queries.UpdateMealAnalysisStatusByID(ctx, reposqlc.UpdateMealAnalysisStatusByIDParams{
		ID:             mealID,
		AnalysisStatus: status,
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.ErrMealNotFound
	}
	return nil
}

func (r MealRepository) SoftDelete(ctx context.Context, mealID int64) error {
	affected, err := r.queries.SoftDeleteMealByID(ctx, mealID)
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.ErrMealNotFound
	}
	return nil
}

func mapMealRow(row reposqlc.Meal) domain.Meal {
	meal := domain.Meal{
		ID:                    row.ID,
		DishName:              row.DishName,
		MealType:              row.MealType,
		ImageURL:              row.ImageUrl,
		RecordedAt:            row.RecordedAt.Time,
		AnalysisStatus:        row.AnalysisStatus,
		IsUserEdited:          row.IsUserEdited,
		AnalysisRetryCount:    row.AnalysisRetryCount,
		LastAnalysisAttemptAt: nullableTime(row.LastAnalysisAttemptAt),
		DeletedAt:             nullableTime(row.DeletedAt),
	}
	// Only populate Analysis when the AI analysis is done.
	// Pending and failed meals intentionally return nil Analysis.
	if row.AnalysisStatus == domain.AnalysisStatusCompleted {
		meal.Analysis = &domain.Nutrition{
			EstimatedSugarGrams:   row.EstimatedSugarGrams,
			EstimatedCarbsGrams:   row.EstimatedCarbsGrams,
			EstimatedProteinGrams: row.EstimatedProteinGrams,
			EstimatedCalories:     int(row.EstimatedCalories),
			RiskLevel:             row.RiskLevel,
			Notes:                 splitNotes(row.AnalysisNotes),
		}
	}
	return meal
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

func splitNotes(value string) []string {
	if value == "" {
		return nil
	}

	return strings.Split(value, "\n")
}
