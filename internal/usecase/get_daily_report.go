package usecase

import (
	"context"
	"time"

	"sugary/internal/domain"
	"sugary/internal/platform/timeutil"
)

type GetDailyReport struct {
	dailyReportRepository domain.DailyReportRepository
}

func NewGetDailyReport(dailyReportRepository domain.DailyReportRepository) GetDailyReport {
	return GetDailyReport{
		dailyReportRepository: dailyReportRepository,
	}
}

func (uc GetDailyReport) Execute(ctx context.Context, day time.Time) (domain.DailyReport, bool, error) {
	if day.IsZero() {
		return domain.DailyReport{}, false, domain.ErrInvalidDate
	}

	return uc.dailyReportRepository.GetByDay(ctx, timeutil.CanonicalUTCDate(day))
}
