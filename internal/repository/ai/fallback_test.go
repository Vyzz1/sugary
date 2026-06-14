package ai

import (
	"context"
	"errors"
	"testing"

	"sugary/internal/domain"
)

type stubWeeklyInterpreter struct {
	providerName string
	generateFn   func(ctx context.Context, input domain.GenerateWeeklyReportSummaryInput) (domain.WeeklyReportAIInsights, error)
}

func (s stubWeeklyInterpreter) GenerateInsights(ctx context.Context, input domain.GenerateWeeklyReportSummaryInput) (domain.WeeklyReportAIInsights, error) {
	return s.generateFn(ctx, input)
}

func (s stubWeeklyInterpreter) AIInsightProviderName() string {
	return s.providerName
}

func TestFallbackWeeklyReportInterpreterUsesFallbackWhenPrimaryFails(t *testing.T) {
	t.Parallel()

	fallbackCalled := false
	interpreter := NewFallbackWeeklyReportInterpreter(
		stubWeeklyInterpreter{
			providerName: "huggingface",
			generateFn: func(ctx context.Context, input domain.GenerateWeeklyReportSummaryInput) (domain.WeeklyReportAIInsights, error) {
				return domain.WeeklyReportAIInsights{}, errors.New("primary failed")
			},
		},
		stubWeeklyInterpreter{
			providerName: "gemini",
			generateFn: func(ctx context.Context, input domain.GenerateWeeklyReportSummaryInput) (domain.WeeklyReportAIInsights, error) {
				fallbackCalled = true
				return domain.WeeklyReportAIInsights{Summary: "fallback summary"}, nil
			},
		},
	)

	insights, err := interpreter.GenerateInsights(context.Background(), domain.GenerateWeeklyReportSummaryInput{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !fallbackCalled {
		t.Fatal("expected fallback interpreter to be called")
	}
	if insights.Summary != "fallback summary" {
		t.Fatalf("expected fallback summary, got %q", insights.Summary)
	}
	if interpreter.AIInsightProviderName() != "gemini" {
		t.Fatalf("expected fallback provider gemini, got %q", interpreter.AIInsightProviderName())
	}
}

func TestFallbackWeeklyReportInterpreterReportsPrimaryProviderOnSuccess(t *testing.T) {
	t.Parallel()

	interpreter := NewFallbackWeeklyReportInterpreter(
		stubWeeklyInterpreter{
			providerName: "huggingface",
			generateFn: func(ctx context.Context, input domain.GenerateWeeklyReportSummaryInput) (domain.WeeklyReportAIInsights, error) {
				return domain.WeeklyReportAIInsights{Summary: "primary summary"}, nil
			},
		},
		stubWeeklyInterpreter{
			providerName: "gemini",
			generateFn: func(ctx context.Context, input domain.GenerateWeeklyReportSummaryInput) (domain.WeeklyReportAIInsights, error) {
				t.Fatal("expected fallback not to be called")
				return domain.WeeklyReportAIInsights{}, nil
			},
		},
	)

	if _, err := interpreter.GenerateInsights(context.Background(), domain.GenerateWeeklyReportSummaryInput{}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if interpreter.AIInsightProviderName() != "huggingface" {
		t.Fatalf("expected primary provider huggingface, got %q", interpreter.AIInsightProviderName())
	}
}
