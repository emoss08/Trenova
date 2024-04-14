package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/feasibilitytoolcontrol"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"

	"github.com/google/uuid"
)

// FeasibilityToolControlService is the service for accounting control settings.
type FeasibilityToolControlService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewFeasibilityToolControlService creates a new accounting control service.
func NewFeasibilityToolControlService(s *api.Server) *FeasibilityToolControlService {
	return &FeasibilityToolControlService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetFeasibilityToolControl gets the feasibility tool control settings for an organization.
func (r *FeasibilityToolControlService) GetFeasibilityToolControl(ctx context.Context, orgID, buID uuid.UUID) (*ent.FeasibilityToolControl, error) {
	feasibilityToolControl, err := r.Client.FeasibilityToolControl.Query().Where(
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
func (r *FeasibilityToolControlService) UpdateFeasibilityToolControl(ctx context.Context, ftc *ent.FeasibilityToolControl) (*ent.FeasibilityToolControl, error) {
	updatedEntity := new(ent.FeasibilityToolControl)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateFeasibilityToolControlEntity(ctx, tx, ftc)
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

func (r *FeasibilityToolControlService) updateFeasibilityToolControlEntity(
	ctx context.Context, tx *ent.Tx, ftc *ent.FeasibilityToolControl,
) (*ent.FeasibilityToolControl, error) {
	updateOp := tx.FeasibilityToolControl.UpdateOneID(ftc.ID).
		SetOtpOperator(ftc.OtpOperator).
		SetOtpValue(ftc.OtpValue).
		SetMpwOperator(ftc.MpwOperator).
		SetMpwValue(ftc.MpwValue).
		SetMpdOperator(ftc.MpdOperator).
		SetMpdValue(ftc.MpdValue).
		SetMpgOperator(ftc.MpgOperator).
		SetMpgValue(ftc.MpgValue)

	updateEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update feasibility tool control entity")
	}

	return updateEntity, nil
}
