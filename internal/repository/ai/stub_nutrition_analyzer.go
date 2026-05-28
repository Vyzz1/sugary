package ai

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"time"

	"sugary/internal/domain"
)

type StubNutritionAnalyzer struct {
	minDelay time.Duration
	maxDelay time.Duration
	failure  int
	random   *rand.Rand
}

type StubNutritionAnalyzerConfig struct {
	MinDelay time.Duration
	MaxDelay time.Duration
	Failure  int
}

func NewStubNutritionAnalyzer() StubNutritionAnalyzer {
	return NewStubNutritionAnalyzerWithConfig(StubNutritionAnalyzerConfig{})
}

func NewStubNutritionAnalyzerWithConfig(cfg StubNutritionAnalyzerConfig) StubNutritionAnalyzer {
	if cfg.MinDelay < 0 {
		cfg.MinDelay = 0
	}
	if cfg.MaxDelay < cfg.MinDelay {
		cfg.MaxDelay = cfg.MinDelay
	}
	if cfg.Failure < 0 {
		cfg.Failure = 0
	}
	if cfg.Failure > 100 {
		cfg.Failure = 100
	}

	return StubNutritionAnalyzer{
		minDelay: cfg.MinDelay,
		maxDelay: cfg.MaxDelay,
		failure:  cfg.Failure,
		random:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s StubNutritionAnalyzer) AnalyzeMeal(ctx context.Context, input domain.AnalyzeMealInput) (domain.Nutrition, error) {
	s = s.withDefaults()

	if err := s.simulateLatency(ctx); err != nil {
		return domain.Nutrition{}, err
	}
	if s.shouldFail() {
		return domain.Nutrition{}, errors.New("stub ai transient failure")
	}

	name := strings.ToLower(input.DishName)
	switch {
	case strings.Contains(name, "milk tea"), strings.Contains(name, "cake"), strings.Contains(name, "soda"):
		return domain.Nutrition{
			EstimatedSugarGrams:   35,
			EstimatedCarbsGrams:   54,
			EstimatedProteinGrams: 6,
			EstimatedCalories:     420,
			RiskLevel:             "high",
			Notes:                 []string{"high-sugar drink or dessert pattern detected"},
		}, nil
	case strings.Contains(name, "pho"), strings.Contains(name, "salad"), strings.Contains(name, "egg"):
		return domain.Nutrition{
			EstimatedSugarGrams:   5,
			EstimatedCarbsGrams:   26,
			EstimatedProteinGrams: 14,
			EstimatedCalories:     320,
			RiskLevel:             "low",
			Notes:                 []string{"generally lower added sugar unless sweet sauces are included"},
		}, nil
	default:
		return domain.Nutrition{
			EstimatedSugarGrams:   18,
			EstimatedCarbsGrams:   42,
			EstimatedProteinGrams: 12,
			EstimatedCalories:     380,
			RiskLevel:             "medium",
			Notes:                 []string{"estimated from dish name; image-aware AI can refine this later"},
		}, nil
	}
}

func (s StubNutritionAnalyzer) simulateLatency(ctx context.Context) error {
	if s.maxDelay <= 0 {
		return nil
	}

	delay := s.minDelay
	if s.maxDelay > s.minDelay {
		window := s.maxDelay - s.minDelay
		delay += time.Duration(s.random.Int63n(int64(window) + 1))
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (s StubNutritionAnalyzer) shouldFail() bool {
	if s.failure <= 0 {
		return false
	}

	return s.random.Intn(100) < s.failure
}

func (s StubNutritionAnalyzer) withDefaults() StubNutritionAnalyzer {
	if s.random == nil {
		s.random = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	if s.maxDelay < s.minDelay {
		s.maxDelay = s.minDelay
	}
	if s.failure < 0 {
		s.failure = 0
	}
	if s.failure > 100 {
		s.failure = 100
	}
	return s
}
