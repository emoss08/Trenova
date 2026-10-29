package documentcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
)

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) ValidateUpdate(
	_ context.Context,
	entity *documentcontrol.DocumentControl,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

var _ validationframework.TenantedEntity = (*documentcontrol.DocumentControl)(nil)
