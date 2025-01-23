package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/trenova-app/transport/internal/core/domain/servicetype"
	"github.com/trenova-app/transport/internal/core/ports"
	"github.com/trenova-app/transport/internal/core/ports/db"
	"github.com/trenova-app/transport/internal/core/ports/repositories"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/logger"
	"github.com/trenova-app/transport/internal/pkg/utils/queryutils/queryfilters"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ServiceTypeRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type serviceTypeRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewServiceTypeRepository(p ServiceTypeRepositoryParams) repositories.ServiceTypeRepository {
	log := p.Logger.With().
		Str("repository", "servicetype").
		Logger()

	return &serviceTypeRepository{
		db: p.DB,
		l:  &log,
	}
}

func (str *serviceTypeRepository) filterQuery(q *bun.SelectQuery, opts *ports.LimitOffsetQueryOptions) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "st",
		Filter:     opts,
	})

	if opts.Query != "" {
		q = q.Where("st.code ILIKE ?", "%"+opts.Query+"%")
	}

	return q.Limit(opts.Limit).Offset(opts.Offset)
}

func (str *serviceTypeRepository) List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*servicetype.ServiceType], error) {
	dba, err := str.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := str.l.With().
		Str("operation", "List").
		Str("buID", opts.TenantOpts.BuID.String()).
		Str("userID", opts.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*servicetype.ServiceType, 0)

	q := dba.NewSelect().Model(&entities)
	q = str.filterQuery(q, opts)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan service types")
		return nil, eris.Wrap(err, "scan service types")
	}

	return &ports.ListResult[*servicetype.ServiceType]{
		Items: entities,
		Total: total,
	}, nil
}

func (str *serviceTypeRepository) GetByID(ctx context.Context, opts repositories.GetServiceTypeByIDOptions) (*servicetype.ServiceType, error) {
	dba, err := str.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := str.l.With().
		Str("operation", "GetByID").
		Str("serviceTypeID", opts.ID.String()).
		Logger()

	entity := new(servicetype.ServiceType)

	query := dba.NewSelect().Model(entity).
		Where("st.id = ? AND st.organization_id = ? AND st.business_unit_id = ?", opts.ID, opts.OrgID, opts.BuID)

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Service Type not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get service type")
		return nil, eris.Wrap(err, "get service type")
	}

	return entity, nil
}

func (str *serviceTypeRepository) Create(ctx context.Context, st *servicetype.ServiceType) (*servicetype.ServiceType, error) {
	dba, err := str.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := str.l.With().
		Str("operation", "Create").
		Str("orgID", st.OrganizationID.String()).
		Str("buID", st.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(st).Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("serviceType", st).
				Msg("failed to insert service type")
			return eris.Wrap(iErr, "insert service type")
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create service type")
		return nil, eris.Wrap(err, "create service type")
	}

	return st, nil
}

func (str *serviceTypeRepository) Update(ctx context.Context, st *servicetype.ServiceType) (*servicetype.ServiceType, error) {
	dba, err := str.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := str.l.With().
		Str("operation", "Update").
		Str("id", st.GetID()).
		Int64("version", st.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := st.Version

		st.Version++

		results, rErr := tx.NewUpdate().
			Model(st).
			WherePK().
			Where("st.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("serviceType", st).
				Msg("failed to update service type")
			return eris.Wrap(rErr, "update service type")
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("serviceType", st).
				Msg("failed to get rows affected")
			return eris.Wrap(roErr, "get rows affected")
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The Service Type (%s) has either been updated or deleted since the last request.", st.GetID()),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update service type")
		return nil, eris.Wrap(err, "update service type")
	}

	return st, nil
}
