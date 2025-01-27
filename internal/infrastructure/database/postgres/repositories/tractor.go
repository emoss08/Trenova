package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type TractorRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type tractorRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewTractorRepository(p TractorRepositoryParams) repositories.TractorRepository {
	log := p.Logger.With().
		Str("repository", "tractor").
		Logger()

	return &tractorRepository{
		db: p.DB,
		l:  &log,
	}
}

func (tr *tractorRepository) filterQuery(q *bun.SelectQuery, opts *repositories.ListTractorOptions) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "tr",
		Filter:     opts.Filter,
	})

	if opts.IncludeEquipmentDetails {
		q = q.Relation("EquipmentType").Relation("EquipmentManufacturer")
	}

	if opts.IncludeWorkerDetails {
		q = q.Relation("PrimaryWorker").Relation("PrimaryWorker.Profile")
		q = q.Relation("SecondaryWorker").Relation("SecondaryWorker.Profile")

	}

	if opts.Filter.Query != "" {
		q = q.Where("tr.code ILIKE ? OR tr.vin ILIKE ?", "%"+opts.Filter.Query+"%", "%"+opts.Filter.Query+"%")
	}

	return q.Limit(opts.Filter.Limit).Offset(opts.Filter.Offset)
}

func (tr *tractorRepository) List(ctx context.Context, opts *repositories.ListTractorOptions) (*ports.ListResult[*tractor.Tractor], error) {
	dba, err := tr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tr.l.With().
		Str("operation", "List").
		Str("buID", opts.Filter.TenantOpts.BuID.String()).
		Str("userID", opts.Filter.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*tractor.Tractor, 0)

	q := dba.NewSelect().Model(&entities)
	q = tr.filterQuery(q, opts)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan tractors")
		return nil, err
	}

	return &ports.ListResult[*tractor.Tractor]{
		Items: entities,
		Total: total,
	}, nil
}

func (tr *tractorRepository) GetByID(ctx context.Context, opts repositories.GetTractorByIDOptions) (*tractor.Tractor, error) {
	dba, err := tr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tr.l.With().
		Str("operation", "GetByID").
		Str("tractorID", opts.ID.String()).
		Logger()

	entity := new(tractor.Tractor)

	query := dba.NewSelect().Model(entity).
		Where("tr.id = ? AND tr.organization_id = ? AND tr.business_unit_id = ?", opts.ID, opts.OrgID, opts.BuID)

	// Include the worker details if requested
	if opts.IncludeWorkerDetails {
		query = query.Relation("PrimaryWorker", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Relation("PrimaryWorker.WorkerProfile")
		})

		query = query.Relation("SecondaryWorker", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Relation("SecondaryWorker.WorkerProfile")
		})
	}

	// Include the equipment details if requested
	if opts.IncludeEquipmentDetails {
		query = query.Relation("EquipmentType").Relation("EquipmentManufacturer")
	}

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Tractor not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get tractor")
		return nil, err
	}

	return entity, nil
}

func (tr *tractorRepository) Create(ctx context.Context, t *tractor.Tractor) (*tractor.Tractor, error) {
	dba, err := tr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tr.l.With().
		Str("operation", "Create").
		Str("orgID", t.OrganizationID.String()).
		Str("buID", t.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(t).Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("tractor", t).
				Msg("failed to insert tractor")
			return err
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create tractor")
		return nil, err
	}

	return t, nil
}

func (tr *tractorRepository) Update(ctx context.Context, t *tractor.Tractor) (*tractor.Tractor, error) {
	dba, err := tr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tr.l.With().
		Str("operation", "Update").
		Str("id", t.GetID()).
		Int64("version", t.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := t.Version

		t.Version++

		results, rErr := tx.NewUpdate().
			Model(t).
			WherePK().
			Where("tr.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("tractor", t).
				Msg("failed to update tractor")
			return err
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("tractor", t).
				Msg("failed to get rows affected")
			return err
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The Tractor (%s) has either been updated or deleted since the last request.", t.GetID()),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update tractor")
		return nil, err
	}

	return t, nil
}
