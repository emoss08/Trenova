package edibulk

import (
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const MaxItems = 500

type Failure struct {
	ID    pulid.ID `json:"id"`
	Error string   `json:"error"`
}

type Result struct {
	Succeeded []pulid.ID `json:"succeeded"`
	Failed    []Failure  `json:"failed"`
}

func NewResult(capacity int) *Result {
	return &Result{
		Succeeded: make([]pulid.ID, 0, capacity),
		Failed:    make([]Failure, 0),
	}
}

func ValidateIDs(field string, ids []pulid.ID) error {
	if len(ids) == 0 {
		return errortypes.NewValidationError(
			field,
			errortypes.ErrRequired,
			"At least one record must be selected",
		)
	}
	if len(ids) > MaxItems {
		return errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"A bulk action may target at most 500 records",
		)
	}
	return nil
}
