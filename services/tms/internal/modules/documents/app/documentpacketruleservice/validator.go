package documentpacketruleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documentpacketrule"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
)

type Validator struct {
	validator tenantedvalidation.EntityValidator[*documentpacketrule.DocumentPacketRule]
}

func NewValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*documentpacketrule.DocumentPacketRule]().
			WithModelName("Document Packet Rule").
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *documentpacketrule.DocumentPacketRule,
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *documentpacketrule.DocumentPacketRule,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
