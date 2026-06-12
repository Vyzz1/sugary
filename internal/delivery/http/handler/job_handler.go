package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	httpresponse "sugary/internal/delivery/http/response"
)

type JobHandler struct {
	reportHandler ReportHandler
}

func NewJobHandler(reportHandler ReportHandler) JobHandler {
	return JobHandler{
		reportHandler: reportHandler,
	}
}

func (h JobHandler) RunDailyReport(ctx *gin.Context) {
	location, _, err := requestLocation(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_timezone", "X-Timezone must be a valid IANA timezone, for example Asia/Ho_Chi_Minh"))
		return
	}

	dayParam := ctx.Query("date")
	day := time.Now().In(location)
	if dayParam != "" {
		parsed, err := parseDayInLocation(strings.TrimSpace(dayParam), location)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_date", "date must be YYYY-MM-DD"))
			return
		}
		day = parsed
	}

	report, err := h.reportHandler.compileDailyReport.Execute(ctx.Request.Context(), day)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, httpresponse.Fail(ctx, "daily_report_failed", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, httpresponse.OK(ctx, report))
}

func (h JobHandler) RunWeeklyReport(ctx *gin.Context) {
	location, _, err := requestLocation(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_timezone", "X-Timezone must be a valid IANA timezone, for example Asia/Ho_Chi_Minh"))
		return
	}

	weekStartParam := ctx.Query("week_start")
	weekStart := time.Now().In(location)
	if weekStartParam != "" {
		parsed, err := parseDayInLocation(strings.TrimSpace(weekStartParam), location)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_week_start", "week_start must be YYYY-MM-DD"))
			return
		}
		if !isMonday(parsed) {
			ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_week_start", "week_start must be a Monday"))
			return
		}
		weekStart = parsed
	}

	report, err := h.reportHandler.compileWeeklyReport.Execute(ctx.Request.Context(), weekStart)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, httpresponse.Fail(ctx, "weekly_report_failed", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, httpresponse.OK(ctx, report))
}
