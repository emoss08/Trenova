package repositories

import (
	"context"
	"database/sql"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
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

func (ar *assignmentRepository) GetByID(ctx context.Context, opts repositories.GetAssignmentByIDOptions) (*shipment.Assignment, error) {
	dba, err := ar.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := ar.l.With().
		Str("operation", "GetByID").
		Str("assignmentID", opts.ID.String()).
		Logger()

	entity := new(shipment.Assignment)

	query := dba.NewSelect().Model(entity).
		Where("a.id = ? AND a.organization_id = ? AND a.business_unit_id = ?", opts.ID, opts.OrganizationID, opts.BusinessUnitID)

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Assignment not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get assignment")
		return nil, eris.Wrap(err, "get assignment")
	}

	return entity, nil
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
		// Insert the assignment
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

func (ar *assignmentRepository) Reassign(ctx context.Context, a *shipment.Assignment) (*shipment.Assignment, error) {
	dba, err := ar.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := ar.l.With().
		Str("operation", "Reassign").
		Str("orgID", a.OrganizationID.String()).
		Str("buID", a.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// Increment the version of the assignment
		ov := a.Version
		a.Version++

		// Update the existing assignment
		if _, err = tx.NewUpdate().
			Model(a).
			Set("tractor_id = ?", a.TractorID).
			Set("trailer_id = ?", a.TrailerID).
			Set("primary_worker_id = ?", a.PrimaryWorkerID).
			Set("secondary_worker_id = ?", a.SecondaryWorkerID).
			Set("version = ?", a.Version).
			WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
				return q.Where("a.id = ?", a.ID).
					Where("a.organization_id = ?", a.OrganizationID).
					Where("a.version = ?", ov)
			}).
			Exec(c); err != nil {
			log.Error().Err(err).
				Interface("assignment", a).
				Msg("failed to update assignment")
			return err
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to reassign assignment")
		return nil, err
	}

	return a, nil
}
