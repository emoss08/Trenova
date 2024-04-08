package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/dispatchcontrol"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

// DispatchControlOps is the service for dispatch control settings.
type DispatchControlOps struct {
	client *ent.Client
}

// NewDispatchControlOps creates a new dispatch control service.
func NewDispatchControlOps() *DispatchControlOps {
	return &DispatchControlOps{
		client: database.GetClient(),
	}
}

// GetDispatchControl gets the dispatch control settings for an organization.
func (r *DispatchControlOps) GetDispatchControl(ctx context.Context, orgID, buID uuid.UUID) (*ent.DispatchControl, error) {
	dispatchControl, err := r.client.DispatchControl.Query().Where(
		dispatchcontrol.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Only(ctx)
	if err != nil {
		return nil, err
	}

	return dispatchControl, nil
}

// UpdateDispatchControl updates the dispatch control settings for an organization.
func (r *DispatchControlOps) UpdateDispatchControl(ctx context.Context, dc ent.DispatchControl) (*ent.DispatchControl, error) {
	updatedDC, err := r.client.DispatchControl.
		UpdateOneID(dc.ID).
		SetRecordServiceIncident(dc.RecordServiceIncident).
		SetDeadheadTarget(dc.DeadheadTarget).
		SetMaxShipmentWeightLimit(dc.MaxShipmentWeightLimit).
		SetGracePeriod(dc.GracePeriod).
		SetEnforceWorkerAssign(dc.EnforceWorkerAssign).
		SetTrailerContinuity(dc.TrailerContinuity).
		SetDupeTrailerCheck(dc.DupeTrailerCheck).
		SetMaintenanceCompliance(dc.MaintenanceCompliance).
		SetRegulatoryCheck(dc.RegulatoryCheck).
		SetPrevShipmentOnHold(dc.PrevShipmentOnHold).
		SetWorkerTimeAwayRestriction(dc.WorkerTimeAwayRestriction).
		SetTractorWorkerFleetConstraint(dc.TractorWorkerFleetConstraint).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedDC, nil
}
