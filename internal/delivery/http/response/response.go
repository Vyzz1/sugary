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

func OK(data any) gin.H {
	return gin.H{
		"success": true,
		"data":    data,
	}
}

func OKWithMeta(data any, meta any) gin.H {
	return gin.H{
		"success": true,
		"data":    data,
		"meta":    meta,
	}
}

func Fail(code string, message string) gin.H {
	return gin.H{
		"success": false,
		"error": Error{
			Code:    code,
			Message: message,
		},
	}
}
