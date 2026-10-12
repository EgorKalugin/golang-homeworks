package transactions

import (
	"errors"
	"fmt"
	"strings"
)

type Validatable interface {
	Validate() error
}

var (
	ErrEmptyCategory    = errors.New("category must not be empty")
	ErrZeroDate         = errors.New("date must be set")
	ErrNonPositiveLimit = errors.New("limit must be positive")
)

func CheckValid(v Validatable) error {
	return v.Validate()
}

func (tx Transaction) Validate() error {
	if tx.Amount <= 0 {
		return fmt.Errorf("%w, got %d", ErrNonPositiveAmount, tx.Amount)
	}
	if strings.TrimSpace(tx.Category) == "" {
		return ErrEmptyCategory
	}
	if tx.Date.IsZero() {
		return ErrZeroDate
	}
	return nil
}

func (b Budget) Validate() error {
	if strings.TrimSpace(b.Category) == "" {
		return ErrEmptyCategory
	}
	if b.Limit <= 0 {
		return fmt.Errorf("%w, got %d", ErrNonPositiveLimit, b.Limit)
	}
	return nil
}
