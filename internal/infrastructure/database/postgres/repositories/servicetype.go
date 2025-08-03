/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/postgressearch"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
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

func (str *serviceTypeRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListServiceTypeRequest,
) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "st",
		Filter:     req.Filter,
	})

	if req.Status != "" {
		status, err := domain.StatusFromString(req.Status)
		if err != nil {
			str.l.Error().Err(err).Str("status", req.Status).Msg("invalid status")
			return q
		}

		q = q.Where("st.status = ?", status)
	}

	if req.Filter.Query != "" {
		q = postgressearch.BuildSearchQuery(
			q,
			req.Filter.Query,
			(*servicetype.ServiceType)(nil),
		)
	}

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (str *serviceTypeRepository) List(
	ctx context.Context,
	req *repositories.ListServiceTypeRequest,
) (*ports.ListResult[*servicetype.ServiceType], error) {
	dba, err := str.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := str.l.With().
		Str("operation", "List").
		Str("buID", req.Filter.TenantOpts.BuID.String()).
		Str("userID", req.Filter.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*servicetype.ServiceType, 0)

	q := dba.NewSelect().Model(&entities)
	q = str.filterQuery(q, req)

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

func (str *serviceTypeRepository) GetByID(
	ctx context.Context,
	opts repositories.GetServiceTypeByIDOptions,
) (*servicetype.ServiceType, error) {
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

func (str *serviceTypeRepository) Create(
	ctx context.Context,
	st *servicetype.ServiceType,
) (*servicetype.ServiceType, error) {
	dba, err := str.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("service_type_repository").
			With("op", "create").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := str.l.With().
		Str("operation", "Create").
		Str("orgID", st.OrganizationID.String()).
		Str("buID", st.BusinessUnitID.String()).
		Logger()

	if _, err = dba.NewInsert().Model(st).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error().
			Err(err).
			Interface("serviceType", st).
			Msg("failed to insert service type")
		return nil, err
	}

	return st, nil
}

func (str *serviceTypeRepository) Update(
	ctx context.Context,
	st *servicetype.ServiceType,
) (*servicetype.ServiceType, error) {
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
			OmitZero().
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
				fmt.Sprintf(
					"Version mismatch. The Service Type (%s) has either been updated or deleted since the last request.",
					st.GetID(),
				),
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
