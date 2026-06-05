package ai

import (
	"fmt"
	"strings"

	"sugary/internal/domain"
)

func buildPrompt(input domain.AnalyzeMealInput) string {
	imageURL := "none"
	if input.ImageURL != nil && strings.TrimSpace(*input.ImageURL) != "" {
		imageURL = strings.TrimSpace(*input.ImageURL)
	}

	return fmt.Sprintf(`You are a nutrition estimation engine.
Estimate sugar and calories for this meal:
- dish_name: %q
- image_url: %q

Return STRICT JSON only with this shape:
{
  "estimated_sugar_grams": number,
  "estimated_carbs_grams": number,
  "estimated_protein_grams": number,
  "estimated_calories": integer,
  "risk_level": "low" | "medium" | "high",
  "notes": ["specific practical note 1", "specific practical note 2"]
}

Rules:
- risk_level low if sugar < 10g, medium if 10-25g, high if >25g.
- notes must contain 2 to 3 specific, practical notes written in English.
- each note should be 8 to 18 words.
- avoid generic notes like "Consider portion size" unless tied to this dish.
- if image_url is "none", mention that the estimate assumes a typical serving of the named dish.
- keep dish_name semantics from the input; do not translate the dish name because it is not part of the output schema.
- no markdown, no extra keys, no explanation text.`, input.DishName, imageURL)
}
