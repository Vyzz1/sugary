package domain

import "errors"

var (
	ErrInvalidMealInput = errors.New("invalid meal input")
	ErrInvalidDate      = errors.New("invalid date")
	ErrNotImplemented   = errors.New("not implemented")
)
