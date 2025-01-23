package domain

import (
	"context"

	"github.com/emoss08/trenova/internal/pkg/errors"
)

type Status string

const (
	StatusActive   = Status("Active")
	StatusInactive = Status("Inactive")
)

type Gender string

const (
	// GenderMale is the male gender
	GenderMale = Gender("Male")

	// GenderFemale is the female gender
	GenderFemale = Gender("Female")
)

type Validatable interface {
	Validate(ctx context.Context, multiErr *errors.MultiError)
	// ValidateUniqueness(ctx context.Context, tx bun.IDB, multiErr *errors.MultiError)
	GetTableName() string
}
