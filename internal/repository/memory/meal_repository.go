package memory

import (
	"context"
	"sync"
	"time"

	"sugary/internal/domain"
)

type MealRepository struct {
	mu     sync.RWMutex
	nextID int64
	meals  []domain.Meal
}

func NewMealRepository() *MealRepository {
	return &MealRepository{
		nextID: 1,
		meals:  make([]domain.Meal, 0),
	}
}

func (r *MealRepository) Create(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	meal.ID = r.nextID
	r.nextID++
	r.meals = append(r.meals, meal)

	return meal, nil
}

func (r *MealRepository) ListByDay(ctx context.Context, day time.Time) ([]domain.Meal, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.Meal, 0)
	for _, meal := range r.meals {
		if sameDayUTC(meal.RecordedAt, day) {
			result = append(result, meal)
		}
	}

	return result, nil
}

func sameDayUTC(left time.Time, right time.Time) bool {
	left = left.UTC()
	right = right.UTC()

	return left.Year() == right.Year() &&
		left.Month() == right.Month() &&
		left.Day() == right.Day()
}
