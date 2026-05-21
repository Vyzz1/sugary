package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"sugary/internal/domain"
	reposqlc "sugary/internal/repository/postgres/sqlc"
)

type DailyReportRepository struct {
	queries *reposqlc.Queries
}

func NewDailyReportRepository(queries *reposqlc.Queries) DailyReportRepository {
	return DailyReportRepository{
		queries: queries,
	}
}

func (r DailyReportRepository) Save(ctx context.Context, report domain.DailyReport) error {
	return r.queries.UpsertDailyReport(ctx, reposqlc.UpsertDailyReportParams{
		ReportDate:        pgtype.Date{Time: report.Date.UTC(), Valid: true},
		MealCount:         int32(report.MealCount),
		TotalSugarGrams:   report.TotalSugarGrams,
		AverageSugarGrams: report.AverageSugarGrams,
		HighestRiskLevel:  report.HighestRiskLevel,
		Summary:           report.Summary,
	})
}

func (r DailyReportRepository) GetByDay(ctx context.Context, day time.Time) (domain.DailyReport, bool, error) {
	row, err := r.queries.GetDailyReportByDate(ctx, pgtype.Date{Time: startOfDayUTC(day), Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DailyReport{}, false, nil
		}
		return domain.DailyReport{}, false, err
	}

	return domain.DailyReport{
		Date:              row.ReportDate.Time,
		MealCount:         int(row.MealCount),
		TotalSugarGrams:   row.TotalSugarGrams,
		AverageSugarGrams: row.AverageSugarGrams,
		HighestRiskLevel:  row.HighestRiskLevel,
		Summary:           row.Summary,
	}, true, nil
}
