package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

const defaultPort = "8080"

type Config struct {
	AppEnv           string
	Port             string
	LogLevel         string
	CORSAllowOrigins string
	GeminiAPIKey     string
	GeminiModel      string
	Auth             AuthConfig
	Upload           UploadConfig
	Postgres         PostgresConfig
	Redis            RedisConfig
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

func Load() Config {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("config: unable to load .env: %v", err)
	}

	return Config{
		AppEnv:           getEnv("APP_ENV", "development"),
		Port:             getEnv("PORT", getEnv("APP_PORT", defaultPort)),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		CORSAllowOrigins: getEnv("CORS_ALLOW_ORIGINS", "*"),
		GeminiAPIKey:     os.Getenv("GEMINI_API_KEY"),
		GeminiModel:      getEnv("GEMINI_MODEL", "gemini-2.5-flash"),
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
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
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
