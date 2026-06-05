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

type GeminiDailyReportInterpreter struct {
	apiKey string
	model  string
	client *http.Client
}

const (
	dailyReportGeminiMaxAttempts = 5
)

func NewGeminiDailyReportInterpreter(apiKey string, model string) GeminiDailyReportInterpreter {
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultGeminiModel
	}

	return GeminiDailyReportInterpreter{
		apiKey: strings.TrimSpace(apiKey),
		model:  model,
		client: &http.Client{Timeout: 20 * time.Second},
	}
}

func (a GeminiDailyReportInterpreter) GenerateInsights(ctx context.Context, input domain.GenerateDailyReportSummaryInput) (domain.DailyReportAIInsights, error) {
	start := time.Now()

	if a.apiKey == "" {
		return domain.DailyReportAIInsights{}, fmt.Errorf("gemini api key is required")
	}

	payload, err := json.Marshal(geminiGenerateRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: buildDailyReportPrompt(input)},
				},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			Temperature:      0.4,
			ResponseMimeType: "application/json",
		},
	})
	if err != nil {
		return domain.DailyReportAIInsights{}, err
	}

	url := fmt.Sprintf(geminiGenerateAPIURL, a.model, a.apiKey)
	var raw []byte
	for attempt := 0; attempt < dailyReportGeminiMaxAttempts; attempt++ {
		if attempt > 0 {
			backoff := dailyReportRetryBackoff(attempt)
			time.Sleep(backoff)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return domain.DailyReportAIInsights{}, err
		}
		req.Header.Set("Content-Type", "application/json")

		attemptStartedAt := time.Now()
		resp, err := a.client.Do(req)
		if err != nil {
			if attempt < dailyReportGeminiMaxAttempts-1 {
				zap.L().Warn("gemini_daily_report_retry",
					zap.Int("attempt", attempt+1),
					zap.String("model", a.model),
					zap.Int64("latency_ms", time.Since(attemptStartedAt).Milliseconds()),
					zap.Error(err),
				)
				continue
			}
			return domain.DailyReportAIInsights{}, err
		}

		raw, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			if attempt < dailyReportGeminiMaxAttempts-1 {
				zap.L().Warn("gemini_daily_report_retry",
					zap.Int("attempt", attempt+1),
					zap.String("model", a.model),
					zap.Int64("latency_ms", time.Since(attemptStartedAt).Milliseconds()),
					zap.Error(err),
				)
				continue
			}
			return domain.DailyReportAIInsights{}, err
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			break
		}

		zap.L().Warn("gemini_daily_report_failed",
			zap.Int("status", resp.StatusCode),
			zap.String("model", a.model),
			zap.Int("attempt", attempt+1),
			zap.Int64("latency_ms", time.Since(attemptStartedAt).Milliseconds()),
		)
		if attempt < dailyReportGeminiMaxAttempts-1 && isRetryableGeminiStatus(resp.StatusCode) {
			zap.L().Warn("gemini_daily_report_retry",
				zap.Int("attempt", attempt+1),
				zap.Int("status", resp.StatusCode),
				zap.String("model", a.model),
			)
			continue
		}

		return domain.DailyReportAIInsights{}, fmt.Errorf("gemini request failed: status=%d body=%s", resp.StatusCode, string(raw))
	}

	var parsed geminiGenerateResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return domain.DailyReportAIInsights{}, err
	}

	text := strings.TrimSpace(parsed.FirstText())
	if text == "" {
		return domain.DailyReportAIInsights{}, fmt.Errorf("gemini returned empty content")
	}

	insights, err := parseDailyReportInsightsJSON(text)
	if err != nil {
		return domain.DailyReportAIInsights{}, err
	}

	zap.L().Info("gemini_daily_report_generated",
		zap.String("model", a.model),
		zap.String("date", input.Report.Date.Format("2006-01-02")),
		zap.Int("meal_count", input.Report.MealCount),
		zap.Int64("latency_ms", time.Since(start).Milliseconds()),
	)

	return insights, nil
}

func dailyReportRetryBackoff(attempt int) time.Duration {
	switch attempt {
	case 1:
		return time.Second
	case 2:
		return 3 * time.Second
	default:
		return 8 * time.Second
	}
}

func isRetryableGeminiStatus(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable || status >= 500
}

func parseDailyReportInsightsJSON(raw string) (domain.DailyReportAIInsights, error) {
	trimmed := strings.TrimSpace(raw)
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		trimmed = trimmed[start : end+1]
	}

	var insights domain.DailyReportAIInsights
	if err := json.Unmarshal([]byte(trimmed), &insights); err != nil {
		return domain.DailyReportAIInsights{}, fmt.Errorf("invalid daily report insights json: %w", err)
	}

	insights.Summary = strings.TrimSpace(insights.Summary)
	if insights.Summary == "" {
		return domain.DailyReportAIInsights{}, fmt.Errorf("daily report insights missing summary")
	}
	if insights.TopContributors == nil {
		insights.TopContributors = []domain.DailyReportTopContributor{}
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
