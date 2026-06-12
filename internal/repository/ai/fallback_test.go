package ai

import (
	"context"
	"errors"
	"testing"

	"sugary/internal/domain"
)

type stubWeeklyInterpreter struct {
	generateFn func(ctx context.Context, input domain.GenerateWeeklyReportSummaryInput) (domain.WeeklyReportAIInsights, error)
}

func (s stubWeeklyInterpreter) GenerateInsights(ctx context.Context, input domain.GenerateWeeklyReportSummaryInput) (domain.WeeklyReportAIInsights, error) {
	return s.generateFn(ctx, input)
}

func TestFallbackWeeklyReportInterpreterUsesFallbackWhenPrimaryFails(t *testing.T) {
	t.Parallel()

	fallbackCalled := false
	interpreter := NewFallbackWeeklyReportInterpreter(
		stubWeeklyInterpreter{
			generateFn: func(ctx context.Context, input domain.GenerateWeeklyReportSummaryInput) (domain.WeeklyReportAIInsights, error) {
				return domain.WeeklyReportAIInsights{}, errors.New("primary failed")
			},
		},
		stubWeeklyInterpreter{
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
}
