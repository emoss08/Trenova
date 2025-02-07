package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type AssignmentRepositoryParams struct {
	fx.In

	DB       db.Connection
	MoveRepo repositories.ShipmentMoveRepository
	Logger   *logger.Logger
}

type assignmentRepository struct {
	db       db.Connection
	moveRepo repositories.ShipmentMoveRepository
	l        *zerolog.Logger
}

func NewAssignmentRepository(p AssignmentRepositoryParams) repositories.AssignmentRepository {
	log := p.Logger.With().
		Str("repository", "assignment").
		Logger()

	return &assignmentRepository{
		db:       p.DB,
		moveRepo: p.MoveRepo,
		l:        &log,
	}
}

func (ar *assignmentRepository) SingleAssign(ctx context.Context, a *shipment.Assignment) (*shipment.Assignment, error) {
	dba, err := ar.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := ar.l.With().
		Str("operation", "SingleAssignment").
		Str("orgID", a.OrganizationID.String()).
		Str("buID", a.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(a).Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("assignment", a).
				Msg("failed to insert assignment")
			return iErr
		}

		// Update the status of the move to assigneds
		_, err = ar.moveRepo.UpdateStatus(c, repositories.UpdateStatusOptions{
			GetMoveOpts: repositories.GetMoveByIDOptions{
				MoveID: a.ShipmentMoveID,
				OrgID:  a.OrganizationID,
				BuID:   a.BusinessUnitID,
			},
			Status: shipment.MoveStatusAssigned,
		})
		if err != nil {
			log.Error().Err(err).
				Interface("assignment", a).
				Msg("failed to update move status")
			return err
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create assignment")
		return nil, err
	}

	return a, nil
}
