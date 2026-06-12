package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"sugary/internal/domain"
)

type stubCompileDailyReportUseCase struct{}

func (stubCompileDailyReportUseCase) Execute(ctx context.Context, day time.Time) (domain.DailyReport, error) {
	return domain.DailyReport{}, nil
}

type stubGetDailyReportUseCase struct{}

func (stubGetDailyReportUseCase) Execute(ctx context.Context, day time.Time) (domain.DailyReport, bool, error) {
	return domain.DailyReport{}, false, nil
}

type stubCompileWeeklyReportUseCase struct{}

func (stubCompileWeeklyReportUseCase) Execute(ctx context.Context, day time.Time) (domain.WeeklyReport, error) {
	return domain.WeeklyReport{}, nil
}

type stubGetWeeklyReportUseCase struct {
	executeFn func(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error)
}

func (s stubGetWeeklyReportUseCase) Execute(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error) {
	return s.executeFn(ctx, weekStart)
}

func TestReportHandlerGetWeeklyRequiresMondayWeekStart(t *testing.T) {
	t.Parallel()

	ctx, recorder := newReportTestContext(http.MethodGet, "/reports/weekly?week_start=2026-06-09")
	handler := NewReportHandler(
		stubCompileDailyReportUseCase{},
		stubGetDailyReportUseCase{},
		stubCompileWeeklyReportUseCase{},
		stubGetWeeklyReportUseCase{
			executeFn: func(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error) {
				t.Fatal("expected get weekly use case not to be called")
				return domain.WeeklyReport{}, false, nil
			},
		},
	)

	handler.GetWeekly(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestReportHandlerGetWeeklyReturnsReport(t *testing.T) {
	t.Parallel()

	ctx, recorder := newReportTestContext(http.MethodGet, "/reports/weekly?week_start=2026-06-08")
	location, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatalf("expected valid location, got %v", err)
	}
	expectedWeekStart := time.Date(2026, 6, 8, 0, 0, 0, 0, location)
	handler := NewReportHandler(
		stubCompileDailyReportUseCase{},
		stubGetDailyReportUseCase{},
		stubCompileWeeklyReportUseCase{},
		stubGetWeeklyReportUseCase{
			executeFn: func(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error) {
				if !weekStart.Equal(expectedWeekStart) {
					t.Fatalf("expected week start %s, got %s", expectedWeekStart, weekStart)
				}
				return domain.WeeklyReport{
					WeekStartDate: expectedWeekStart,
					WeekEndDate:   time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC),
					CreatedAt:     time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC),
				}, true, nil
			},
		},
	)

	handler.GetWeekly(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func newReportTestContext(method string, target string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, nil)
	return ctx, recorder
}
