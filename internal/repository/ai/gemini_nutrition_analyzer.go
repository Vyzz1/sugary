package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"sugary/internal/domain"

	"go.uber.org/zap"
)

const (
	defaultGeminiModel   = "gemini-2.5-flash"
	geminiGenerateAPIURL = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"
)

type GeminiNutritionAnalyzer struct {
	apiKey string
	model  string
	client *http.Client
}

func NewGeminiNutritionAnalyzer(apiKey string, model string) GeminiNutritionAnalyzer {
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultGeminiModel
	}

	return GeminiNutritionAnalyzer{
		apiKey: strings.TrimSpace(apiKey),
		model:  model,
		client: &http.Client{Timeout: 20 * time.Second},
	}
}

func (a GeminiNutritionAnalyzer) AnalyzeMeal(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
	start := time.Now()

	if a.apiKey == "" {
		return domain.Nutrition{}, fmt.Errorf("gemini api key is required")
	}

	prompt := buildPrompt(input)
	reqBody := geminiGenerateRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			Temperature:      0.2,
			ResponseMimeType: "application/json",
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return domain.Nutrition{}, err
	}

	url := fmt.Sprintf(geminiGenerateAPIURL, a.model, a.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return domain.Nutrition{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return domain.Nutrition{}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.Nutrition{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		zap.L().Warn("gemini_request_failed",
			zap.Int("status", resp.StatusCode),
			zap.String("model", a.model),
			zap.Int64("latency_ms", time.Since(start).Milliseconds()),
		)
		return domain.Nutrition{}, fmt.Errorf("gemini request failed: status=%d body=%s", resp.StatusCode, string(raw))
	}

	var parsed geminiGenerateResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return domain.Nutrition{}, err
	}

	text := parsed.FirstText()
	if text == "" {
		return domain.Nutrition{}, fmt.Errorf("gemini returned empty content")
	}

	nutrition, err := parseNutritionJSON(text)
	if err != nil {
		return domain.Nutrition{}, err
	}

	zap.L().Info("gemini_meal_analyzed",
		zap.String("model", a.model),
		zap.String("dish_name", input.DishName),
		zap.Float64("estimated_sugar_grams", nutrition.EstimatedSugarGrams),
		zap.Int("estimated_calories", nutrition.EstimatedCalories),
		zap.String("risk_level", nutrition.RiskLevel),
		zap.Int64("latency_ms", time.Since(start).Milliseconds()),
	)

	return nutrition, nil
}

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
  "notes": ["short note 1", "short note 2"]
}

Rules:
- risk_level low if sugar < 10g, medium if 10-25g, high if >25g.
- notes must be concise and practical.
- no markdown, no extra keys, no explanation text.`, input.DishName, imageURL)
}

func parseNutritionJSON(raw string) (domain.Nutrition, error) {
	type nutritionPayload struct {
		EstimatedSugarGrams   float64  `json:"estimated_sugar_grams"`
		EstimatedCarbsGrams   float64  `json:"estimated_carbs_grams"`
		EstimatedProteinGrams float64  `json:"estimated_protein_grams"`
		EstimatedCalories     int      `json:"estimated_calories"`
		RiskLevel             string   `json:"risk_level"`
		Notes                 []string `json:"notes"`
	}

	trimmed := strings.TrimSpace(raw)
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		trimmed = trimmed[start : end+1]
	}

	var payload nutritionPayload
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return domain.Nutrition{}, fmt.Errorf("invalid nutrition json: %w", err)
	}

	payload.RiskLevel = strings.TrimSpace(strings.ToLower(payload.RiskLevel))
	if payload.RiskLevel != "low" && payload.RiskLevel != "medium" && payload.RiskLevel != "high" {
		return domain.Nutrition{}, fmt.Errorf("invalid risk_level from ai")
	}
	if payload.EstimatedSugarGrams < 0 {
		return domain.Nutrition{}, fmt.Errorf("invalid estimated_sugar_grams from ai")
	}
	if payload.EstimatedCarbsGrams < 0 {
		return domain.Nutrition{}, fmt.Errorf("invalid estimated_carbs_grams from ai")
	}
	if payload.EstimatedProteinGrams < 0 {
		return domain.Nutrition{}, fmt.Errorf("invalid estimated_protein_grams from ai")
	}
	if payload.EstimatedCalories < 0 {
		return domain.Nutrition{}, fmt.Errorf("invalid estimated_calories from ai")
	}

	notes := make([]string, 0, len(payload.Notes))
	for _, note := range payload.Notes {
		note = strings.TrimSpace(note)
		if note != "" {
			notes = append(notes, note)
		}
	}

	return domain.Nutrition{
		EstimatedSugarGrams:   payload.EstimatedSugarGrams,
		EstimatedCarbsGrams:   payload.EstimatedCarbsGrams,
		EstimatedProteinGrams: payload.EstimatedProteinGrams,
		EstimatedCalories:     payload.EstimatedCalories,
		RiskLevel:             payload.RiskLevel,
		Notes:                 notes,
	}, nil
}

type geminiGenerateRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature      float64 `json:"temperature"`
	ResponseMimeType string  `json:"responseMimeType"`
}

type geminiGenerateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (r geminiGenerateResponse) FirstText() string {
	if len(r.Candidates) == 0 {
		return ""
	}
	parts := r.Candidates[0].Content.Parts
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0].Text)
}
