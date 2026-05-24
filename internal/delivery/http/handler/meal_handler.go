package handler

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	httpresponse "sugary/internal/delivery/http/response"
	"sugary/internal/domain"
)

type logMealUseCase interface {
	Execute(ctx context.Context, input domain.LogMealInput) (domain.Meal, error)
}

type MealHandler struct {
	logMeal          logMealUseCase
	listMealsByDay   listMealsByDayUseCase
	listRecentMeals  listRecentMealsUseCase
	editMealAnalysis editMealAnalysisUseCase
	editMeal         editMealUseCase
	deleteMeal       deleteMealUseCase
}

type logMealRequest struct {
	SourceMealID *int64  `json:"source_meal_id"`
	DishName     string  `json:"dish_name"`
	MealType     string  `json:"meal_type"`
	ImageURL     *string `json:"image_url"`
	RecordedAt   string  `json:"recorded_at"`
}

type editMealUseCase interface {
	Execute(ctx context.Context, input domain.EditMealInput) (domain.Meal, error)
}

type listMealsByDayUseCase interface {
	Execute(ctx context.Context, day time.Time) ([]domain.Meal, time.Time, error)
}

type listRecentMealsUseCase interface {
	Execute(ctx context.Context, filter domain.RecentMealsFilter) ([]domain.Meal, int64, domain.RecentMealsFilter, error)
}

type editMealAnalysisUseCase interface {
	Execute(ctx context.Context, input domain.EditMealAnalysisInput) (domain.Meal, error)
}

type deleteMealUseCase interface {
	Execute(ctx context.Context, mealID int64) error
}

type editMealRequest struct {
	DishName   *string `json:"dish_name"`
	MealType   *string `json:"meal_type"`
	ImageURL   *string `json:"image_url"`
	RecordedAt *string `json:"recorded_at"`
}

type editMealAnalysisRequest struct {
	EstimatedSugarGrams   float64 `json:"estimated_sugar_grams"`
	EstimatedCarbsGrams   float64 `json:"estimated_carbs_grams"`
	EstimatedProteinGrams float64 `json:"estimated_protein_grams"`
	EstimatedCalories     int     `json:"estimated_calories"`
}

func NewMealHandler(
	logMeal logMealUseCase,
	listMealsByDay listMealsByDayUseCase,
	listRecentMeals listRecentMealsUseCase,
	editMealAnalysis editMealAnalysisUseCase,
	editMeal editMealUseCase,
	deleteMeal deleteMealUseCase,
) MealHandler {
	return MealHandler{
		logMeal:          logMeal,
		listMealsByDay:   listMealsByDay,
		listRecentMeals:  listRecentMeals,
		editMealAnalysis: editMealAnalysis,
		editMeal:         editMeal,
		deleteMeal:       deleteMeal,
	}
}

func (h MealHandler) Create(ctx *gin.Context) {
	var request logMealRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "bad_request", err.Error()))
		return
	}

	if request.SourceMealID != nil && *request.SourceMealID <= 0 {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_source_meal_id", "source_meal_id must be a positive integer"))
		return
	}

	request.DishName = strings.TrimSpace(request.DishName)
	if request.SourceMealID == nil && request.DishName == "" {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "missing_dish_name", "dish_name is required"))
		return
	}
	if request.SourceMealID != nil && request.DishName != "" {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_meal_input", "dish_name must be omitted when source_meal_id is provided"))
		return
	}

	request.MealType = strings.TrimSpace(strings.ToLower(request.MealType))
	if request.MealType != "" && !domain.IsValidMealType(request.MealType) {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_meal_type", "meal_type must be one of breakfast, lunch, dinner, snack, unspecified"))
		return
	}
	if request.ImageURL != nil {
		if request.SourceMealID != nil {
			ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_meal_input", "image_url must be omitted when source_meal_id is provided"))
			return
		}
		imageURL := strings.TrimSpace(*request.ImageURL)
		if imageURL == "" {
			ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_image_url", "image_url must not be empty"))
			return
		}
		if !isValidHTTPURL(imageURL) {
			ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_image_url", "image_url must be a valid http/https URL"))
			return
		}
		request.ImageURL = &imageURL
	}

	var recordedAt time.Time
	if request.RecordedAt != "" {
		parsed, err := time.Parse(time.RFC3339, request.RecordedAt)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_recorded_at", "recorded_at must be RFC3339"))
			return
		}
		recordedAt = parsed
	}

	meal, err := h.logMeal.Execute(ctx.Request.Context(), domain.LogMealInput{
		SourceMealID: request.SourceMealID,
		DishName:     request.DishName,
		MealType:     request.MealType,
		ImageURL:     request.ImageURL,
		RecordedAt:   recordedAt,
	})
	if err != nil {
		status := http.StatusInternalServerError
		code := "meal_create_failed"
		if errors.Is(err, domain.ErrInvalidMealInput) {
			status = http.StatusBadRequest
			code = "invalid_meal_input"
		}
		if errors.Is(err, domain.ErrMealNotFound) {
			status = http.StatusNotFound
			code = "source_meal_not_found"
		}

		ctx.JSON(status, httpresponse.Fail(ctx, code, err.Error()))
		return
	}

	ctx.JSON(http.StatusCreated, httpresponse.OK(ctx, meal))
}

func (h MealHandler) ListByDay(ctx *gin.Context) {
	day := time.Now().UTC()
	if dayParam := strings.TrimSpace(ctx.Query("date")); dayParam != "" {
		parsed, err := time.Parse("2006-01-02", dayParam)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_date", "date must be YYYY-MM-DD"))
			return
		}
		day = parsed
	}

	meals, normalizedDay, err := h.listMealsByDay.Execute(ctx.Request.Context(), day)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, httpresponse.Fail(ctx, "list_meals_failed", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, httpresponse.OKWithMeta(ctx, meals, gin.H{
		"date":  normalizedDay.Format("2006-01-02"),
		"count": len(meals),
	}))
}

func (h MealHandler) ListRecent(ctx *gin.Context) {
	filter := domain.RecentMealsFilter{
		Query:    strings.TrimSpace(ctx.Query("q")),
		Sort:     strings.TrimSpace(ctx.Query("sort")),
		Page:     1,
		PageSize: 20,
	}

	if rawPage := strings.TrimSpace(ctx.Query("page")); rawPage != "" {
		parsed, err := strconv.ParseInt(rawPage, 10, 32)
		if err != nil || parsed <= 0 {
			ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_page", "page must be a positive integer"))
			return
		}
		filter.Page = int32(parsed)
	}
	if rawPageSize := strings.TrimSpace(ctx.Query("page_size")); rawPageSize != "" {
		parsed, err := strconv.ParseInt(rawPageSize, 10, 32)
		if err != nil || parsed <= 0 {
			ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_page_size", "page_size must be a positive integer"))
			return
		}
		filter.PageSize = int32(parsed)
	}

	meals, total, normalized, err := h.listRecentMeals.Execute(ctx.Request.Context(), filter)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, httpresponse.Fail(ctx, "recent_meals_failed", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, httpresponse.OKWithMeta(ctx, meals, gin.H{
		"query":     normalized.Query,
		"sort":      normalized.Sort,
		"page":      normalized.Page,
		"page_size": normalized.PageSize,
		"total":     total,
	}))
}

func (h MealHandler) EditMeal(ctx *gin.Context) {
	mealID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || mealID <= 0 {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_meal_id", "meal id must be a positive integer"))
		return
	}

	var request editMealRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "bad_request", err.Error()))
		return
	}

	if request.MealType != nil {
		normalized := strings.TrimSpace(strings.ToLower(*request.MealType))
		request.MealType = &normalized
	}
	if request.DishName != nil {
		trimmed := strings.TrimSpace(*request.DishName)
		request.DishName = &trimmed
	}
	if request.ImageURL != nil {
		trimmed := strings.TrimSpace(*request.ImageURL)
		if trimmed != "" && !isValidHTTPURL(trimmed) {
			ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_image_url", "image_url must be a valid http/https URL"))
			return
		}
		request.ImageURL = &trimmed
	}

	var recordedAt *time.Time
	if request.RecordedAt != nil {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*request.RecordedAt))
		if err != nil {
			ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_recorded_at", "recorded_at must be RFC3339"))
			return
		}
		recordedAt = &parsed
	}

	meal, err := h.editMeal.Execute(ctx.Request.Context(), domain.EditMealInput{
		MealID:     mealID,
		DishName:   request.DishName,
		MealType:   request.MealType,
		ImageURL:   request.ImageURL,
		RecordedAt: recordedAt,
	})
	if err != nil {
		status := http.StatusInternalServerError
		code := "meal_edit_failed"
		switch {
		case errors.Is(err, domain.ErrInvalidMealInput):
			status = http.StatusBadRequest
			code = "invalid_meal_input"
		case errors.Is(err, domain.ErrNoMealChanges):
			status = http.StatusBadRequest
			code = "no_meal_changes"
		case errors.Is(err, domain.ErrMealNotFound):
			status = http.StatusNotFound
			code = "meal_not_found"
		}
		ctx.JSON(status, httpresponse.Fail(ctx, code, err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, httpresponse.OK(ctx, meal))
}

func (h MealHandler) EditAnalysis(ctx *gin.Context) {
	mealID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || mealID <= 0 {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_meal_id", "meal id must be a positive integer"))
		return
	}

	var request editMealAnalysisRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "bad_request", err.Error()))
		return
	}

	meal, err := h.editMealAnalysis.Execute(ctx.Request.Context(), domain.EditMealAnalysisInput{
		MealID: mealID,
		Nutrition: domain.Nutrition{
			EstimatedSugarGrams:   request.EstimatedSugarGrams,
			EstimatedCarbsGrams:   request.EstimatedCarbsGrams,
			EstimatedProteinGrams: request.EstimatedProteinGrams,
			EstimatedCalories:     request.EstimatedCalories,
		},
	})
	if err != nil {
		status := http.StatusInternalServerError
		code := "meal_analysis_update_failed"
		switch {
		case errors.Is(err, domain.ErrInvalidMealInput):
			status = http.StatusBadRequest
			code = "invalid_meal_id"
		case errors.Is(err, domain.ErrInvalidNutrition):
			status = http.StatusBadRequest
			code = "invalid_analysis_input"
		case errors.Is(err, domain.ErrMealNotFound):
			status = http.StatusNotFound
			code = "meal_not_found"
		}
		ctx.JSON(status, httpresponse.Fail(ctx, code, err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, httpresponse.OK(ctx, meal))
}

func (h MealHandler) DeleteMeal(ctx *gin.Context) {
	mealID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || mealID <= 0 {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_meal_id", "meal id must be a positive integer"))
		return
	}

	if err := h.deleteMeal.Execute(ctx.Request.Context(), mealID); err != nil {
		status := http.StatusInternalServerError
		code := "meal_delete_failed"
		switch {
		case errors.Is(err, domain.ErrInvalidMealInput):
			status = http.StatusBadRequest
			code = "invalid_meal_id"
		case errors.Is(err, domain.ErrMealNotFound):
			status = http.StatusNotFound
			code = "meal_not_found"
		}
		ctx.JSON(status, httpresponse.Fail(ctx, code, err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, httpresponse.OK(ctx, gin.H{"deleted": true}))
}

func isValidHTTPURL(raw string) bool {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	return parsed.Host != ""
}
