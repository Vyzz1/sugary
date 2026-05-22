package response

import "github.com/gin-gonic/gin"

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Envelope struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   *Error `json:"error,omitempty"`
	Meta    any    `json:"meta,omitempty"`
}

func OK(ctx *gin.Context, data any) gin.H {
	res := gin.H{
		"success": true,
		"data":    data,
	}
	withRequestID(ctx, res)
	return res
}

func OKWithMeta(ctx *gin.Context, data any, meta any) gin.H {
	res := gin.H{
		"success": true,
		"data":    data,
		"meta":    meta,
	}
	withRequestID(ctx, res)
	return res
}

func Fail(ctx *gin.Context, code string, message string) gin.H {
	res := gin.H{
		"success": false,
		"error": Error{
			Code:    code,
			Message: message,
		},
	}
	withRequestID(ctx, res)
	return res
}

func withRequestID(ctx *gin.Context, res gin.H) {
	if ctx == nil {
		return
	}
	if requestID, ok := ctx.Get("request_id"); ok {
		if value, ok := requestID.(string); ok && value != "" {
			res["request_id"] = value
		}
	}
}
