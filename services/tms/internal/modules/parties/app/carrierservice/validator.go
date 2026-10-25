package carrierservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator validationframework.TenantedEntityValidator[*carrier.Carrier]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := validationframework.NewKeelChecker(p.DB, carrierdb.CarriersSchema())
	states := validationframework.KeelReferenceCheck(p.DB, usstatedb.UsStatesSchema())

	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*carrier.Carrier]().
			WithModelName("Carrier").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
			WithUniqueField(
				"code",
				"code",
				"Carrier with this code already exists in your organization",
				func(c *carrier.Carrier) any { return c.Code },
			).
			WithUniqueField(
				"dotNumber",
				"dot_number",
				"Carrier with this DOT number already exists in your organization",
				func(c *carrier.Carrier) any { return c.DOTNumber },
			).
			WithUniqueField(
				"scac",
				"scac",
				"Carrier with this SCAC already exists in your organization",
				func(c *carrier.Carrier) any { return c.SCAC },
			).
			WithCustomReferenceCheck(
				"stateId",
				"State does not exist",
				func(c *carrier.Carrier) pulid.ID {
					if c.StateID == nil {
						return ""
					}
					return *c.StateID
				},
				states,
			).
			WithCustomReferenceCheck(
				"remitStateId",
				"Remit state does not exist",
				func(c *carrier.Carrier) pulid.ID {
					if c.RemitStateID == nil {
						return ""
					}
					return *c.RemitStateID
				},
				states,
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *carrier.Carrier,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *carrier.Carrier,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
