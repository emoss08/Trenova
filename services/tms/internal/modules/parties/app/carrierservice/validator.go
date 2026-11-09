package carrierservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker

	DB *postgres.Connection
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*carrier.Carrier]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*carrier.Carrier]().
			WithModelName("Carrier").
			WithIntegrityChecker(p.Integrity).
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
			WithReferenceCheck(
				"stateId",
				"us_states",
				"State does not exist",
				func(c *carrier.Carrier) pulid.ID {
					if c.StateID == nil {
						return ""
					}
					return *c.StateID
				},
			).
			WithReferenceCheck(
				"remitStateId",
				"us_states",
				"Remit state does not exist",
				func(c *carrier.Carrier) pulid.ID {
					if c.RemitStateID == nil {
						return ""
					}
					return *c.RemitStateID
				},
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *carrier.Carrier,
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *carrier.Carrier,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
