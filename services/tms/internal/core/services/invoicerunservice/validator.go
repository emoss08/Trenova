package invoicerunservice

import (
	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In
}

type Validator struct{}

func NewValidator(ValidatorParams) *Validator {
	return &Validator{}
}

func (v *Validator) ValidateCreate(entity *invoicerun.InvoiceRun) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// ValidateCommit guards the one transition that issues money to customers.
func (v *Validator) ValidateCommit(entity *invoicerun.InvoiceRun) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if !invoicerun.CanTransition(entity.Status, invoicerun.StatusCommitting) {
		multiErr.Add(
			"status",
			errortypes.ErrInvalidOperation,
			"An invoice run that is {0} cannot be committed", string(entity.Status),
		)
	}
	if len(entity.Groups) == 0 {
		multiErr.Add(
			"groups",
			errortypes.ErrInvalidOperation,
			"There is nothing on this run to invoice",
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// ValidateAdjust rejects an edit to a run that is no longer a proposal.
func (v *Validator) ValidateAdjust(entity *invoicerun.InvoiceRun) *errortypes.MultiError {
	if entity.Status.IsEditable() {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		"status",
		errortypes.ErrInvalidOperation,
		"An invoice run that is {0} can no longer be adjusted", string(entity.Status),
	)

	return multiErr
}
