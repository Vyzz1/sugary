package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const maxLoggedBodyBytes = 4096

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseBodyWriter) Write(data []byte) (int, error) {
	if w.body.Len() < maxLoggedBodyBytes {
		remaining := maxLoggedBodyBytes - w.body.Len()
		if len(data) > remaining {
			w.body.Write(data[:remaining])
		} else {
			w.body.Write(data)
		}
	}
	return w.ResponseWriter.Write(data)
}

func RequestLogger() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		requestID := ctx.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("%d", start.UnixNano())
		}
		ctx.Writer.Header().Set("X-Request-ID", requestID)
		ctx.Set("request_id", requestID)

		reqBody := extractRequestBody(ctx.Request)
		writer := &responseBodyWriter{ResponseWriter: ctx.Writer, body: bytes.NewBuffer(nil)}
		ctx.Writer = writer

		ctx.Next()

		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", ctx.Request.Method),
			zap.String("path", ctx.FullPath()),
			zap.Int("status", ctx.Writer.Status()),
			zap.Int64("latency_ms", time.Since(start).Milliseconds()),
			zap.String("client_ip", ctx.ClientIP()),
		}
		if reqBody != "" {
			fields = append(fields, zap.String("request_body", reqBody))
		}
		respBody := extractResponseBody(ctx.Writer.Header(), writer.body.String())
		if respBody != "" {
			fields = append(fields, zap.String("response_body", respBody))
		}

		zap.L().Info("http_request", fields...)
	}
}

func extractRequestBody(req *http.Request) string {
	if req == nil || req.Body == nil {
		return ""
	}
	if !strings.Contains(strings.ToLower(req.Header.Get("Content-Type")), "application/json") {
		return ""
	}

	raw, err := io.ReadAll(req.Body)
	if err != nil {
		return ""
	}
	req.Body = io.NopCloser(bytes.NewBuffer(raw))
	if len(raw) == 0 {
		return ""
	}
	return sanitizeAndLimitJSON(raw)
}

func extractResponseBody(headers http.Header, body string) string {
	if !strings.Contains(strings.ToLower(headers.Get("Content-Type")), "application/json") {
		return ""
	}
	if body == "" {
		return ""
	}
	return sanitizeAndLimitJSON([]byte(body))
}

func sanitizeAndLimitJSON(raw []byte) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return ""
	}

	var obj any
	if err := json.Unmarshal(trimmed, &obj); err == nil {
		redactJSON(obj)
		if encoded, err := json.Marshal(obj); err == nil {
			trimmed = encoded
		}
	}

	if len(trimmed) > maxLoggedBodyBytes {
		return string(trimmed[:maxLoggedBodyBytes]) + "...(truncated)"
	}
	return string(trimmed)
}

func redactJSON(value any) {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "password") {
				v[key] = "***"
				continue
			}
			redactJSON(item)
		}
	case []any:
		for _, item := range v {
			redactJSON(item)
		}
	}
}
