package handler

import (
	"net/http"
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
	dayParam := ctx.Query("date")
	day := time.Now().UTC()
	if dayParam != "" {
		parsed, err := time.Parse("2006-01-02", dayParam)
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
