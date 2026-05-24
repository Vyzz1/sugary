package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"sugary/internal/domain"
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
	params := reposqlc.CreateMealParams{
		DishName:              meal.DishName,
		MealType:              meal.MealType,
		ImageUrl:              meal.ImageURL,
		RecordedAt:            pgtype.Timestamptz{Time: meal.RecordedAt.UTC(), Valid: true},
		AnalysisStatus:        meal.AnalysisStatus,
		EstimatedSugarGrams:   meal.Analysis.EstimatedSugarGrams,
		EstimatedCarbsGrams:   meal.Analysis.EstimatedCarbsGrams,
		EstimatedProteinGrams: meal.Analysis.EstimatedProteinGrams,
		EstimatedCalories:     int32(meal.Analysis.EstimatedCalories),
		RiskLevel:             meal.Analysis.RiskLevel,
		AnalysisNotes:         strings.Join(meal.Analysis.Notes, "\n"),
		IsUserEdited:          meal.IsUserEdited,
	}

	row, err := r.queries.CreateMeal(ctx, params)
	if err != nil {
		return domain.Meal{}, err
	}

	return mapMealRow(row), nil
}

func (r MealRepository) ListByDay(ctx context.Context, filter domain.MealsByDayFilter) ([]domain.Meal, error) {
	rows, err := r.queries.ListMealsByDay(ctx, reposqlc.ListMealsByDayParams{
		DayStart: pgtype.Timestamptz{Time: startOfDayUTC(filter.Day), Valid: true},
		DayEnd:   pgtype.Timestamptz{Time: startOfDayUTC(filter.Day).Add(24 * time.Hour), Valid: true},
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
		ID:             row.ID,
		DishName:       row.DishName,
		MealType:       row.MealType,
		ImageURL:       row.ImageUrl,
		RecordedAt:     row.RecordedAt.Time,
		AnalysisStatus: row.AnalysisStatus,
		IsUserEdited:   row.IsUserEdited,
		DeletedAt:      nullableTime(row.DeletedAt),
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
	return domain.Meal{
		ID:             row.ID,
		DishName:       row.DishName,
		MealType:       row.MealType,
		ImageURL:       row.ImageUrl,
		RecordedAt:     row.RecordedAt.Time,
		AnalysisStatus: row.AnalysisStatus,
		IsUserEdited:   row.IsUserEdited,
		DeletedAt:      nullableTime(row.DeletedAt),
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

func startOfDayUTC(day time.Time) time.Time {
	day = day.UTC()
	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
}
