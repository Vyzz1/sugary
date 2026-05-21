package domain

import (
	"context"
	"time"
)

type DailyReport struct {
	Date              time.Time `json:"date"`
	MealCount         int       `json:"meal_count"`
	TotalSugarGrams   float64   `json:"total_sugar_grams"`
	AverageSugarGrams float64   `json:"average_sugar_grams"`
	HighestRiskLevel  string    `json:"highest_risk_level"`
	Summary           string    `json:"summary"`
}

type DailyReportRepository interface {
	Save(ctx context.Context, report DailyReport) error
	GetByDay(ctx context.Context, day time.Time) (DailyReport, bool, error)
}
