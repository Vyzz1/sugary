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

type GeminiWeeklyReportInterpreter struct {
	apiKey string
	model  string
	client *http.Client
}

func NewGeminiWeeklyReportInterpreter(apiKey string, model string) GeminiWeeklyReportInterpreter {
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultGeminiModel
	}

	return GeminiWeeklyReportInterpreter{
		apiKey: strings.TrimSpace(apiKey),
		model:  model,
		client: &http.Client{Timeout: 20 * time.Second},
	}
}

func (a GeminiWeeklyReportInterpreter) AIInsightProviderName() string {
	return "gemini"
}

func (a GeminiWeeklyReportInterpreter) GenerateInsights(ctx context.Context, input domain.GenerateWeeklyReportSummaryInput) (domain.WeeklyReportAIInsights, error) {
	start := time.Now()

	if a.apiKey == "" {
		return domain.WeeklyReportAIInsights{}, fmt.Errorf("gemini api key is required")
	}

	payload, err := json.Marshal(geminiGenerateRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: buildWeeklyReportPrompt(input)},
				},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			Temperature:      0.4,
			ResponseMimeType: "application/json",
		},
	})
	if err != nil {
		return domain.WeeklyReportAIInsights{}, err
	}

	url := fmt.Sprintf(geminiGenerateAPIURL, a.model, a.apiKey)
	var raw []byte
	for attempt := 0; attempt < dailyReportGeminiMaxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(dailyReportRetryBackoff(attempt))
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return domain.WeeklyReportAIInsights{}, err
		}
		req.Header.Set("Content-Type", "application/json")

		attemptStartedAt := time.Now()
		resp, err := a.client.Do(req)
		if err != nil {
			if attempt < dailyReportGeminiMaxAttempts-1 {
				zap.L().Warn("gemini_weekly_report_retry",
					zap.Int("attempt", attempt+1),
					zap.String("model", a.model),
					zap.Int64("latency_ms", time.Since(attemptStartedAt).Milliseconds()),
					zap.Error(err),
				)
				continue
			}
			return domain.WeeklyReportAIInsights{}, err
		}

		raw, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			if attempt < dailyReportGeminiMaxAttempts-1 {
				zap.L().Warn("gemini_weekly_report_retry",
					zap.Int("attempt", attempt+1),
					zap.String("model", a.model),
					zap.Int64("latency_ms", time.Since(attemptStartedAt).Milliseconds()),
					zap.Error(err),
				)
				continue
			}
			return domain.WeeklyReportAIInsights{}, err
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			break
		}

		zap.L().Warn("gemini_weekly_report_failed",
			zap.Int("status", resp.StatusCode),
			zap.String("model", a.model),
			zap.Int("attempt", attempt+1),
			zap.Int64("latency_ms", time.Since(attemptStartedAt).Milliseconds()),
		)
		if attempt < dailyReportGeminiMaxAttempts-1 && isRetryableGeminiStatus(resp.StatusCode) {
			zap.L().Warn("gemini_weekly_report_retry",
				zap.Int("attempt", attempt+1),
				zap.Int("status", resp.StatusCode),
				zap.String("model", a.model),
			)
			continue
		}

		return domain.WeeklyReportAIInsights{}, fmt.Errorf("gemini request failed: status=%d body=%s", resp.StatusCode, string(raw))
	}

	var parsed geminiGenerateResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return domain.WeeklyReportAIInsights{}, err
	}

	text := strings.TrimSpace(parsed.FirstText())
	if text == "" {
		return domain.WeeklyReportAIInsights{}, fmt.Errorf("gemini returned empty content")
	}

	insights, err := parseWeeklyReportInsightsJSON(text)
	if err != nil {
		return domain.WeeklyReportAIInsights{}, err
	}

	zap.L().Info("gemini_weekly_report_generated",
		zap.String("model", a.model),
		zap.String("week_start_date", input.Report.WeekStartDate.Format("2006-01-02")),
		zap.Int("meal_count", input.Report.MealCount),
		zap.Int64("latency_ms", time.Since(start).Milliseconds()),
	)

	return insights, nil
}

func parseWeeklyReportInsightsJSON(raw string) (domain.WeeklyReportAIInsights, error) {
	trimmed := strings.TrimSpace(raw)
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		trimmed = trimmed[start : end+1]
	}

	var insights domain.WeeklyReportAIInsights
	if err := json.Unmarshal([]byte(trimmed), &insights); err != nil {
		return domain.WeeklyReportAIInsights{}, fmt.Errorf("invalid weekly report insights json: %w", err)
	}

	insights.Summary = strings.TrimSpace(insights.Summary)
	if insights.Summary == "" {
		return domain.WeeklyReportAIInsights{}, fmt.Errorf("weekly report insights missing summary")
	}
	if insights.TopContributors == nil {
		insights.TopContributors = []domain.WeeklyReportTopContributor{}
	}
	if insights.Recommendations == nil {
		insights.Recommendations = []string{}
	}
	if insights.PatternSignals == nil {
		insights.PatternSignals = []string{}
	}

	for i := range insights.TopContributors {
		insights.TopContributors[i].DishName = strings.TrimSpace(insights.TopContributors[i].DishName)
		insights.TopContributors[i].MealType = strings.TrimSpace(strings.ToLower(insights.TopContributors[i].MealType))
		insights.TopContributors[i].RiskLevel = strings.TrimSpace(strings.ToLower(insights.TopContributors[i].RiskLevel))
	}
	for i := range insights.Recommendations {
		insights.Recommendations[i] = strings.TrimSpace(insights.Recommendations[i])
	}
	for i := range insights.PatternSignals {
		insights.PatternSignals[i] = strings.TrimSpace(insights.PatternSignals[i])
	}

	return insights, nil
}
