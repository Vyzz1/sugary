package usecase

import (
	"context"

	"sugary/internal/domain"
)

type stubReportEmailSender struct {
	sendDailyFn  func(ctx context.Context, report domain.DailyReport) error
	sendWeeklyFn func(ctx context.Context, report domain.WeeklyReport) error
}

func (s stubReportEmailSender) SendDailyReport(ctx context.Context, report domain.DailyReport) error {
	if s.sendDailyFn == nil {
		return nil
	}
	return s.sendDailyFn(ctx, report)
}

func (s stubReportEmailSender) SendWeeklyReport(ctx context.Context, report domain.WeeklyReport) error {
	if s.sendWeeklyFn == nil {
		return nil
	}
	return s.sendWeeklyFn(ctx, report)
}
