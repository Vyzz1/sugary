package handler

import (
	"context"
	"errors"
	"net/http"
	"net/url"
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
	logMeal logMealUseCase
}

type logMealRequest struct {
	DishName   string  `json:"dish_name"`
	MealType   string  `json:"meal_type"`
	ImageURL   *string `json:"image_url"`
	RecordedAt string  `json:"recorded_at"`
}

func NewMealHandler(logMeal logMealUseCase) MealHandler {
	return MealHandler{
		logMeal: logMeal,
	}
}

func (h MealHandler) Create(ctx *gin.Context) {
	var request logMealRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "bad_request", err.Error()))
		return
	}

	request.DishName = strings.TrimSpace(request.DishName)
	if request.DishName == "" {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "missing_dish_name", "dish_name is required"))
		return
	}
	request.MealType = strings.TrimSpace(strings.ToLower(request.MealType))
	if request.MealType != "" && !domain.IsValidMealType(request.MealType) {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "invalid_meal_type", "meal_type must be one of breakfast, lunch, dinner, snack, unspecified"))
		return
	}
	if request.ImageURL != nil {
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
		DishName:   request.DishName,
		MealType:   request.MealType,
		ImageURL:   request.ImageURL,
		RecordedAt: recordedAt,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, domain.ErrInvalidMealInput) {
			status = http.StatusBadRequest
		}

		ctx.JSON(status, httpresponse.Fail(ctx, "meal_create_failed", err.Error()))
		return
	}

	ctx.JSON(http.StatusCreated, httpresponse.OK(ctx, meal))
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
