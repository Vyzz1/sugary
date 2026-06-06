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

type getInsightUseCase interface {
	Execute(ctx context.Context, rangeType string, loc *time.Location) (domain.InsightResponse, error)
}

type InsightHandler struct {
	getInsight getInsightUseCase
}

func NewInsightHandler(getInsight getInsightUseCase) InsightHandler {
	return InsightHandler{getInsight: getInsight}
}

func (h InsightHandler) Get(ctx *gin.Context) {
	location, _, err := requestLocation(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_timezone", "X-Timezone must be a valid IANA timezone, for example Asia/Ho_Chi_Minh"))
		return
	}

	insight, err := h.getInsight.Execute(ctx.Request.Context(), strings.TrimSpace(ctx.Query("range")), location)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidRange) {
			ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_range", "range must be one of: 7d, 30d, 90d"))
			return
		}

		ctx.JSON(http.StatusInternalServerError, httpresponse.Fail(ctx, "insight_failed", "failed to get insight"))
		return
	}

	ctx.JSON(http.StatusOK, httpresponse.OK(ctx, insight))
}
