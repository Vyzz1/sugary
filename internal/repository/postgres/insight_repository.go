package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"sugary/internal/domain"
	reposqlc "sugary/internal/repository/postgres/sqlc"
)

type InsightRepository struct {
	queries *reposqlc.Queries
}

func NewInsightRepository(queries *reposqlc.Queries) InsightRepository {
	return InsightRepository{queries: queries}
}

func (r InsightRepository) GetPeriodStats(ctx context.Context, filter domain.InsightPeriodFilter) (domain.InsightPeriodStats, error) {
	row, err := r.queries.GetInsightPeriodStats(ctx, reposqlc.GetInsightPeriodStatsParams{
		FromInclusive: timestamptz(filter.FromInclusive),
		ToExclusive:   timestamptz(filter.ToExclusive),
	})
	if err != nil {
		return domain.InsightPeriodStats{}, err
	}

	return domain.InsightPeriodStats{
		TotalSugarGrams: row.TotalSugarGrams,
		MealCount:       row.MealCount,
		HighRiskMeals:   row.HighRiskMeals,
	}, nil
}

func (r InsightRepository) GetDailySugar(ctx context.Context, filter domain.InsightPeriodFilter) ([]domain.DailySugarPoint, error) {
	rows, err := r.queries.GetInsightDailySugar(ctx, reposqlc.GetInsightDailySugarParams{
		TimezoneName:  filter.Timezone,
		FromInclusive: timestamptz(filter.FromInclusive),
		ToExclusive:   timestamptz(filter.ToExclusive),
	})
	if err != nil {
		return nil, err
	}

	points := make([]domain.DailySugarPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, domain.DailySugarPoint{
			Date:              dateString(row.Date),
			TotalSugarGrams:   row.TotalSugarGrams,
			TotalCalories:     row.TotalCalories,
			TotalCarbsGrams:   row.TotalCarbsGrams,
			TotalProteinGrams: row.TotalProteinGrams,
			MealCount:         row.MealCount,
			RiskLevel:         row.RiskLevel,
		})
	}
	return points, nil
}

func (r InsightRepository) GetMealTypeBreakdown(ctx context.Context, filter domain.InsightPeriodFilter) ([]domain.MealTypeBreakdown, error) {
	rows, err := r.queries.GetInsightMealTypeBreakdown(ctx, reposqlc.GetInsightMealTypeBreakdownParams{
		FromInclusive: timestamptz(filter.FromInclusive),
		ToExclusive:   timestamptz(filter.ToExclusive),
	})
	if err != nil {
		return nil, err
	}

	items := make([]domain.MealTypeBreakdown, 0, len(rows))
	for _, row := range rows {
		items = append(items, domain.MealTypeBreakdown{
			MealType:          row.MealType,
			TotalSugarGrams:   row.TotalSugarGrams,
			MealCount:         row.MealCount,
			AverageSugarGrams: row.AverageSugarGrams,
		})
	}
	return items, nil
}

func (r InsightRepository) GetRiskDistribution(ctx context.Context, filter domain.InsightPeriodFilter) ([]domain.RiskDistributionPoint, error) {
	rows, err := r.queries.GetInsightRiskDistribution(ctx, reposqlc.GetInsightRiskDistributionParams{
		FromInclusive: timestamptz(filter.FromInclusive),
		ToExclusive:   timestamptz(filter.ToExclusive),
	})
	if err != nil {
		return nil, err
	}

	points := make([]domain.RiskDistributionPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, domain.RiskDistributionPoint{
			RiskLevel: row.RiskLevel,
			Count:     row.Count,
		})
	}
	return points, nil
}

func (r InsightRepository) GetWeeklySugar(ctx context.Context, filter domain.InsightPeriodFilter) ([]domain.WeeklySugarPoint, error) {
	rows, err := r.queries.GetInsightWeeklySugar(ctx, reposqlc.GetInsightWeeklySugarParams{
		TimezoneName:  filter.Timezone,
		FromInclusive: timestamptz(filter.FromInclusive),
		ToExclusive:   timestamptz(filter.ToExclusive),
	})
	if err != nil {
		return nil, err
	}

	points := make([]domain.WeeklySugarPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, domain.WeeklySugarPoint{
			Week:            isoWeekLabel(row.WeekStart),
			TotalSugarGrams: row.TotalSugarGrams,
			AveragePerDay:   row.AveragePerDay,
			MealCount:       row.MealCount,
			HighRiskMeals:   row.HighRiskMeals,
		})
	}
	return points, nil
}

func (r InsightRepository) GetSugarVsCalories(ctx context.Context, filter domain.InsightPeriodFilter) ([]domain.SugarVsCaloriesPoint, error) {
	rows, err := r.queries.GetInsightSugarVsCalories(ctx, reposqlc.GetInsightSugarVsCaloriesParams{
		FromInclusive: timestamptz(filter.FromInclusive),
		ToExclusive:   timestamptz(filter.ToExclusive),
	})
	if err != nil {
		return nil, err
	}

	points := make([]domain.SugarVsCaloriesPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, domain.SugarVsCaloriesPoint{
			MealID:     row.MealID,
			DishName:   row.DishName,
			SugarGrams: row.SugarGrams,
			Calories:   int(row.Calories),
			RiskLevel:  row.RiskLevel,
			RecordedAt: row.RecordedAt.Time,
		})
	}
	return points, nil
}

func (r InsightRepository) GetTopSugarMeals(ctx context.Context, filter domain.InsightPeriodFilter) ([]domain.TopSugarMeal, error) {
	rows, err := r.queries.GetInsightTopSugarMeals(ctx, reposqlc.GetInsightTopSugarMealsParams{
		FromInclusive: timestamptz(filter.FromInclusive),
		ToExclusive:   timestamptz(filter.ToExclusive),
	})
	if err != nil {
		return nil, err
	}

	meals := make([]domain.TopSugarMeal, 0, len(rows))
	for _, row := range rows {
		meals = append(meals, domain.TopSugarMeal{
			MealID:     row.MealID,
			DishName:   row.DishName,
			SugarGrams: row.SugarGrams,
			Calories:   int(row.Calories),
			MealType:   row.MealType,
			RiskLevel:  row.RiskLevel,
			RecordedAt: row.RecordedAt.Time,
		})
	}
	return meals, nil
}

func (r InsightRepository) GetTopSugarDishes(ctx context.Context, filter domain.InsightPeriodFilter) ([]domain.TopSugarDish, error) {
	rows, err := r.queries.GetInsightTopSugarDishes(ctx, reposqlc.GetInsightTopSugarDishesParams{
		FromInclusive: timestamptz(filter.FromInclusive),
		ToExclusive:   timestamptz(filter.ToExclusive),
	})
	if err != nil {
		return nil, err
	}

	dishes := make([]domain.TopSugarDish, 0, len(rows))
	for _, row := range rows {
		dishes = append(dishes, domain.TopSugarDish{
			DishName:          row.DishName,
			TimesLogged:       row.TimesLogged,
			TotalSugarGrams:   row.TotalSugarGrams,
			AverageSugarGrams: row.AverageSugarGrams,
		})
	}
	return dishes, nil
}

func timestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

func dateString(value pgtype.Date) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format("2006-01-02")
}

func isoWeekLabel(value pgtype.Date) string {
	if !value.Valid {
		return ""
	}
	year, week := value.Time.ISOWeek()
	return fmt.Sprintf("%04d-W%02d", year, week)
}
