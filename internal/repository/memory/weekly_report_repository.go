package memory

import (
	"context"
	"sync"
	"time"

	"sugary/internal/domain"
)

type WeeklyReportRepository struct {
	mu      sync.RWMutex
	reports map[string]domain.WeeklyReport
}

func NewWeeklyReportRepository() *WeeklyReportRepository {
	return &WeeklyReportRepository{
		reports: make(map[string]domain.WeeklyReport),
	}
}

func (r *WeeklyReportRepository) Save(ctx context.Context, report domain.WeeklyReport) error {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	key := keyForDay(report.WeekStartDate)
	if existing, ok := r.reports[key]; ok && !existing.CreatedAt.IsZero() {
		report.CreatedAt = existing.CreatedAt
	}
	r.reports[key] = report
	return nil
}

func (r *WeeklyReportRepository) GetByWeekStart(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	report, ok := r.reports[keyForDay(weekStart)]
	return report, ok, nil
}
