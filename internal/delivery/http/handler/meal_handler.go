package handler

import (
	"context"
	"errors"
	"net/http"
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
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail("bad_request", err.Error()))
		return
	}

	var recordedAt time.Time
	if request.RecordedAt != "" {
		parsed, err := time.Parse(time.RFC3339, request.RecordedAt)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, httpresponse.Fail("invalid_recorded_at", "recorded_at must be RFC3339"))
			return
		}
		recordedAt = parsed
	}

	meal, err := h.logMeal.Execute(ctx.Request.Context(), domain.LogMealInput{
		DishName:   request.DishName,
		ImageURL:   request.ImageURL,
		RecordedAt: recordedAt,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, domain.ErrInvalidMealInput) {
			status = http.StatusBadRequest
		}

		ctx.JSON(status, httpresponse.Fail("meal_create_failed", err.Error()))
		return
	}

	ctx.JSON(http.StatusCreated, httpresponse.OK(meal))
}
