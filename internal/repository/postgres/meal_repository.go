package postgres

import (
	"context"
	"strings"
	"time"

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
		DishName:            meal.DishName,
		ImageUrl:            meal.ImageURL,
		RecordedAt:          pgtype.Timestamptz{Time: meal.RecordedAt.UTC(), Valid: true},
		AnalysisStatus:      meal.AnalysisStatus,
		EstimatedSugarGrams: meal.Analysis.EstimatedSugarGrams,
		EstimatedCalories:   int32(meal.Analysis.EstimatedCalories),
		RiskLevel:           meal.Analysis.RiskLevel,
		AnalysisNotes:       strings.Join(meal.Analysis.Notes, "\n"),
	}

	row, err := r.queries.CreateMeal(ctx, params)
	if err != nil {
		return domain.Meal{}, err
	}

	return mapMealRow(row), nil
}

func (r MealRepository) ListByDay(ctx context.Context, day time.Time) ([]domain.Meal, error) {
	rows, err := r.queries.ListMealsByDay(ctx, reposqlc.ListMealsByDayParams{
		DayStart: pgtype.Timestamptz{Time: startOfDayUTC(day), Valid: true},
		DayEnd:   pgtype.Timestamptz{Time: startOfDayUTC(day).Add(24 * time.Hour), Valid: true},
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

func mapMealRow(row reposqlc.Meal) domain.Meal {
	return domain.Meal{
		ID:             row.ID,
		DishName:       row.DishName,
		ImageURL:       row.ImageUrl,
		RecordedAt:     row.RecordedAt.Time,
		AnalysisStatus: row.AnalysisStatus,
		Analysis: &domain.Nutrition{
			EstimatedSugarGrams: row.EstimatedSugarGrams,
			EstimatedCalories:   int(row.EstimatedCalories),
			RiskLevel:           row.RiskLevel,
			Notes:               splitNotes(row.AnalysisNotes),
		},
	}
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
