package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	httpresponse "sugary/internal/delivery/http/response"
	"sugary/internal/domain"
)

type compileDailyReportUseCase interface {
	Execute(ctx context.Context, day time.Time) (domain.DailyReport, error)
}

type getDailyReportUseCase interface {
	Execute(ctx context.Context, day time.Time) (domain.DailyReport, bool, error)
}

type ReportHandler struct {
	compileDailyReport compileDailyReportUseCase
	getDailyReport     getDailyReportUseCase
}

func NewReportHandler(
	compileDailyReport compileDailyReportUseCase,
	getDailyReport getDailyReportUseCase,
) ReportHandler {
	return ReportHandler{
		compileDailyReport: compileDailyReport,
		getDailyReport:     getDailyReport,
	}
}

func (h ReportHandler) GetDaily(ctx *gin.Context) {
	dayParam := ctx.Query("date")
	if dayParam == "" {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "missing_date", "date is required in YYYY-MM-DD format"))
		return
	}

	day, err := time.Parse("2006-01-02", dayParam)
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
