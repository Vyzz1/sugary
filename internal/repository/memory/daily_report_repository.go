package memory

import (
	"context"
	"sync"
	"time"

	"sugary/internal/domain"
	"sugary/internal/platform/timeutil"
)

type DailyReportRepository struct {
	mu      sync.RWMutex
	reports map[string]domain.DailyReport
}

func NewDailyReportRepository() *DailyReportRepository {
	return &DailyReportRepository{
		reports: make(map[string]domain.DailyReport),
	}
}

func (r *DailyReportRepository) Save(ctx context.Context, report domain.DailyReport) error {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	r.reports[keyForDay(report.Date)] = report
	return nil
}

func (r *DailyReportRepository) GetByDay(ctx context.Context, day time.Time) (domain.DailyReport, bool, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	report, ok := r.reports[keyForDay(day)]
	return report, ok, nil
}

func keyForDay(day time.Time) string {
	return timeutil.CanonicalUTCDate(day).Format("2006-01-02")
}
