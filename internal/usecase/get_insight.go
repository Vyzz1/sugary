package usecase

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	"sugary/internal/domain"
	"sugary/internal/platform/timeutil"
)

const dailySugarTargetGrams = 25

type GetInsight struct {
	repository domain.InsightRepository
	now        func() time.Time
}

type TrendMetric struct {
	Direction string
	Percent   float64
}

func NewGetInsight(repository domain.InsightRepository) GetInsight {
	return GetInsight{
		repository: repository,
		now:        time.Now,
	}
}

func (uc GetInsight) WithClock(now func() time.Time) GetInsight {
	uc.now = now
	return uc
}

func (uc GetInsight) Execute(ctx context.Context, rangeType string, loc *time.Location) (domain.InsightResponse, error) {
	if uc.now == nil {
		uc.now = time.Now
	}
	if loc == nil {
		loc = time.Local
	}

	period, err := ResolveInsightPeriod(rangeType, uc.now().In(loc))
	if err != nil {
		return domain.InsightResponse{}, err
	}

	filter := domain.InsightPeriodFilter{
		FromInclusive: period.FromInclusive,
		ToExclusive:   period.ToExclusive,
		Timezone:      period.Timezone,
	}
	previousFilter := domain.InsightPeriodFilter{
		FromInclusive: period.PrevInclusive,
		ToExclusive:   period.PrevExclusive,
		Timezone:      period.Timezone,
	}

	currentStats, err := uc.repository.GetPeriodStats(ctx, filter)
	if err != nil {
		return domain.InsightResponse{}, err
	}
	previousStats, err := uc.repository.GetPeriodStats(ctx, previousFilter)
	if err != nil {
		return domain.InsightResponse{}, err
	}
	dailySugar, err := uc.repository.GetDailySugar(ctx, filter)
	if err != nil {
		return domain.InsightResponse{}, err
	}
	mealTypeBreakdown, err := uc.repository.GetMealTypeBreakdown(ctx, filter)
	if err != nil {
		return domain.InsightResponse{}, err
	}
	riskDistribution, err := uc.repository.GetRiskDistribution(ctx, filter)
	if err != nil {
		return domain.InsightResponse{}, err
	}
	weeklySugar, err := uc.repository.GetWeeklySugar(ctx, filter)
	if err != nil {
		return domain.InsightResponse{}, err
	}
	sugarVsCalories, err := uc.repository.GetSugarVsCalories(ctx, filter)
	if err != nil {
		return domain.InsightResponse{}, err
	}
	topSugarMeals, err := uc.repository.GetTopSugarMeals(ctx, filter)
	if err != nil {
		return domain.InsightResponse{}, err
	}
	topSugarDishes, err := uc.repository.GetTopSugarDishes(ctx, filter)
	if err != nil {
		return domain.InsightResponse{}, err
	}

	dailySugar = FillDailySugar(period, dailySugar)
	riskDistribution = normalizeRiskDistribution(riskDistribution, currentStats.MealCount)
	summary := buildInsightSummary(period, currentStats, dailySugar)
	trend := buildInsightTrend(period, currentStats, previousStats)
	patterns := buildInsightPatterns(topSugarMeals, topSugarDishes, mealTypeBreakdown)

	return domain.InsightResponse{
		Range: domain.InsightRange{
			From:      formatDate(period.From),
			To:        formatDate(period.To),
			Days:      period.Days,
			RangeType: period.RangeType,
		},
		Summary: summary,
		Trend:   trend,
		Charts: domain.InsightCharts{
			DailySugar:        dailySugar,
			MealTypeBreakdown: mealTypeBreakdown,
			RiskDistribution:  riskDistribution,
			WeeklySugar:       weeklySugar,
			SugarVsCalories:   sugarVsCalories,
		},
		Patterns: patterns,
	}, nil
}

func ResolveInsightPeriod(rangeType string, now time.Time) (domain.InsightPeriod, error) {
	rangeType = strings.TrimSpace(strings.ToLower(rangeType))
	if rangeType == "" {
		rangeType = domain.InsightRange30D
	}

	days := 0
	switch rangeType {
	case domain.InsightRange7D:
		days = 7
	case domain.InsightRange30D:
		days = 30
	case domain.InsightRange90D:
		days = 90
	default:
		return domain.InsightPeriod{}, domain.ErrInvalidRange
	}

	today := timeutil.StartOfDay(now)
	from := today.AddDate(0, 0, -(days - 1))
	to := today
	previousFrom := from.AddDate(0, 0, -days)
	previousTo := from.AddDate(0, 0, -1)

	return domain.InsightPeriod{
		RangeType:     rangeType,
		Days:          days,
		From:          from,
		To:            to,
		PreviousFrom:  previousFrom,
		PreviousTo:    previousTo,
		FromInclusive: from.UTC(),
		ToExclusive:   to.AddDate(0, 0, 1).UTC(),
		PrevInclusive: previousFrom.UTC(),
		PrevExclusive: previousTo.AddDate(0, 0, 1).UTC(),
		Timezone:      now.Location().String(),
	}, nil
}

func CalculateTrend(current float64, previous float64) TrendMetric {
	var percent float64
	switch {
	case previous == 0 && current == 0:
		percent = 0
	case previous == 0:
		percent = 100
	default:
		percent = ((current - previous) / previous) * 100
	}
	percent = round1(percent)

	direction := "stable"
	if previous == 0 && current > 0 {
		direction = "new"
	} else if percent >= 10 {
		direction = "up"
	} else if percent <= -10 {
		direction = "down"
	}

	return TrendMetric{Direction: direction, Percent: percent}
}

func FillDailySugar(period domain.InsightPeriod, points []domain.DailySugarPoint) []domain.DailySugarPoint {
	byDate := make(map[string]domain.DailySugarPoint, len(points))
	for _, point := range points {
		point.Date = strings.TrimSpace(point.Date)
		if point.RiskLevel == "" {
			point.RiskLevel = "none"
		}
		point.TargetGrams = dailySugarTargetGrams
		byDate[point.Date] = point
	}

	filled := make([]domain.DailySugarPoint, 0, period.Days)
	for day := period.From; !day.After(period.To); day = day.AddDate(0, 0, 1) {
		key := formatDate(day)
		point, ok := byDate[key]
		if !ok {
			point = domain.DailySugarPoint{
				Date:              key,
				TotalSugarGrams:   0,
				TotalCalories:     0,
				TotalCarbsGrams:   0,
				TotalProteinGrams: 0,
				MealCount:         0,
				RiskLevel:         "none",
				TargetGrams:       dailySugarTargetGrams,
			}
		}
		filled = append(filled, point)
	}
	return filled
}

func buildInsightSummary(period domain.InsightPeriod, stats domain.InsightPeriodStats, daily []domain.DailySugarPoint) domain.InsightSummary {
	var worst *domain.InsightDaySummary
	var best *domain.InsightDaySummary
	var highRiskDays int64

	for _, point := range daily {
		if point.RiskLevel == "high" {
			highRiskDays++
		}
		if point.MealCount == 0 {
			continue
		}
		candidate := domain.InsightDaySummary{
			Date:            point.Date,
			TotalSugarGrams: point.TotalSugarGrams,
			MealCount:       point.MealCount,
			RiskLevel:       point.RiskLevel,
		}
		if worst == nil || candidate.TotalSugarGrams > worst.TotalSugarGrams {
			worst = &candidate
		}
		if best == nil || candidate.TotalSugarGrams < best.TotalSugarGrams {
			best = &candidate
		}
	}

	averagePerMeal := 0.0
	if stats.MealCount > 0 {
		averagePerMeal = stats.TotalSugarGrams / float64(stats.MealCount)
	}

	return domain.InsightSummary{
		TotalSugarGrams:     round1(stats.TotalSugarGrams),
		AverageSugarPerDay:  round1(stats.TotalSugarGrams / float64(period.Days)),
		AverageSugarPerMeal: round1(averagePerMeal),
		TotalMeals:          stats.MealCount,
		HighRiskMeals:       stats.HighRiskMeals,
		HighRiskDays:        highRiskDays,
		WorstDay:            worst,
		BestDay:             best,
	}
}

func buildInsightTrend(period domain.InsightPeriod, current domain.InsightPeriodStats, previous domain.InsightPeriodStats) domain.InsightTrend {
	currentAverageDailySugar := current.TotalSugarGrams / float64(period.Days)
	previousAverageDailySugar := previous.TotalSugarGrams / float64(period.Days)
	sugarTrend := CalculateTrend(currentAverageDailySugar, previousAverageDailySugar)
	highRiskTrend := CalculateTrend(float64(current.HighRiskMeals), float64(previous.HighRiskMeals))
	mealCountTrend := CalculateTrend(float64(current.MealCount), float64(previous.MealCount))

	return domain.InsightTrend{
		ComparisonLabel: "vs previous " + strconv.Itoa(period.Days) + " days",
		CurrentPeriod: domain.InsightPeriodRange{
			From: formatDate(period.From),
			To:   formatDate(period.To),
		},
		PreviousPeriod: domain.InsightPeriodRange{
			From: formatDate(period.PreviousFrom),
			To:   formatDate(period.PreviousTo),
		},
		Sugar: domain.TrendAverageMetric{
			CurrentAverageDailyGrams:  round1(currentAverageDailySugar),
			PreviousAverageDailyGrams: round1(previousAverageDailySugar),
			Direction:                 sugarTrend.Direction,
			Percent:                   sugarTrend.Percent,
		},
		HighRiskMeals: domain.TrendCountMetric{
			CurrentCount:  current.HighRiskMeals,
			PreviousCount: previous.HighRiskMeals,
			Direction:     highRiskTrend.Direction,
			Percent:       highRiskTrend.Percent,
		},
		MealCount: domain.TrendCountMetric{
			CurrentCount:  current.MealCount,
			PreviousCount: previous.MealCount,
			Direction:     mealCountTrend.Direction,
			Percent:       mealCountTrend.Percent,
		},
	}
}

func normalizeRiskDistribution(points []domain.RiskDistributionPoint, totalCount int64) []domain.RiskDistributionPoint {
	if points == nil {
		points = []domain.RiskDistributionPoint{}
	}
	for i := range points {
		if totalCount > 0 {
			points[i].Percentage = round1(float64(points[i].Count) / float64(totalCount) * 100)
		}
	}
	return points
}

func buildInsightPatterns(topMeals []domain.TopSugarMeal, topDishes []domain.TopSugarDish, mealTypes []domain.MealTypeBreakdown) domain.InsightPatterns {
	if topMeals == nil {
		topMeals = []domain.TopSugarMeal{}
	}
	if topDishes == nil {
		topDishes = []domain.TopSugarDish{}
	}

	var worstMealType *domain.WorstMealTypePoint
	if len(mealTypes) > 0 {
		worstMealType = &domain.WorstMealTypePoint{
			MealType:          mealTypes[0].MealType,
			TotalSugarGrams:   mealTypes[0].TotalSugarGrams,
			AverageSugarGrams: mealTypes[0].AverageSugarGrams,
		}
	}

	return domain.InsightPatterns{
		TopSugarMeals:  topMeals,
		TopSugarDishes: topDishes,
		WorstMealType:  worstMealType,
	}
}

func formatDate(value time.Time) string {
	return value.Format("2006-01-02")
}

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}
