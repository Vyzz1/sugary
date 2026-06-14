package main

import (
	"context"
	"log"
	"strings"
	"time"

	"go.uber.org/zap"

	"sugary/internal/config"
	httpdelivery "sugary/internal/delivery/http"
	"sugary/internal/delivery/http/handler"
	"sugary/internal/domain"
	"sugary/internal/platform/hub"
	"sugary/internal/platform/logging"
	cronplatform "sugary/internal/platform/scheduler/cron"
	"sugary/internal/repository/ai"
	"sugary/internal/repository/mail"
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
	weeklyReportRepository := postgres.NewWeeklyReportRepository(store.Queries)
	insightRepository := postgres.NewInsightRepository(store.Queries)
	geminiNutritionAnalyzer := ai.NewGeminiNutritionAnalyzer(cfg.GeminiAPIKey, cfg.GeminiModel)
	geminiDailyReportInterpreter := ai.NewGeminiDailyReportInterpreter(cfg.GeminiAPIKey, cfg.GeminiModel)
	geminiWeeklyReportInterpreter := ai.NewGeminiWeeklyReportInterpreter(cfg.GeminiAPIKey, cfg.GeminiModel)
	nutritionAnalyzer := domain.NutritionAnalyzer(geminiNutritionAnalyzer)
	dailyReportInterpreter := domain.DailyReportInterpreter(geminiDailyReportInterpreter)
	weeklyReportInterpreter := domain.WeeklyReportInterpreter(geminiWeeklyReportInterpreter)
	if strings.EqualFold(strings.TrimSpace(cfg.AIProvider), "huggingface") {
		huggingFaceConfig := ai.HuggingFaceConfig{
			APIToken: cfg.HuggingFace.APIToken,
			Model:    cfg.HuggingFace.Model,
			APIURL:   cfg.HuggingFace.APIURL,
		}
		nutritionAnalyzer = ai.NewFallbackNutritionAnalyzer(
			ai.NewHuggingFaceNutritionAnalyzer(huggingFaceConfig),
			geminiNutritionAnalyzer,
		)
		dailyReportInterpreter = ai.NewFallbackDailyReportInterpreter(
			ai.NewHuggingFaceDailyReportInterpreter(huggingFaceConfig),
			geminiDailyReportInterpreter,
		)
		weeklyReportInterpreter = ai.NewFallbackWeeklyReportInterpreter(
			ai.NewHuggingFaceWeeklyReportInterpreter(huggingFaceConfig),
			geminiWeeklyReportInterpreter,
		)
		logger.Info("ai_provider_selected", zap.String("provider", "huggingface"), zap.String("fallback", "gemini"))
	} else {
		logger.Info("ai_provider_selected", zap.String("provider", "gemini"))
	}
	fileUploader := uploadproxy.NewHTTPUploader(cfg.Upload)
	reportEmailSender, err := mail.NewBrevoReportEmailSender(cfg.Brevo)
	if err != nil {
		logger.Error("brevo_report_email_sender_init_failed", zap.Error(err))
		return
	}

	// Hub broadcasts async AI results to all connected WebSocket clients.
	wsHub := hub.New()

	logMeal := usecase.NewLogMeal(mealRepository, nutritionAnalyzer).WithPublisher(wsHub)
	uploadFile := usecase.NewUploadFile(fileUploader)
	listMeals := usecase.NewListMeals(mealRepository)
	listRecentMeals := usecase.NewListRecentMeals(mealRepository)
	editMealAnalysis := usecase.NewEditMealAnalysis(mealRepository)
	editMeal := usecase.NewEditMeal(mealRepository, nutritionAnalyzer).WithPublisher(wsHub)
	deleteMeal := usecase.NewDeleteMeal(mealRepository)
	compileDailyReport := usecase.NewCompileDailyReport(mealRepository, dailyReportRepository, dailyReportInterpreter).
		WithPublisher(wsHub).
		WithEmailSender(reportEmailSender)
	compileWeeklyReport := usecase.NewCompileWeeklyReport(mealRepository, weeklyReportRepository, weeklyReportInterpreter).
		WithPublisher(wsHub).
		WithEmailSender(reportEmailSender)
	getDailyReport := usecase.NewGetDailyReport(dailyReportRepository)
	getWeeklyReport := usecase.NewGetWeeklyReport(weeklyReportRepository)
	getInsight := usecase.NewGetInsight(insightRepository)
	retryFailedMealAnalyses := usecase.NewRetryFailedMealAnalyses(
		mealRepository,
		nutritionAnalyzer,
		cfg.Cron.MealAnalysisRetryMaxAttempts,
		time.Duration(cfg.Cron.MealAnalysisRetryCooldownMinutes)*time.Minute,
		cfg.Cron.MealAnalysisRetryBatchSize,
	).WithPublisher(wsHub)

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
		weeklyReportJob, err := usecase.NewWeeklyReportJob(compileWeeklyReport, cfg.Cron.Timezone)
		if err != nil {
			logger.Error("cron_weekly_report_job_init_failed", zap.Error(err))
			return
		}
		if err := scheduler.Register(usecase.ScheduleSpec{
			Expression: cfg.Cron.WeeklyReportExpression,
			Timezone:   cfg.Cron.Timezone,
		}, weeklyReportJob); err != nil {
			logger.Error("cron_weekly_report_job_register_failed", zap.Error(err))
			return
		}
		if err := scheduler.Register(usecase.ScheduleSpec{
			Expression: cfg.Cron.MealAnalysisRetryExpression,
			Timezone:   cfg.Cron.Timezone,
		}, usecase.NewRetryFailedMealAnalysesJob(retryFailedMealAnalyses)); err != nil {
			logger.Error("cron_meal_analysis_retry_register_failed", zap.Error(err))
			return
		}
		if err := scheduler.Start(); err != nil {
			logger.Error("cron_scheduler_start_failed", zap.Error(err))
			return
		}
		defer scheduler.Stop(context.Background())
		logger.Info("cron_scheduler_started",
			zap.String("daily_report_expression", cfg.Cron.DailyReportExpression),
			zap.String("weekly_report_expression", cfg.Cron.WeeklyReportExpression),
			zap.String("timezone", cfg.Cron.Timezone),
			zap.String("meal_analysis_retry_expression", cfg.Cron.MealAnalysisRetryExpression),
		)
	}

	reportHandler := handler.NewReportHandler(compileDailyReport, getDailyReport, compileWeeklyReport, getWeeklyReport)
	insightHandler := handler.NewInsightHandler(getInsight)
	authHandler := handler.NewAuthHandler(cfg.Auth)
	uploadHandler := handler.NewUploadHandler(uploadFile)
	mealHandler := handler.NewMealHandler(logMeal, listMeals, listRecentMeals, editMealAnalysis, editMeal, deleteMeal)
	wsHandler := handler.NewWSHandler(wsHub, cfg.Auth.JWTSecret)

	router := httpdelivery.NewRouter(
		cfg,
		handler.NewHealthHandler(),
		authHandler,
		uploadHandler,
		mealHandler,
		reportHandler,
		insightHandler,
		handler.NewJobHandler(reportHandler),
		wsHandler,
	)

	if err := router.Run(":" + cfg.Port); err != nil {
		logger.Error("http_server_failed", zap.Error(err), zap.String("port", cfg.Port))
	}
}
