package services

import (
	"context"
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/dispatchcontrol"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/google/uuid"
)

// DispatchControlService is the service for dispatch control settings.
type DispatchControlService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewDispatchControlService creates a new dispatch control service.
func NewDispatchControlService(s *api.Server) *DispatchControlService {
	return &DispatchControlService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetDispatchControl gets the dispatch control settings for an organization.
func (r *DispatchControlService) GetDispatchControl(ctx context.Context, orgID, buID uuid.UUID) (*ent.DispatchControl, error) {
	dispatchControl, err := r.Client.DispatchControl.Query().Where(
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
func (r *DispatchControlService) UpdateDispatchControl(ctx context.Context, dc *ent.DispatchControl) (*ent.DispatchControl, error) {
	updatedEntity := new(ent.DispatchControl)
	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateDispatchControl(ctx, tx, dc)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *DispatchControlService) updateDispatchControl(
	ctx context.Context, tx *ent.Tx, dc *ent.DispatchControl,
) (*ent.DispatchControl, error) {
	updateOp := tx.DispatchControl.UpdateOneID(dc.ID).
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
		SetTractorWorkerFleetConstraint(dc.TractorWorkerFleetConstraint)

	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
