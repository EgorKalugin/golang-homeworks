package domain

import (
	"fmt"
	"strings"
)

type Validatable interface {
	Validate() error
}

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
