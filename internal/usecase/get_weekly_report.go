package usecase

import (
	"context"
	"time"

	"sugary/internal/domain"
	"sugary/internal/platform/timeutil"
)

type GetWeeklyReport struct {
	weeklyReportRepository domain.WeeklyReportRepository
}

func NewGetWeeklyReport(weeklyReportRepository domain.WeeklyReportRepository) GetWeeklyReport {
	return GetWeeklyReport{
		weeklyReportRepository: weeklyReportRepository,
	}
}

func (uc GetWeeklyReport) Execute(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error) {
	if weekStart.IsZero() {
		return domain.WeeklyReport{}, false, domain.ErrInvalidDate
	}

	return uc.weeklyReportRepository.GetByWeekStart(ctx, timeutil.CanonicalUTCDate(weekStart))
}
