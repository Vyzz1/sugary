package ai

import (
	"encoding/json"
	"fmt"
	"time"

	"sugary/internal/domain"
)

func buildDailyReportPrompt(input domain.GenerateDailyReportSummaryInput) string {
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

	return fmt.Sprintf(`You are generating a concise daily nutrition insight for a sugar tracking app.

Daily stats:
- date: %s
- meal_count: %d
- analyzed_meal_count: %d
- total_sugar_grams: %.1f
- average_sugar_grams: %.1f
- highest_risk_level: %s

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
- mention the overall sugar picture for the day
- call out the biggest likely contributor if obvious
- mention a practical suggestion for the next day
- include 0 to 3 top_contributors, 0 to 3 recommendations, and 0 to 3 pattern_signals
- do not use markdown
- do not invent medical claims or exact nutrition beyond the provided data`,
		input.Report.Date.Format("2006-01-02"),
		input.Report.MealCount,
		input.AnalyzedMealCount,
		input.Report.TotalSugarGrams,
		input.Report.AverageSugarGrams,
		input.Report.HighestRiskLevel,
		string(mealsJSON),
	)
}
