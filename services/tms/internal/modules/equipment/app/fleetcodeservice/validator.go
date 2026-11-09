package fleetcodeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*fleetcode.FleetCode]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*fleetcode.FleetCode]().
			WithModelName("FleetCode").
			WithIntegrityChecker(p.Integrity).
			WithUniqueField(
				"code",
				"code",
				"Fleet code with this code already exists in your organization",
				func(fc *fleetcode.FleetCode) any { return fc.Code },
			).
			WithReference(tenantedvalidation.ReferenceConfig[*fleetcode.FleetCode]{
				FieldName: "managerId",
				Table:     "user_organization_memberships",
				Column:    "user_id",
				Message:   "Manager does not exist or is not a member of your organization",
				GetID:     func(fc *fleetcode.FleetCode) pulid.ID { return fc.ManagerID },
			}).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *fleetcode.FleetCode,
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *fleetcode.FleetCode,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
