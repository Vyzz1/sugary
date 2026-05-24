package http

import (
	"github.com/gin-gonic/gin"

	"sugary/internal/config"
	"sugary/internal/delivery/http/handler"
	"sugary/internal/delivery/http/middleware"
)

func NewRouter(
	cfg config.Config,
	healthHandler handler.HealthHandler,
	authHandler handler.AuthHandler,
	uploadHandler handler.UploadHandler,
	mealHandler handler.MealHandler,
	reportHandler handler.ReportHandler,
	jobHandler handler.JobHandler,
) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.CORS(cfg.CORSAllowOrigins))
	router.Use(middleware.RequestLogger())
	router.GET("/health", healthHandler.Check)
	api := router.Group("/api")
	{

		api.POST("/login", authHandler.Login)

	}

	protected := router.Group("/api")
	protected.Use(middleware.JWT(cfg.Auth.JWTSecret))
	protected.POST("/upload", uploadHandler.Upload)
	protected.POST("/meals", mealHandler.Create)
	protected.GET("/meals", mealHandler.ListByDay)
	protected.GET("/meals/recent", mealHandler.ListRecent)
	protected.PATCH("/meals/:id", mealHandler.EditMeal)
	protected.PATCH("/meals/:id/analysis", mealHandler.EditAnalysis)
	protected.DELETE("/meals/:id", mealHandler.DeleteMeal)
	protected.POST("/jobs/daily-report", jobHandler.RunDailyReport)
	protected.GET("/reports/daily", reportHandler.GetDaily)

	return router
}
