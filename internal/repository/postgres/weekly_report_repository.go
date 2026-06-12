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

type WeeklyReportRepository struct {
	queries *reposqlc.Queries
}

func NewWeeklyReportRepository(queries *reposqlc.Queries) WeeklyReportRepository {
	return WeeklyReportRepository{
		queries: queries,
	}
}

func (r WeeklyReportRepository) Save(ctx context.Context, report domain.WeeklyReport) error {
	breakdownJSON, err := json.Marshal(report.DailyBreakdown)
	if err != nil {
		return err
	}
	insightsJSON, err := json.Marshal(report.AIInsights)
	if err != nil {
		return err
	}

	createdAt := report.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	return r.queries.UpsertWeeklyReport(ctx, reposqlc.UpsertWeeklyReportParams{
		WeekStartDate:     pgtype.Date{Time: timeutil.CanonicalUTCDate(report.WeekStartDate), Valid: true},
		WeekEndDate:       pgtype.Date{Time: timeutil.CanonicalUTCDate(report.WeekEndDate), Valid: true},
		CreatedAt:         pgtype.Timestamptz{Time: createdAt.UTC(), Valid: true},
		MealCount:         int32(report.MealCount),
		AnalyzedMealCount: int32(report.AnalyzedMealCount),
		TotalSugarGrams:   report.TotalSugarGrams,
		AverageSugarGrams: report.AverageSugarGrams,
		HighestRiskLevel:  report.HighestRiskLevel,
		Summary:           report.Summary,
		DailyBreakdown:    breakdownJSON,
		AiInsights:        insightsJSON,
		AiInsightSource:   report.AIInsightSource,
		AiInsightStatus:   report.AIInsightStatus,
	})
}

func (r WeeklyReportRepository) GetByWeekStart(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error) {
	row, err := r.queries.GetWeeklyReportByWeekStart(ctx, pgtype.Date{Time: timeutil.CanonicalUTCDate(weekStart), Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.WeeklyReport{}, false, nil
		}
		return domain.WeeklyReport{}, false, err
	}

	report := domain.WeeklyReport{
		WeekStartDate:     row.WeekStartDate.Time,
		WeekEndDate:       row.WeekEndDate.Time,
		CreatedAt:         row.CreatedAt.Time,
		MealCount:         int(row.MealCount),
		AnalyzedMealCount: int(row.AnalyzedMealCount),
		TotalSugarGrams:   row.TotalSugarGrams,
		AverageSugarGrams: row.AverageSugarGrams,
		HighestRiskLevel:  row.HighestRiskLevel,
		Summary:           row.Summary,
		DailyBreakdown:    []domain.WeeklyReportDaily{},
		AIInsightSource:   row.AiInsightSource,
		AIInsightStatus:   row.AiInsightStatus,
		AIInsights:        domain.WeeklyReportAIInsights{},
	}

	if len(row.DailyBreakdown) > 0 {
		if err := json.Unmarshal(row.DailyBreakdown, &report.DailyBreakdown); err != nil {
			report.DailyBreakdown = []domain.WeeklyReportDaily{}
		}
	}
	if len(row.AiInsights) > 0 {
		if err := json.Unmarshal(row.AiInsights, &report.AIInsights); err != nil {
			report.AIInsights = domain.WeeklyReportAIInsights{}
		}
	}
	if report.AIInsights.Summary == "" {
		report.AIInsights = fallbackWeeklyInsights(report.Summary)
	}
	if report.AIInsightSource == "" {
		report.AIInsightSource = "fallback"
	}
	if report.AIInsightStatus == "" {
		report.AIInsightStatus = "fallback"
	}

	return report, true, nil
}

func fallbackWeeklyInsights(summary string) domain.WeeklyReportAIInsights {
	return domain.WeeklyReportAIInsights{
		Summary:         summary,
		TopContributors: []domain.WeeklyReportTopContributor{},
		Recommendations: []string{},
		PatternSignals:  []string{},
	}
}
