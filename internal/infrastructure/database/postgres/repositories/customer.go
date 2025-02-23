package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/postgressearch"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type CustomerRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type customerRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewCustomerRepository(p CustomerRepositoryParams) repositories.CustomerRepository {
	log := p.Logger.With().
		Str("repository", "customer").
		Logger()

	return &customerRepository{
		db: p.DB,
		l:  &log,
	}
}

func (cr *customerRepository) filterQuery(q *bun.SelectQuery, opts *repositories.ListCustomerOptions) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "cus",
		Filter:     opts.Filter,
	})

	if opts.IncludeState {
		q = q.Relation("State")
	}

	if opts.Filter.Query != "" {
		q = postgressearch.BuildSearchQuery(
			q,
			opts.Filter.Query,
			(*customer.Customer)(nil),
		)
	}

	return q.Limit(opts.Filter.Limit).Offset(opts.Filter.Offset)
}

func (cr *customerRepository) List(ctx context.Context, opts *repositories.ListCustomerOptions) (*ports.ListResult[*customer.Customer], error) {
	dba, err := cr.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	log := cr.l.With().
		Str("operation", "List").
		Str("buID", opts.Filter.TenantOpts.BuID.String()).
		Str("userID", opts.Filter.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*customer.Customer, 0)

	q := dba.NewSelect().Model(&entities)
	q = cr.filterQuery(q, opts)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan customers")
		return nil, err
	}

	return &ports.ListResult[*customer.Customer]{
		Items: entities,
		Total: total,
	}, nil
}

func (cr *customerRepository) GetByID(ctx context.Context, opts repositories.GetCustomerByIDOptions) (*customer.Customer, error) {
	dba, err := cr.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	log := cr.l.With().
		Str("operation", "GetByID").
		Str("customerID", opts.ID.String()).
		Logger()

	entity := new(customer.Customer)

	query := dba.NewSelect().Model(entity).
		Where("cus.id = ? AND cus.organization_id = ? AND cus.business_unit_id = ?", opts.ID, opts.OrgID, opts.BuID)

	if opts.IncludeState {
		query = query.Relation("State")
	}

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Customer not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get customer")
		return nil, err
	}

	return entity, nil
}

func (cr *customerRepository) Create(ctx context.Context, cus *customer.Customer) (*customer.Customer, error) {
	dba, err := cr.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	log := cr.l.With().
		Str("operation", "Create").
		Str("orgID", cus.OrganizationID.String()).
		Str("buID", cus.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(cus).Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("customer", cus).
				Msg("failed to insert customer")
			return iErr
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create customer")
		return nil, err
	}

	return cus, nil
}

func (cr *customerRepository) Update(ctx context.Context, cus *customer.Customer) (*customer.Customer, error) {
	dba, err := cr.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	log := cr.l.With().
		Str("operation", "Update").
		Str("id", cus.GetID()).
		Int64("version", cus.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := cus.Version

		cus.Version++

		results, rErr := tx.NewUpdate().
			Model(cus).
			Where("cus.version = ?", ov).
			WherePK().
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("customer", cus).
				Msg("failed to update customer")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("customer", cus).
				Msg("failed to get rows affected")
			return roErr
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The Customer (%s) has either been updated or deleted since the last request.", cus.GetID()),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update customer")
		return nil, err
	}

	return cus, nil
}
