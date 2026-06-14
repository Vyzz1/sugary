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

	"go.uber.org/zap"

	"sugary/internal/domain"
)

const (
	defaultHuggingFaceAPIURL = "https://router.huggingface.co/v1/chat/completions"
	defaultHuggingFaceModel  = "Qwen/Qwen2.5-7B-Instruct"
)

type HuggingFaceConfig struct {
	APIToken string
	Model    string
	APIURL   string
}

type HuggingFaceNutritionAnalyzer struct {
	token  string
	model  string
	apiURL string
	client *http.Client
}

func NewHuggingFaceNutritionAnalyzer(config HuggingFaceConfig) HuggingFaceNutritionAnalyzer {
	return HuggingFaceNutritionAnalyzer{
		token:  strings.TrimSpace(config.APIToken),
		model:  normalizeHuggingFaceModel(config.Model),
		apiURL: normalizeHuggingFaceAPIURL(config.APIURL),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (a HuggingFaceNutritionAnalyzer) AnalyzeMeal(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
	start := time.Now()
	if a.token == "" {
		return domain.Nutrition{}, fmt.Errorf("huggingface api token is required")
	}

	text, err := a.chat(ctx, buildHuggingFaceMealContent(input), 0.2)
	if err != nil {
		return domain.Nutrition{}, err
	}

	nutrition, err := parseNutritionJSON(text)
	if err != nil {
		return domain.Nutrition{}, err
	}

	zap.L().Info("huggingface_meal_analyzed",
		zap.String("model", a.model),
		zap.String("dish_name", input.DishName),
		zap.Float64("estimated_sugar_grams", nutrition.EstimatedSugarGrams),
		zap.Int("estimated_calories", nutrition.EstimatedCalories),
		zap.String("risk_level", nutrition.RiskLevel),
		zap.Int64("latency_ms", time.Since(start).Milliseconds()),
	)

	return nutrition, nil
}

func (a HuggingFaceNutritionAnalyzer) chat(ctx context.Context, content any, temperature float64) (string, error) {
	payload, err := json.Marshal(huggingFaceChatRequest{
		Model: a.model,
		Messages: []huggingFaceMessage{
			{Role: "user", Content: content},
		},
		Temperature: temperature,
		ResponseFormat: huggingFaceResponseFormat{
			Type: "json_object",
		},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.apiURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		zap.L().Warn("huggingface_request_failed",
			zap.Int("status", resp.StatusCode),
			zap.String("model", a.model),
			zap.Int64("latency_ms", time.Since(start).Milliseconds()),
		)
		return "", fmt.Errorf("huggingface request failed: status=%d body=%s", resp.StatusCode, string(raw))
	}

	var parsed huggingFaceChatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}

	text := parsed.FirstContent()
	if text == "" {
		return "", fmt.Errorf("huggingface returned empty content")
	}

	return text, nil
}

type HuggingFaceDailyReportInterpreter struct {
	analyzer HuggingFaceNutritionAnalyzer
}

type HuggingFaceWeeklyReportInterpreter struct {
	analyzer HuggingFaceNutritionAnalyzer
}

func NewHuggingFaceDailyReportInterpreter(config HuggingFaceConfig) HuggingFaceDailyReportInterpreter {
	return HuggingFaceDailyReportInterpreter{
		analyzer: NewHuggingFaceNutritionAnalyzer(config),
	}
}

func NewHuggingFaceWeeklyReportInterpreter(config HuggingFaceConfig) HuggingFaceWeeklyReportInterpreter {
	return HuggingFaceWeeklyReportInterpreter{
		analyzer: NewHuggingFaceNutritionAnalyzer(config),
	}
}

func (i HuggingFaceDailyReportInterpreter) AIInsightProviderName() string {
	return "huggingface"
}

func (i HuggingFaceWeeklyReportInterpreter) AIInsightProviderName() string {
	return "huggingface"
}

func (i HuggingFaceDailyReportInterpreter) GenerateInsights(ctx context.Context, input domain.GenerateDailyReportSummaryInput) (domain.DailyReportAIInsights, error) {
	start := time.Now()

	text, err := i.analyzer.chat(ctx, buildDailyReportPrompt(input), 0.4)
	if err != nil {
		return domain.DailyReportAIInsights{}, err
	}

	insights, err := parseDailyReportInsightsJSON(text)
	if err != nil {
		return domain.DailyReportAIInsights{}, err
	}

	zap.L().Info("huggingface_daily_report_generated",
		zap.String("model", i.analyzer.model),
		zap.String("date", input.Report.Date.Format("2006-01-02")),
		zap.Int("meal_count", input.Report.MealCount),
		zap.Int64("latency_ms", time.Since(start).Milliseconds()),
	)

	return insights, nil
}

func (i HuggingFaceWeeklyReportInterpreter) GenerateInsights(ctx context.Context, input domain.GenerateWeeklyReportSummaryInput) (domain.WeeklyReportAIInsights, error) {
	start := time.Now()

	text, err := i.analyzer.chat(ctx, buildWeeklyReportPrompt(input), 0.4)
	if err != nil {
		return domain.WeeklyReportAIInsights{}, err
	}

	insights, err := parseWeeklyReportInsightsJSON(text)
	if err != nil {
		return domain.WeeklyReportAIInsights{}, err
	}

	zap.L().Info("huggingface_weekly_report_generated",
		zap.String("model", i.analyzer.model),
		zap.String("week_start_date", input.Report.WeekStartDate.Format("2006-01-02")),
		zap.Int("meal_count", input.Report.MealCount),
		zap.Int64("latency_ms", time.Since(start).Milliseconds()),
	)

	return insights, nil
}

type huggingFaceChatRequest struct {
	Model          string                    `json:"model"`
	Messages       []huggingFaceMessage      `json:"messages"`
	Temperature    float64                   `json:"temperature"`
	ResponseFormat huggingFaceResponseFormat `json:"response_format"`
}

type huggingFaceMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type huggingFaceContentPart struct {
	Type     string               `json:"type"`
	Text     string               `json:"text,omitempty"`
	ImageURL *huggingFaceImageURL `json:"image_url,omitempty"`
}

type huggingFaceImageURL struct {
	URL string `json:"url"`
}

func buildHuggingFaceMealContent(input domain.AnalyzeMealInput) any {
	prompt := buildPrompt(input)
	if input.ImageURL == nil {
		return prompt
	}

	imageURL := strings.TrimSpace(*input.ImageURL)
	if imageURL == "" {
		return prompt
	}

	return []huggingFaceContentPart{
		{
			Type: "text",
			Text: prompt,
		},
		{
			Type: "image_url",
			ImageURL: &huggingFaceImageURL{
				URL: imageURL,
			},
		},
	}
}

type huggingFaceResponseFormat struct {
	Type string `json:"type"`
}

type huggingFaceChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (r huggingFaceChatResponse) FirstContent() string {
	if len(r.Choices) == 0 {
		return ""
	}
	return strings.TrimSpace(r.Choices[0].Message.Content)
}

func normalizeHuggingFaceModel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return defaultHuggingFaceModel
	}
	return model
}

func normalizeHuggingFaceAPIURL(apiURL string) string {
	apiURL = strings.TrimSpace(apiURL)
	if apiURL == "" {
		return defaultHuggingFaceAPIURL
	}
	return apiURL
}
