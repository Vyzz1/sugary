package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"

	"sugary/internal/platform/timeutil"
)

const defaultPort = "8080"

type Config struct {
	AppEnv           string
	Port             string
	LogLevel         string
	CORSAllowOrigins string
	AIProvider       string
	GeminiAPIKey     string
	GeminiModel      string
	HuggingFace      HuggingFaceConfig
	Auth             AuthConfig
	Upload           UploadConfig
	Postgres         PostgresConfig
	Redis            RedisConfig
	Cron             CronConfig
}

type AuthConfig struct {
	LoginUser     string
	LoginPassword string
	JWTSecret     string
	JWTExpiresIn  string
}

type UploadConfig struct {
	APIURL        string
	InternalToken string
	Folder        string
}

type HuggingFaceConfig struct {
	APIToken string
	Model    string
	APIURL   string
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       string
}

type CronConfig struct {
	Enabled                          bool
	DailyReportExpression            string
	Timezone                         string
	MealAnalysisRetryExpression      string
	MealAnalysisRetryMaxAttempts     int32
	MealAnalysisRetryCooldownMinutes int
	MealAnalysisRetryBatchSize       int32
}

func Load() Config {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("config: unable to load .env: %v", err)
	}

	return Config{
		AppEnv:           getEnv("APP_ENV", "development"),
		Port:             getEnv("PORT", getEnv("APP_PORT", defaultPort)),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		CORSAllowOrigins: getEnv("CORS_ALLOW_ORIGINS", "*"),
		AIProvider:       getEnv("AI_PROVIDER", "gemini"),
		GeminiAPIKey:     os.Getenv("GEMINI_API_KEY"),
		GeminiModel:      getEnv("GEMINI_MODEL", "gemini-2.5-flash"),
		HuggingFace: HuggingFaceConfig{
			APIToken: os.Getenv("HUGGINGFACE_API_TOKEN"),
			Model:    getEnv("HUGGINGFACE_MODEL", "Qwen/Qwen2.5-7B-Instruct"),
			APIURL:   getEnv("HUGGINGFACE_API_URL", "https://router.huggingface.co/v1/chat/completions"),
		},
		Auth: AuthConfig{
			LoginUser:     os.Getenv("LOGIN_USER"),
			LoginPassword: os.Getenv("LOGIN_PASSWORD"),
			JWTSecret:     os.Getenv("JWT_SECRET"),
			JWTExpiresIn:  getEnv("JWT_EXPIRES_IN", "24h"),
		},
		Upload: UploadConfig{
			APIURL:        os.Getenv("UPLOAD_API_URL"),
			InternalToken: os.Getenv("UPLOAD_INTERNAL_TOKEN"),
			Folder:        getEnv("UPLOAD_FOLDER", "sugary"),
		},
		Postgres: PostgresConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "postgres"),
			Password: getEnv("POSTGRES_PASSWORD", "postgres"),
			Database: getEnv("POSTGRES_DB", "sugary"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       getEnv("REDIS_DB", "0"),
		},
		Cron: CronConfig{
			Enabled:                          getEnvBool("CRON_ENABLED", false),
			DailyReportExpression:            getEnv("CRON_DAILY_REPORT_EXPRESSION", "5 0 * * *"),
			Timezone:                         getEnv("CRON_TIMEZONE", timeutil.DefaultTimezone),
			MealAnalysisRetryExpression:      getEnv("CRON_MEAL_ANALYSIS_RETRY_EXPRESSION", "*/15 * * * *"),
			MealAnalysisRetryMaxAttempts:     int32(getEnvInt("CRON_MEAL_ANALYSIS_RETRY_MAX_ATTEMPTS", 5)),
			MealAnalysisRetryCooldownMinutes: getEnvInt("CRON_MEAL_ANALYSIS_RETRY_COOLDOWN_MINUTES", 15),
			MealAnalysisRetryBatchSize:       int32(getEnvInt("CRON_MEAL_ANALYSIS_RETRY_BATCH_SIZE", 25)),
		},
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func (c PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.SSLMode,
	)
}
