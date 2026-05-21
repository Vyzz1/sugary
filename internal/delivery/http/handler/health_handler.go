package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	httpresponse "sugary/internal/delivery/http/response"
	"sugary/internal/usecase"
)

type HealthHandler struct {
	healthCheck usecase.HealthCheck
}

func NewHealthHandler() HealthHandler {
	return HealthHandler{
		healthCheck: usecase.NewHealthCheck(),
	}
}

func (h HealthHandler) Check(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, httpresponse.OK(h.healthCheck.Execute()))
}
