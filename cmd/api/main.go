package main

import (
	"context"
	"log"

	"go.uber.org/zap"

	"sugary/internal/config"
	httpdelivery "sugary/internal/delivery/http"
	"sugary/internal/delivery/http/handler"
	"sugary/internal/platform/logging"
	"sugary/internal/repository/ai"
	"sugary/internal/repository/postgres"
	"sugary/internal/usecase"
)

func main() {
	cfg := config.Load()
	logger, err := logging.New(cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logger.Sync()
	zap.ReplaceGlobals(logger)
	ctx := context.Background()

	store, err := postgres.NewStore(ctx, cfg.Postgres)
	if err != nil {
		logger.Error("postgres_connect_failed", zap.Error(err))
		return
	}
	defer store.Close()

	mealRepository := postgres.NewMealRepository(store.Queries)
	dailyReportRepository := postgres.NewDailyReportRepository(store.Queries)
	nutritionAnalyzer := ai.NewGeminiNutritionAnalyzer(cfg.GeminiAPIKey, cfg.GeminiModel)

	logMeal := usecase.NewLogMeal(mealRepository, nutritionAnalyzer)
	compileDailyReport := usecase.NewCompileDailyReport(mealRepository, dailyReportRepository)
	getDailyReport := usecase.NewGetDailyReport(dailyReportRepository)

	reportHandler := handler.NewReportHandler(compileDailyReport, getDailyReport)
	authHandler := handler.NewAuthHandler(cfg.Auth)
	mealHandler := handler.NewMealHandler(logMeal)

	router := httpdelivery.NewRouter(
		cfg.Auth,
		handler.NewHealthHandler(),
		authHandler,
		mealHandler,
		reportHandler,
		handler.NewJobHandler(reportHandler),
	)

	if err := router.Run(":" + cfg.Port); err != nil {
		logger.Error("http_server_failed", zap.Error(err), zap.String("port", cfg.Port))
	}
}
