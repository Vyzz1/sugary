package ai

import (
	"encoding/json"
	"fmt"
	"time"

	"sugary/internal/domain"
)

func buildWeeklyReportPrompt(input domain.GenerateWeeklyReportSummaryInput) string {
	type mealPromptItem struct {
		DishName   string   `json:"dish_name"`
		MealType   string   `json:"meal_type"`
		RecordedAt string   `json:"recorded_at"`
		SugarGrams float64  `json:"estimated_sugar_grams"`
		RiskLevel  string   `json:"risk_level"`
		Notes      []string `json:"notes,omitempty"`
	}

	items := make([]mealPromptItem, 0, len(input.Meals))
	for _, meal := range input.Meals {
		item := mealPromptItem{
			DishName:   meal.DishName,
			MealType:   meal.MealType,
			RecordedAt: meal.RecordedAt.UTC().Format(time.RFC3339),
		}
		if meal.Analysis != nil {
			item.SugarGrams = meal.Analysis.EstimatedSugarGrams
			item.RiskLevel = meal.Analysis.RiskLevel
			item.Notes = meal.Analysis.Notes
		}
		items = append(items, item)
	}

	mealsJSON, _ := json.Marshal(items)
	breakdownJSON, _ := json.Marshal(input.Report.DailyBreakdown)

	return fmt.Sprintf(`You are generating a concise weekly nutrition insight for a sugar tracking app.

Weekly stats:
- week_start_date: %s
- week_end_date: %s
- meal_count: %d
- analyzed_meal_count: %d
- total_sugar_grams: %.1f
- average_sugar_grams: %.1f
- highest_risk_level: %s

Daily breakdown JSON:
%s

Meals JSON:
%s

Return STRICT JSON only with this shape:
{
  "summary": "string",
  "top_contributors": [
    {
      "dish_name": "string",
      "meal_type": "string",
      "estimated_sugar_grams": number,
      "risk_level": "low|medium|high|unknown"
    }
  ],
  "recommendations": ["string"],
  "pattern_signals": ["string"]
}

Requirements:
- summary should be 2 to 4 concise sentences in English
- recommendations and pattern_signals must also be written in English
- do not translate dish_name values in top_contributors; keep the original meal names from the input
- mention the overall sugar picture for the week
- call out the biggest likely contributors and recurring patterns if obvious
- mention a practical suggestion for the next week
- include 0 to 5 top_contributors, 0 to 4 recommendations, and 0 to 4 pattern_signals
- do not use markdown
- do not invent medical claims or exact nutrition beyond the provided data`,
		input.Report.WeekStartDate.Format("2006-01-02"),
		input.Report.WeekEndDate.Format("2006-01-02"),
		input.Report.MealCount,
		input.Report.AnalyzedMealCount,
		input.Report.TotalSugarGrams,
		input.Report.AverageSugarGrams,
		input.Report.HighestRiskLevel,
		string(breakdownJSON),
		string(mealsJSON),
	)
}
