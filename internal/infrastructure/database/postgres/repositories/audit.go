package repositories

import (
	"context"
	"database/sql"

	"github.com/emoss08/trenova/internal/core/domain/audit"
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

// AuditRepositoryParams contains the dependencies for the AuditRepository.
// This includes database connection and logger.
type AuditRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// auditRepository implements the AuditRepository interface.
//
// It provides methods to interact with the audit table in the database.
type auditRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewAuditRepository initializes a new instance of auditRepository with its dependencies.
//
// Parameters:
//   - p: AuditRepositoryParams containing database connection and logger.
//
// Returns:
//   - A new instance of auditRepository.
func NewAuditRepository(p AuditRepositoryParams) repositories.AuditRepository {
	log := p.Logger.With().
		Str("repository", "audit").
		Str("component", "database").
		Logger()

	return &auditRepository{
		db: p.DB,
		l:  &log,
	}
}

// filterQuery filters the query based on the tenant options
//
// Parameters:
//   - q: The query to filter.
//   - opts: The options for the operation.
//
// Returns:
//   - A filtered query.
func (ar *auditRepository) filterQuery(q *bun.SelectQuery, opts *ports.LimitOffsetQueryOptions) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "ae",
		Filter:     opts,
	})

	q = q.Relation("User")

	// * Order by the created at date
	q = q.Order("ae.timestamp DESC")

	return q.Limit(opts.Limit).Offset(opts.Offset)
}

// GetByID fetches an audit entry by id
//
// Parameters:
//   - ctx: The context for the operation.
//   - opts: The options for the operation.
//
// Returns:
//   - An audit entry.
//   - An error if the operation fails.
func (ar *auditRepository) GetByID(ctx context.Context, opts repositories.GetAuditEntryByIDOptions) (*audit.Entry, error) {
	dba, err := ar.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := ar.l.With().
		Str("operation", "GetByID").
		Str("auditEntryID", opts.ID.String()).
		Logger()

	entity := new(audit.Entry)

	q := dba.NewSelect().Model(entity).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("ae.id = ?", opts.ID).
				Where("ae.organization_id = ?", opts.OrgID).
				Where("ae.business_unit_id = ?", opts.BuID)
		})

	// * Include the user relation
	q = q.Relation("User")

	if err = q.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Err(err).Msg("failed to get audit entry")
			return nil, errors.NewNotFoundError("Audit Entry not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get audit entry")
		return nil, eris.Wrap(err, "get audit entry by id")
	}

	return entity, nil
}

// List fetches a lists of audit entries
//
// Parameters:
//   - ctx: The context for the operation.
//   - opts: The options for the operation.
//
// Returns:
//   - A list of audit entries.
//   - An error if the operation fails.
func (ar *auditRepository) List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*audit.Entry], error) {
	dba, err := ar.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := ar.l.With().
		Str("operation", "List").
		Str("buID", opts.TenantOpts.BuID.String()).
		Str("userID", opts.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*audit.Entry, 0)

	q := dba.NewSelect().Model(&entities)
	q = ar.filterQuery(q, opts)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan audit entries")
		return nil, eris.Wrap(err, "scan audit entries")
	}

	return &ports.ListResult[*audit.Entry]{
		Items: entities,
		Total: total,
	}, nil
}

// ListByResourceID fetches a lists of audit entries by resource id
//
// Parameters:
//   - ctx: The context for the operation.
//   - opts: The options for the operation.
//
// Returns:
//   - A list of audit entries.
//   - An error if the operation fails.
func (ar *auditRepository) ListByResourceID(ctx context.Context, opts repositories.ListByResourceIDRequest) (*ports.ListResult[*audit.Entry], error) {
	dba, err := ar.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := ar.l.With().
		Str("operation", "ListByResourceID").
		Str("resourceID", opts.ResourceID.String()).
		Logger()

	entities := make([]*audit.Entry, 0)

	q := dba.NewSelect().Model(&entities).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("ae.resource_id = ?", opts.ResourceID).
				Where("ae.organization_id = ?", opts.OrgID).
				Where("ae.business_unit_id = ?", opts.BuID)
		}).Relation("User")

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Str("resourceID", opts.ResourceID.String()).Err(err).Msg("failed to scan audit entries")
		return nil, eris.Wrap(err, "scan audit entries")
	}

	return &ports.ListResult[*audit.Entry]{
		Items: entities,
		Total: total,
	}, nil
}

// InsertAuditEntries inserts audit entries into the database.
//
// Parameters:
//   - ctx: The context for the operation.
//   - entries: The audit entries to insert.
//
// Returns:
//   - error: If any database operation fails.
func (ar *auditRepository) InsertAuditEntries(ctx context.Context, entries []*audit.Entry) error {
	dba, err := ar.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}
	log := ar.l.With().
		Str("operation", "InsertAuditEntries").
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		_, err = tx.NewInsert().Model(&entries).Exec(c)
		if err != nil {
			ar.l.Error().Err(err).Msg("failed to insert audit entries")
			return eris.Wrap(err, "failed to insert audit entries")
		}
		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to insert audit entries")
		return err
	}

	return nil
}
