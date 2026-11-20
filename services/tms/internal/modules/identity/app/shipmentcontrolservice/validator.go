package shipmentcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const (
	detentionPolicyTable      = "detention_policies"
	detentionPolicyActive     = "Active"
	defaultDetentionPolicyMsg = "The default detention policy must be an active policy that covers all freight from today on: no customer, facility, type or appointment scope, no end date and no start date in the future"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*shipmentcontrol.ShipmentControl]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{validator: buildValidator(p.Integrity)}
}

func buildValidator(
	integrity validationframework.IntegrityChecker,
) tenantedvalidation.EntityValidator[*shipmentcontrol.ShipmentControl] {
	return tenantedvalidation.NewBuilder[*shipmentcontrol.ShipmentControl]().
		WithModelName("ShipmentControl").
		WithIntegrityChecker(integrity).
		WithOptionalReferenceCheck(
			"detentionChargeId",
			"accessorial_charges",
			"Detention charge does not exist in your organization",
			func(sc *shipmentcontrol.ShipmentControl) pulid.ID {
				if sc.DetentionChargeID == nil {
					return ""
				}

				return *sc.DetentionChargeID
			},
		).
		WithReference(tenantedvalidation.ReferenceConfig[*shipmentcontrol.ShipmentControl]{
			FieldName: "defaultDetentionPolicyId",
			Table:     detentionPolicyTable,
			Message:   defaultDetentionPolicyMsg,
			Optional:  true,
			Conditions: []validationframework.Condition{
				{Column: "status", Value: detentionPolicyActive},
				{Column: "specificity_score", Value: 0},
				{Column: "appointment_stops_only", Value: false},
				{Column: "effective_end_date", Op: validationframework.ConditionIsNull},
				{
					Column: "effective_start_date",
					Op:     validationframework.ConditionNullOrNotAfterNow,
				},
			},
			GetID: func(sc *shipmentcontrol.ShipmentControl) pulid.ID {
				if sc.DefaultDetentionPolicyID == nil {
					return ""
				}

				return *sc.DefaultDetentionPolicyID
			},
		}).
		Build()
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *shipmentcontrol.ShipmentControl,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
