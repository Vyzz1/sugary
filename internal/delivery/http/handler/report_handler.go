package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	httpresponse "sugary/internal/delivery/http/response"
	"sugary/internal/domain"
)

type compileDailyReportUseCase interface {
	Execute(ctx context.Context, day time.Time) (domain.DailyReport, error)
}

type compileWeeklyReportUseCase interface {
	Execute(ctx context.Context, day time.Time) (domain.WeeklyReport, error)
}

type getDailyReportUseCase interface {
	Execute(ctx context.Context, day time.Time) (domain.DailyReport, bool, error)
}

type getWeeklyReportUseCase interface {
	Execute(ctx context.Context, weekStart time.Time) (domain.WeeklyReport, bool, error)
}

type ReportHandler struct {
	compileDailyReport  compileDailyReportUseCase
	compileWeeklyReport compileWeeklyReportUseCase
	getDailyReport      getDailyReportUseCase
	getWeeklyReport     getWeeklyReportUseCase
}

func NewReportHandler(
	compileDailyReport compileDailyReportUseCase,
	getDailyReport getDailyReportUseCase,
	compileWeeklyReport compileWeeklyReportUseCase,
	getWeeklyReport getWeeklyReportUseCase,
) ReportHandler {
	return ReportHandler{
		compileDailyReport:  compileDailyReport,
		compileWeeklyReport: compileWeeklyReport,
		getDailyReport:      getDailyReport,
		getWeeklyReport:     getWeeklyReport,
	}
}

func (h ReportHandler) GetDaily(ctx *gin.Context) {
	location, _, err := requestLocation(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_timezone", "X-Timezone must be a valid IANA timezone, for example Asia/Ho_Chi_Minh"))
		return
	}

	dayParam := ctx.Query("date")
	if dayParam == "" {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "missing_date", "date is required in YYYY-MM-DD format"))
		return
	}

	day, err := parseDayInLocation(strings.TrimSpace(dayParam), location)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_date", "date must be YYYY-MM-DD"))
		return
	}

	report, found, err := h.getDailyReport.Execute(ctx.Request.Context(), day)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, domain.ErrInvalidDate) {
			status = http.StatusBadRequest
		}

		ctx.JSON(status, httpresponse.Fail(ctx, "daily_report_failed", err.Error()))
		return
	}

	if !found {
		ctx.JSON(http.StatusNotFound, httpresponse.Fail(ctx, "daily_report_not_found", "daily report not found"))
		return
	}

	ctx.JSON(http.StatusOK, httpresponse.OK(ctx, report))
}

func (h ReportHandler) GetWeekly(ctx *gin.Context) {
	location, _, err := requestLocation(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_timezone", "X-Timezone must be a valid IANA timezone, for example Asia/Ho_Chi_Minh"))
		return
	}

	weekStartParam := ctx.Query("week_start")
	if weekStartParam == "" {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "missing_week_start", "week_start is required in YYYY-MM-DD format"))
		return
	}

	weekStart, err := parseDayInLocation(strings.TrimSpace(weekStartParam), location)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_week_start", "week_start must be YYYY-MM-DD"))
		return
	}
	if !isMonday(weekStart) {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_week_start", "week_start must be a Monday"))
		return
	}

	report, found, err := h.getWeeklyReport.Execute(ctx.Request.Context(), weekStart)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, domain.ErrInvalidDate) {
			status = http.StatusBadRequest
		}

		ctx.JSON(status, httpresponse.Fail(ctx, "weekly_report_failed", err.Error()))
		return
	}

	if !found {
		ctx.JSON(http.StatusNotFound, httpresponse.Fail(ctx, "weekly_report_not_found", "weekly report not found"))
		return
	}

	ctx.JSON(http.StatusOK, httpresponse.OK(ctx, report))
}

func isMonday(day time.Time) bool {
	return day.Weekday() == time.Monday
}
