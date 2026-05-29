package main

import (
	"context"
	"log"

	"go.uber.org/zap"

	"sugary/internal/config"
	httpdelivery "sugary/internal/delivery/http"
	"sugary/internal/delivery/http/handler"
	"sugary/internal/platform/hub"
	"sugary/internal/platform/logging"
	cronplatform "sugary/internal/platform/scheduler/cron"
	"sugary/internal/repository/ai"
	"sugary/internal/repository/postgres"
	"sugary/internal/repository/uploadproxy"
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
	dailyReportInterpreter := ai.NewGeminiDailyReportInterpreter(cfg.GeminiAPIKey, cfg.GeminiModel)
	fileUploader := uploadproxy.NewHTTPUploader(cfg.Upload)

	// Hub broadcasts async AI results to all connected WebSocket clients.
	wsHub := hub.New()

	logMeal := usecase.NewLogMeal(mealRepository, nutritionAnalyzer).WithPublisher(wsHub)
	uploadFile := usecase.NewUploadFile(fileUploader)
	listMealsByDay := usecase.NewListMealsByDay(mealRepository)
	listRecentMeals := usecase.NewListRecentMeals(mealRepository)
	editMealAnalysis := usecase.NewEditMealAnalysis(mealRepository)
	editMeal := usecase.NewEditMeal(mealRepository, nutritionAnalyzer).WithPublisher(wsHub)
	deleteMeal := usecase.NewDeleteMeal(mealRepository)
	compileDailyReport := usecase.NewCompileDailyReport(mealRepository, dailyReportRepository, dailyReportInterpreter).WithPublisher(wsHub)
	getDailyReport := usecase.NewGetDailyReport(dailyReportRepository)

	if cfg.Cron.Enabled {
		scheduler := cronplatform.New()
		dailyReportJob, err := usecase.NewDailyReportJob(compileDailyReport, cfg.Cron.Timezone)
		if err != nil {
			logger.Error("cron_daily_report_job_init_failed", zap.Error(err))
			return
		}
		if err := scheduler.Register(usecase.ScheduleSpec{
			Expression: cfg.Cron.DailyReportExpression,
			Timezone:   cfg.Cron.Timezone,
		}, dailyReportJob); err != nil {
			logger.Error("cron_job_register_failed", zap.Error(err))
			return
		}
		if err := scheduler.Start(); err != nil {
			logger.Error("cron_scheduler_start_failed", zap.Error(err))
			return
		}
		defer scheduler.Stop(context.Background())
		logger.Info("cron_scheduler_started",
			zap.String("daily_report_expression", cfg.Cron.DailyReportExpression),
			zap.String("timezone", cfg.Cron.Timezone),
		)
	}

	reportHandler := handler.NewReportHandler(compileDailyReport, getDailyReport)
	authHandler := handler.NewAuthHandler(cfg.Auth)
	uploadHandler := handler.NewUploadHandler(uploadFile)
	mealHandler := handler.NewMealHandler(logMeal, listMealsByDay, listRecentMeals, editMealAnalysis, editMeal, deleteMeal)
	wsHandler := handler.NewWSHandler(wsHub, cfg.Auth.JWTSecret)

	router := httpdelivery.NewRouter(
		cfg,
		handler.NewHealthHandler(),
		authHandler,
		uploadHandler,
		mealHandler,
		reportHandler,
		handler.NewJobHandler(reportHandler),
		wsHandler,
	)

	if err := router.Run(":" + cfg.Port); err != nil {
		logger.Error("http_server_failed", zap.Error(err), zap.String("port", cfg.Port))
	}
}
