package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/feasibilitytoolcontrol"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

// FeasibilityControlOps is the service for feasibility tool control settings.
type FeasibilityControlOps struct {
	client *ent.Client
}

// NewFeasibilityControlOps creates a new feasibility tool control service.
func NewFeasibilityControlOps() *FeasibilityControlOps {
	return &FeasibilityControlOps{
		client: database.GetClient(),
	}
}

// GetFeasibilityToolControl gets the feasibility tool control settings for an organization.
func (r *FeasibilityControlOps) GetFeasibilityToolControl(ctx context.Context, orgID, buID uuid.UUID) (*ent.FeasibilityToolControl, error) {
	feasibilityToolControl, err := r.client.FeasibilityToolControl.Query().Where(
		feasibilitytoolcontrol.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Only(ctx)
	if err != nil {
		return nil, err
	}

	return feasibilityToolControl, nil
}

// UpdateFeasibilityToolControl updates the feasibility tool control settings for an organization.
func (r *FeasibilityControlOps) UpdateFeasibilityToolControl(ctx context.Context, ftc ent.FeasibilityToolControl) (*ent.FeasibilityToolControl, error) {
	updatedFTC, err := r.client.FeasibilityToolControl.
		UpdateOneID(ftc.ID).
		SetOtpOperator(ftc.OtpOperator).
		SetOtpValue(ftc.OtpValue).
		SetMpwOperator(ftc.MpwOperator).
		SetMpwValue(ftc.MpwValue).
		SetMpdOperator(ftc.MpdOperator).
		SetMpdValue(ftc.MpdValue).
		SetMpgOperator(ftc.MpgOperator).
		SetMpgValue(ftc.MpgValue).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedFTC, nil
}
