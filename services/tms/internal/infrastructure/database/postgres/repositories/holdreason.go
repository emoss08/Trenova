package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/querybuilder"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type HoldReasonRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type holdReasonRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewHoldReasonRepository(p HoldReasonRepositoryParams) repositories.HoldReasonRepository {
	log := p.Logger.With().
		Str("repository", "hold_reason").
		Logger()

	return &holdReasonRepository{
		db: p.DB,
		l:  &log,
	}
}

func (hr *holdReasonRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListHoldReasonRequest,
) *bun.SelectQuery {
	qb := querybuilder.NewWithPostgresSearch(
		q,
		"hr",
		repositories.HoldReasonFieldConfig,
		(*shipment.HoldReason)(nil),
	)
	qb.ApplyTenantFilters(req.Filter.TenantOpts)

	if req.Filter != nil {
		qb.ApplyFilters(req.Filter.FieldFilters)

		if len(req.Filter.Sort) > 0 {
			qb.ApplySort(req.Filter.Sort)
		}

		if req.Filter.Query != "" {
			qb.ApplyTextSearch(req.Filter.Query, []string{"code", "label", "description"})
		}

		q = qb.GetQuery()
	}

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (hr *holdReasonRepository) List(
	ctx context.Context,
	req *repositories.ListHoldReasonRequest,
) (*ports.ListResult[*shipment.HoldReason], error) {
	dba, err := hr.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := hr.l.With().
		Str("operation", "List").
		Str("buID", req.Filter.TenantOpts.BuID.String()).
		Logger()

	entities := make([]*shipment.HoldReason, 0, req.Filter.Limit)

	q := dba.NewSelect().Model(&entities)
	q = hr.filterQuery(q, req)

	log.Info().Interface("req", req).Msg("req in list op for hold reason")
	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan hold reasons")
		return nil, err
	}

	return &ports.ListResult[*shipment.HoldReason]{
		Items: entities,
		Total: total,
	}, nil
}

func (hr *holdReasonRepository) GetByID(
	ctx context.Context,
	req *repositories.GetHoldReasonByIDRequest,
) (*shipment.HoldReason, error) {
	dba, err := hr.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := hr.l.With().
		Str("operation", "GetByID").
		Str("holdReasonID", req.ID.String()).
		Logger()

	entity := new(shipment.HoldReason)

	query := dba.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("hr.id = ?", req.ID).
				Where("hr.organization_id = ?", req.OrgID).
				Where("hr.business_unit_id = ?", req.BuID)
		})

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Hold reason not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get hold reason")
		return nil, err
	}

	return entity, nil
}

func (hr *holdReasonRepository) Create(
	ctx context.Context,
	h *shipment.HoldReason,
) (*shipment.HoldReason, error) {
	dba, err := hr.db.WriteDB(ctx)
	if err != nil {
		return nil, err
	}

	log := hr.l.With().
		Str("operation", "Create").
		Str("orgID", h.OrganizationID.String()).
		Str("buID", h.BusinessUnitID.String()).
		Logger()

	// * set the default sort order based on the type
	hr.defaultSortOrderByType(h)

	if _, err = dba.NewInsert().Model(h).Returning("*").Exec(ctx); err != nil {
		log.Error().Err(err).Msg("failed to insert hold reason")
		return nil, err
	}

	return h, nil
}

func (hr *holdReasonRepository) Update(
	ctx context.Context,
	h *shipment.HoldReason,
) (*shipment.HoldReason, error) {
	dba, err := hr.db.WriteDB(ctx)
	if err != nil {
		return nil, err
	}

	log := hr.l.With().
		Str("operation", "Update").
		Str("holdReasonID", h.GetID()).
		Int64("version", h.Version).
		Logger()

	// * set the default sort order based on the type
	hr.defaultSortOrderByType(h)

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := h.Version

		h.Version++

		results, rErr := tx.NewUpdate().
			Model(h).
			WherePK().
			OmitZero().
			Where("hr.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("holdReason", h).
				Msg("failed to update hold reason")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).Msg("failed to get rows affected")
			return roErr
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The Hold Reason (%s) has either been updated or deleted since the last request.",
					h.GetID(),
				),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update hold reason")
		return nil, err
	}

	return h, nil
}

func (hr *holdReasonRepository) defaultSortOrderByType(h *shipment.HoldReason) {
	switch h.Type {
	case shipment.HoldCompliance:
		h.SortOrder = 10
	case shipment.HoldOperational:
		h.SortOrder = 110
	case shipment.HoldCustomer:
		h.SortOrder = 210
	case shipment.HoldFinance:
		h.SortOrder = 310
	}
}
