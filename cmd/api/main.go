package main

import (
	"context"
	"log"

	"sugary/internal/config"
	httpdelivery "sugary/internal/delivery/http"
	"sugary/internal/delivery/http/handler"
	"sugary/internal/repository/ai"
	"sugary/internal/repository/postgres"
	"sugary/internal/usecase"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	store, err := postgres.NewStore(ctx, cfg.Postgres)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	mealRepository := postgres.NewMealRepository(store.Queries)
	dailyReportRepository := postgres.NewDailyReportRepository(store.Queries)
	nutritionAnalyzer := ai.NewStubNutritionAnalyzer()

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
		log.Fatal(err)
	}
}
