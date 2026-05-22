package usecase

import (
	"context"
	"fmt"
	"time"

	"sugary/internal/domain"
)

type CompileDailyReport struct {
	mealRepository        domain.MealRepository
	dailyReportRepository domain.DailyReportRepository
}

func NewCompileDailyReport(
	mealRepository domain.MealRepository,
	dailyReportRepository domain.DailyReportRepository,
) CompileDailyReport {
	return CompileDailyReport{
		mealRepository:        mealRepository,
		dailyReportRepository: dailyReportRepository,
	}
}

func (uc CompileDailyReport) Execute(ctx context.Context, day time.Time) (domain.DailyReport, error) {
	if day.IsZero() {
		return domain.DailyReport{}, domain.ErrInvalidDate
	}

	day = startOfDayUTC(day)

	meals, err := uc.mealRepository.ListByDay(ctx, day)
	if err != nil {
		return domain.DailyReport{}, err
	}

	report := domain.DailyReport{
		Date: day,
	}

	if len(meals) == 0 {
		report.HighestRiskLevel = "unknown"
		report.Summary = "No meals were recorded for the selected day."
		if err := uc.dailyReportRepository.Save(ctx, report); err != nil {
			return domain.DailyReport{}, err
		}
		return report, nil
	}

	var totalSugar float64
	highestRisk := "unknown"
	analyzedMealCount := 0

	for _, meal := range meals {
		report.MealCount++
		if meal.Analysis == nil {
			continue
		}

		analyzedMealCount++
		totalSugar += meal.Analysis.EstimatedSugarGrams
		if highestRisk == "unknown" || compareRisk(meal.Analysis.RiskLevel, highestRisk) > 0 {
			highestRisk = meal.Analysis.RiskLevel
		}
	}

	report.TotalSugarGrams = totalSugar
	report.HighestRiskLevel = highestRisk
	if analyzedMealCount == 0 {
		report.Summary = fmt.Sprintf(
			"%d meals logged, but nutrition analysis is not available yet.",
			report.MealCount,
		)
	} else {
		report.AverageSugarGrams = totalSugar / float64(analyzedMealCount)
		report.Summary = fmt.Sprintf(
			"%d meals logged (%d analyzed). Estimated sugar intake %.1fg. Highest meal risk: %s.",
			report.MealCount,
			analyzedMealCount,
			report.TotalSugarGrams,
			report.HighestRiskLevel,
		)
	}

	if err := uc.dailyReportRepository.Save(ctx, report); err != nil {
		return domain.DailyReport{}, err
	}

	return report, nil
}

func startOfDayUTC(day time.Time) time.Time {
	day = day.UTC()
	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
}

func compareRisk(left string, right string) int {
	return riskScore(left) - riskScore(right)
}

func riskScore(level string) int {
	switch level {
	case "high":
		return 3
	case "medium":
		return 2
	default:
		return 1
	}
}
