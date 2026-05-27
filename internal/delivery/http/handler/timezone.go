package handler

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"sugary/internal/platform/timeutil"
)

const timezoneHeader = "X-Timezone"

func requestLocation(ctx *gin.Context) (*time.Location, string, error) {
	name := strings.TrimSpace(ctx.GetHeader(timezoneHeader))
	loc, err := timeutil.ResolveLocation(name)
	if err != nil {
		return nil, "", err
	}

	return loc, loc.String(), nil
}

func parseDayInLocation(value string, loc *time.Location) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", strings.TrimSpace(value), loc)
}
