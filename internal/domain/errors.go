package domain

import "errors"

var (
	ErrInvalidMealInput = errors.New("invalid meal input")
	ErrInvalidNutrition = errors.New("invalid nutrition input")
	ErrInvalidUpload    = errors.New("invalid upload input")
	ErrNoMealChanges    = errors.New("no meal changes provided")
	ErrInvalidDate      = errors.New("invalid date")
	ErrInvalidRange     = errors.New("invalid range")
	ErrMealNotFound     = errors.New("meal not found")
	ErrNotImplemented   = errors.New("not implemented")
)
