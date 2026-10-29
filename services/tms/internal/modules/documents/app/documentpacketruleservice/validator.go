package documentpacketruleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentpacketrule"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentpacketrule/documentpacketruledb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/keelcheck"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*documentpacketrule.DocumentPacketRule]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*documentpacketrule.DocumentPacketRule]().
			WithModelName("Document Packet Rule").
			WithReferenceChecker(
				keelcheck.NewChecker(
					p.DB,
					documentpacketruledb.DocumentPacketRulesSchema(),
				),
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *documentpacketrule.DocumentPacketRule,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *documentpacketrule.DocumentPacketRule,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
