package domain

import "errors"

var (
	ErrNonPositiveAmount = errors.New("amount must be positive")
	ErrEmptyCategory     = errors.New("category must not be empty")
	ErrZeroDate          = errors.New("date must be set")
	ErrNonPositiveLimit  = errors.New("limit must be positive")

	ErrBudgetExceeded = errors.New("budget exceeded")
	ErrBudgetNotFound = errors.New("budget not found")
	ErrInvalidPeriod  = errors.New("from must not be after to")
)

func IsValidationError(err error) bool {
	for _, target := range []error{
		ErrNonPositiveAmount, ErrEmptyCategory, ErrZeroDate, ErrNonPositiveLimit, ErrInvalidPeriod,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
