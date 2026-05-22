package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func RequestLogger() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		requestID := ctx.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("%d", start.UnixNano())
		}
		ctx.Writer.Header().Set("X-Request-ID", requestID)
		ctx.Set("request_id", requestID)

		ctx.Next()

		zap.L().Info("http_request",
			zap.String("request_id", requestID),
			zap.String("method", ctx.Request.Method),
			zap.String("path", ctx.FullPath()),
			zap.Int("status", ctx.Writer.Status()),
			zap.Int64("latency_ms", time.Since(start).Milliseconds()),
			zap.String("client_ip", ctx.ClientIP()),
		)
	}
}
