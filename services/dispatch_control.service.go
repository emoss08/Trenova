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
	ctx    context.Context
	client *ent.Client
}

// NewDispatchControlOps creates a new dispatch control service.
func NewDispatchControlOps(ctx context.Context) *DispatchControlOps {
	return &DispatchControlOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetDispatchControl gets the dispatch control settings for an organization.
func (r *DispatchControlOps) GetDispatchControl(orgID, buID uuid.UUID) (*ent.DispatchControl, error) {
	dispatchControl, err := r.client.DispatchControl.Query().Where(
		dispatchcontrol.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Only(r.ctx)
	if err != nil {
		return nil, err
	}

	return dispatchControl, nil
}

// UpdateDispatchControl updates the dispatch control settings for an organization.
func (r *DispatchControlOps) UpdateDispatchControl(dc ent.DispatchControl) (*ent.DispatchControl, error) {
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
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedDC, nil
}
