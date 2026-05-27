package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"sugary/internal/domain"
	"sugary/internal/platform/timeutil"
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
	insightsJSON, err := json.Marshal(report.AIInsights)
	if err != nil {
		return err
	}

	return r.queries.UpsertDailyReport(ctx, reposqlc.UpsertDailyReportParams{
		ReportDate:        pgtype.Date{Time: timeutil.CanonicalUTCDate(report.Date), Valid: true},
		MealCount:         int32(report.MealCount),
		TotalSugarGrams:   report.TotalSugarGrams,
		AverageSugarGrams: report.AverageSugarGrams,
		HighestRiskLevel:  report.HighestRiskLevel,
		Summary:           report.Summary,
		AiInsights:        insightsJSON,
	})
}

func (r DailyReportRepository) GetByDay(ctx context.Context, day time.Time) (domain.DailyReport, bool, error) {
	row, err := r.queries.GetDailyReportByDate(ctx, pgtype.Date{Time: timeutil.CanonicalUTCDate(day), Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DailyReport{}, false, nil
		}
		return domain.DailyReport{}, false, err
	}

	report := domain.DailyReport{
		Date:              row.ReportDate.Time,
		MealCount:         int(row.MealCount),
		TotalSugarGrams:   row.TotalSugarGrams,
		AverageSugarGrams: row.AverageSugarGrams,
		HighestRiskLevel:  row.HighestRiskLevel,
		Summary:           row.Summary,
		AIInsights:        domain.DailyReportAIInsights{},
	}

	if len(row.AiInsights) > 0 {
		if err := json.Unmarshal(row.AiInsights, &report.AIInsights); err != nil {
			report.AIInsights = domain.DailyReportAIInsights{}
		}
	}
	if report.AIInsights.Summary == "" {
		report.AIInsights = domain.DailyReportAIInsights{
			Summary:         report.Summary,
			TopContributors: []domain.DailyReportTopContributor{},
			Recommendations: []string{},
			PatternSignals:  []string{},
		}
	}

	return report, true, nil
}
