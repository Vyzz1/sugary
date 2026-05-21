package http

import (
	"github.com/gin-gonic/gin"

	"sugary/internal/config"
	"sugary/internal/delivery/http/handler"
	"sugary/internal/delivery/http/middleware"
)

func NewRouter(
	auth config.AuthConfig,
	healthHandler handler.HealthHandler,
	authHandler handler.AuthHandler,
	mealHandler handler.MealHandler,
	reportHandler handler.ReportHandler,
	jobHandler handler.JobHandler,
) *gin.Engine {
	router := gin.Default()
	router.GET("/health", healthHandler.Check)
	api := router.Group("/api")
	{

		api.POST("/login", authHandler.Login)

	}

	protected := router.Group("/api")
	protected.Use(middleware.JWT(auth.JWTSecret))
	protected.POST("/meals", mealHandler.Create)
	protected.POST("/jobs/daily-report", jobHandler.RunDailyReport)
	protected.GET("/reports/daily", reportHandler.GetDaily)

	return router
}
