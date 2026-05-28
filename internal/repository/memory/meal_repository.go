package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"sugary/internal/domain"
	"sugary/internal/platform/timeutil"
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

func (r *MealRepository) ListByDay(ctx context.Context, filter domain.MealsByDayFilter) ([]domain.Meal, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	dayStart, dayEnd := timeutil.DayBoundsUTC(filter.Day)
	result := make([]domain.Meal, 0)
	for _, meal := range r.meals {
		if meal.DeletedAt != nil {
			continue
		}
		recordedAtUTC := meal.RecordedAt.UTC()
		if !recordedAtUTC.Before(dayStart) && recordedAtUTC.Before(dayEnd) {
			result = append(result, meal)
		}
	}

	sortRecentMeals(result, filter.Sort)

	return result, nil
}

func (r *MealRepository) ListRecentDistinct(ctx context.Context, filter domain.RecentMealsFilter) ([]domain.Meal, int64, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	meals := make([]domain.Meal, 0, len(r.meals))
	for _, meal := range r.meals {
		if meal.DeletedAt == nil {
			meals = append(meals, meal)
		}
	}

	sort.Slice(meals, func(i, j int) bool {
		if meals[i].RecordedAt.Equal(meals[j].RecordedAt) {
			return meals[i].ID > meals[j].ID
		}
		return meals[i].RecordedAt.After(meals[j].RecordedAt)
	})

	seen := make(map[string]struct{})
	result := make([]domain.Meal, 0)
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	for _, meal := range meals {
		if query != "" && !strings.Contains(strings.ToLower(meal.DishName), query) {
			continue
		}
		key := recentMealKey(meal)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, meal)
	}

	sortRecentMeals(result, filter.Sort)

	total := int64(len(result))
	start := int((filter.Page - 1) * filter.PageSize)
	if start >= len(result) {
		return []domain.Meal{}, total, nil
	}
	end := start + int(filter.PageSize)
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], total, nil
}

func (r *MealRepository) GetByID(ctx context.Context, mealID int64) (domain.Meal, error) {
	_ = ctx

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, meal := range r.meals {
		if meal.ID == mealID && meal.DeletedAt == nil {
			return meal, nil
		}
	}

	return domain.Meal{}, domain.ErrMealNotFound
}

func (r *MealRepository) UpdateMeta(ctx context.Context, mealID int64, mealType string, recordedAt time.Time) (domain.Meal, error) {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.meals {
		if r.meals[i].ID == mealID && r.meals[i].DeletedAt == nil {
			r.meals[i].MealType = mealType
			r.meals[i].RecordedAt = recordedAt
			return r.meals[i], nil
		}
	}
	return domain.Meal{}, domain.ErrMealNotFound
}

func (r *MealRepository) UpdateForReanalysis(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.meals {
		if r.meals[i].ID == meal.ID && r.meals[i].DeletedAt == nil {
			r.meals[i].DishName = meal.DishName
			r.meals[i].MealType = meal.MealType
			r.meals[i].ImageURL = meal.ImageURL
			r.meals[i].RecordedAt = meal.RecordedAt
			r.meals[i].AnalysisStatus = domain.AnalysisStatusProcessing
			r.meals[i].IsUserEdited = false
			r.meals[i].Analysis = nil
			return r.meals[i], nil
		}
	}
	return domain.Meal{}, domain.ErrMealNotFound
}

func (r *MealRepository) UpdateWithAnalysis(ctx context.Context, meal domain.Meal) (domain.Meal, error) {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.meals {
		if r.meals[i].ID == meal.ID && r.meals[i].DeletedAt == nil {
			r.meals[i].DishName = meal.DishName
			r.meals[i].MealType = meal.MealType
			r.meals[i].ImageURL = meal.ImageURL
			r.meals[i].RecordedAt = meal.RecordedAt
			r.meals[i].Analysis = meal.Analysis
			r.meals[i].IsUserEdited = false
			return r.meals[i], nil
		}
	}
	return domain.Meal{}, domain.ErrMealNotFound
}

func (r *MealRepository) UpdateAnalysis(ctx context.Context, mealID int64, nutrition domain.Nutrition) (domain.Meal, error) {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.meals {
		if r.meals[i].ID == mealID {
			r.meals[i].Analysis = &nutrition
			r.meals[i].IsUserEdited = true
			return r.meals[i], nil
		}
	}

	return domain.Meal{}, domain.ErrMealNotFound
}

func (r *MealRepository) SoftDelete(ctx context.Context, mealID int64) error {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	for i := range r.meals {
		if r.meals[i].ID == mealID && r.meals[i].DeletedAt == nil {
			r.meals[i].DeletedAt = &now
			return nil
		}
	}
	return domain.ErrMealNotFound
}

func (r *MealRepository) UpdateAnalysisResult(ctx context.Context, mealID int64, nutrition domain.Nutrition) (domain.Meal, error) {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.meals {
		if r.meals[i].ID == mealID && r.meals[i].DeletedAt == nil {
			r.meals[i].Analysis = &nutrition
			r.meals[i].AnalysisStatus = domain.AnalysisStatusCompleted
			r.meals[i].IsUserEdited = false
			return r.meals[i], nil
		}
	}

	return domain.Meal{}, domain.ErrMealNotFound
}

func (r *MealRepository) UpdateAnalysisStatus(ctx context.Context, mealID int64, status string) error {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.meals {
		if r.meals[i].ID == mealID && r.meals[i].DeletedAt == nil {
			r.meals[i].AnalysisStatus = status
			if status != domain.AnalysisStatusCompleted {
				r.meals[i].Analysis = nil
			}
			return nil
		}
	}

	return domain.ErrMealNotFound
}

func recentMealKey(meal domain.Meal) string {
	imageURL := ""
	if meal.ImageURL != nil {
		imageURL = *meal.ImageURL
	}
	return strings.ToLower(strings.TrimSpace(meal.DishName)) + "|" + imageURL
}

func sortRecentMeals(meals []domain.Meal, sortType string) {
	sort.Slice(meals, func(i, j int) bool {
		switch sortType {
		case "created_asc":
			if meals[i].RecordedAt.Equal(meals[j].RecordedAt) {
				return meals[i].ID < meals[j].ID
			}
			return meals[i].RecordedAt.Before(meals[j].RecordedAt)
		case "name_asc":
			left := strings.ToLower(meals[i].DishName)
			right := strings.ToLower(meals[j].DishName)
			if left == right {
				if meals[i].RecordedAt.Equal(meals[j].RecordedAt) {
					return meals[i].ID > meals[j].ID
				}
				return meals[i].RecordedAt.After(meals[j].RecordedAt)
			}
			return left < right
		case "name_desc":
			left := strings.ToLower(meals[i].DishName)
			right := strings.ToLower(meals[j].DishName)
			if left == right {
				if meals[i].RecordedAt.Equal(meals[j].RecordedAt) {
					return meals[i].ID > meals[j].ID
				}
				return meals[i].RecordedAt.After(meals[j].RecordedAt)
			}
			return left > right
		default:
			if meals[i].RecordedAt.Equal(meals[j].RecordedAt) {
				return meals[i].ID > meals[j].ID
			}
			return meals[i].RecordedAt.After(meals[j].RecordedAt)
		}
	})
}
