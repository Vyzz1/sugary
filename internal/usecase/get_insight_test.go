package usecase

import (
	"testing"
	"time"

	"sugary/internal/domain"
)

func TestResolveInsightPeriod(t *testing.T) {
	loc := time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)
	now := time.Date(2026, 6, 5, 15, 30, 0, 0, loc)

	tests := []struct {
		name          string
		rangeType     string
		wantDays      int
		wantFrom      string
		wantTo        string
		wantPrevFrom  string
		wantPrevTo    string
		wantRangeType string
	}{
		{
			name:          "7d",
			rangeType:     "7d",
			wantDays:      7,
			wantFrom:      "2026-05-30",
			wantTo:        "2026-06-05",
			wantPrevFrom:  "2026-05-23",
			wantPrevTo:    "2026-05-29",
			wantRangeType: "7d",
		},
		{
			name:          "30d",
			rangeType:     "30d",
			wantDays:      30,
			wantFrom:      "2026-05-07",
			wantTo:        "2026-06-05",
			wantPrevFrom:  "2026-04-07",
			wantPrevTo:    "2026-05-06",
			wantRangeType: "30d",
		},
		{
			name:          "90d",
			rangeType:     "90d",
			wantDays:      90,
			wantFrom:      "2026-03-08",
			wantTo:        "2026-06-05",
			wantPrevFrom:  "2025-12-08",
			wantPrevTo:    "2026-03-07",
			wantRangeType: "90d",
		},
		{
			name:          "default",
			rangeType:     "",
			wantDays:      30,
			wantFrom:      "2026-05-07",
			wantTo:        "2026-06-05",
			wantPrevFrom:  "2026-04-07",
			wantPrevTo:    "2026-05-06",
			wantRangeType: "30d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			period, err := ResolveInsightPeriod(tt.rangeType, now)
			if err != nil {
				t.Fatalf("ResolveInsightPeriod() error = %v", err)
			}

			if period.Days != tt.wantDays {
				t.Fatalf("Days = %d, want %d", period.Days, tt.wantDays)
			}
			if period.RangeType != tt.wantRangeType {
				t.Fatalf("RangeType = %q, want %q", period.RangeType, tt.wantRangeType)
			}
			assertDate(t, "From", period.From, tt.wantFrom)
			assertDate(t, "To", period.To, tt.wantTo)
			assertDate(t, "PreviousFrom", period.PreviousFrom, tt.wantPrevFrom)
			assertDate(t, "PreviousTo", period.PreviousTo, tt.wantPrevTo)
		})
	}
}

func TestResolveInsightPeriodInvalidRange(t *testing.T) {
	_, err := ResolveInsightPeriod("14d", time.Now())
	if err != domain.ErrInvalidRange {
		t.Fatalf("err = %v, want %v", err, domain.ErrInvalidRange)
	}
}

func TestCalculateTrend(t *testing.T) {
	tests := []struct {
		name          string
		current       float64
		previous      float64
		wantDirection string
		wantPercent   float64
	}{
		{name: "up", current: 42, previous: 35, wantDirection: "up", wantPercent: 20},
		{name: "down", current: 30, previous: 35, wantDirection: "down", wantPercent: -14.3},
		{name: "stable", current: 36, previous: 35, wantDirection: "stable", wantPercent: 2.9},
		{name: "zero stable", current: 0, previous: 0, wantDirection: "stable", wantPercent: 0},
		{name: "new", current: 20, previous: 0, wantDirection: "new", wantPercent: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateTrend(tt.current, tt.previous)
			if got.Direction != tt.wantDirection {
				t.Fatalf("Direction = %q, want %q", got.Direction, tt.wantDirection)
			}
			if got.Percent != tt.wantPercent {
				t.Fatalf("Percent = %.1f, want %.1f", got.Percent, tt.wantPercent)
			}
		})
	}
}

func TestFillDailySugarFillsMissingDates(t *testing.T) {
	loc := time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)
	period, err := ResolveInsightPeriod("7d", time.Date(2026, 6, 5, 10, 0, 0, 0, loc))
	if err != nil {
		t.Fatalf("ResolveInsightPeriod() error = %v", err)
	}

	points := FillDailySugar(period, []domain.DailySugarPoint{
		{
			Date:              "2026-05-31",
			TotalSugarGrams:   32.5,
			TotalCalories:     700,
			TotalCarbsGrams:   88,
			TotalProteinGrams: 24,
			MealCount:         3,
			RiskLevel:         "medium",
		},
		{
			Date:            "2026-06-05",
			TotalSugarGrams: 12,
			MealCount:       1,
			RiskLevel:       "low",
		},
	})

	if len(points) != 7 {
		t.Fatalf("len(points) = %d, want 7", len(points))
	}
	if points[0].Date != "2026-05-30" || points[0].TotalSugarGrams != 0 || points[0].TotalCalories != 0 || points[0].TotalCarbsGrams != 0 || points[0].TotalProteinGrams != 0 || points[0].MealCount != 0 || points[0].RiskLevel != "none" {
		t.Fatalf("first point = %+v, want zero point for 2026-05-30", points[0])
	}
	if points[1].Date != "2026-05-31" || points[1].TotalSugarGrams != 32.5 || points[1].TotalCalories != 700 || points[1].TotalCarbsGrams != 88 || points[1].TotalProteinGrams != 24 || points[1].TargetGrams != 25 {
		t.Fatalf("second point = %+v, want existing point with target", points[1])
	}
	if points[6].Date != "2026-06-05" || points[6].TotalSugarGrams != 12 {
		t.Fatalf("last point = %+v, want 2026-06-05 existing point", points[6])
	}
}

func assertDate(t *testing.T, name string, got time.Time, want string) {
	t.Helper()
	if got.Format("2006-01-02") != want {
		t.Fatalf("%s = %s, want %s", name, got.Format("2006-01-02"), want)
	}
}
