package repositories

import (
	"context"
	"database/sql"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ShipmentControlRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type shipmentControlRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewShipmentControlRepository(p ShipmentControlRepositoryParams) repositories.ShipmentControlRepository {
	log := p.Logger.With().
		Str("repository", "shipmentcontrol").
		Logger()

	return &shipmentControlRepository{
		db: p.DB,
		l:  &log,
	}
}

func (r *shipmentControlRepository) GetByOrgID(ctx context.Context, orgID pulid.ID) (*shipment.ShipmentControl, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetByOrgID").
		Str("orgID", orgID.String()).
		Logger()

	entity := new(shipment.ShipmentControl)

	query := dba.NewSelect().Model(entity).Where("sc.organization_id = ?", orgID)

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Msg("shipment control not found within your organization")
			return nil, errors.NewNotFoundError("Shipment control not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get shipment control")
		return nil, eris.Wrap(err, "get shipment control")
	}

	return entity, nil
}
