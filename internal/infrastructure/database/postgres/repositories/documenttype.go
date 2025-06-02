package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/billing"
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

// DocumentTypeRepositoryParams defines dependencies required for initializing the DocumentTypeRepository.
// This includes database connection, logger, and document type repository.
type DocumentTypeRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// documentTypeRepository implements the DocumentTypeRepository interface
// and provides methods to manage document types, including CRUD operations.
type documentTypeRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewDocumentTypeRepository initializes a new instance of documentTypeRepository with its dependencies.
//
// Parameters:
//   - p: DocumentTypeRepositoryParams containing dependencies.
//
// Returns:
//   - repositories.DocumentTypeRepository: A ready-to-use document type repository instance.
func NewDocumentTypeRepository(p DocumentTypeRepositoryParams) repositories.DocumentTypeRepository {
	log := p.Logger.With().
		Str("repository", "fleetcode").
		Logger()

	return &documentTypeRepository{
		db: p.DB,
		l:  &log,
	}
}

// filterQuery applies filters and pagination to the document type query.
// It includes tenant-based filtering and full-text search when provided.
//
// Parameters:
//   - q: The base select query.
//   - opts: ListDocumentTypeOptions containing filter and pagination details.
//
// Returns:
//   - *bun.SelectQuery: The filtered and paginated query.
func (dt *documentTypeRepository) filterQuery(
	q *bun.SelectQuery,
	opts *ports.LimitOffsetQueryOptions,
) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "dt",
		Filter:     opts,
	})

	if opts.Query != "" {
		q = postgressearch.BuildSearchQuery(
			q,
			opts.Query,
			(*billing.DocumentType)(nil),
		)
	}

	return q.Limit(opts.Limit).Offset(opts.Offset)
}

// List retrieves a list of document types with optional filtering and pagination.
//
// Parameters:
//   - ctx: The context for the operation.
//   - opts: The options for the list operation, including filter and pagination details.
//
// Returns:
//   - *ports.ListResult[*billing.DocumentType]: A list of document types with pagination information.
//   - error: An error if the operation fails.
func (dt *documentTypeRepository) List(
	ctx context.Context,
	opts *ports.LimitOffsetQueryOptions,
) (*ports.ListResult[*billing.DocumentType], error) {
	dba, err := dt.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	log := dt.l.With().
		Str("operation", "List").
		Str("buID", opts.TenantOpts.BuID.String()).
		Str("userID", opts.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*billing.DocumentType, 0)

	q := dba.NewSelect().Model(&entities)
	q = dt.filterQuery(q, opts)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan document types")
		return nil, err
	}

	return &ports.ListResult[*billing.DocumentType]{
		Items: entities,
		Total: total,
	}, nil
}

// GetByID retrieves a document type by its ID.
//
// Parameters:
//   - ctx: The context for the operation.
//   - opts: The options for the get operation, including ID, organization ID, business unit ID, and user ID.
//
// Returns:
//   - *billing.DocumentType: The document type if found.
//   - error: An error if the operation fails.
func (dt *documentTypeRepository) GetByID(
	ctx context.Context,
	opts repositories.GetDocumentTypeByIDRequest,
) (*billing.DocumentType, error) {
	dba, err := dt.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	log := dt.l.With().
		Str("operation", "GetByID").
		Str("documentTypeID", opts.ID.String()).
		Logger()

	entity := new(billing.DocumentType)

	query := dba.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("dt.id = ?", opts.ID).
				Where("dt.organization_id = ?", opts.OrgID).
				Where("dt.business_unit_id = ?", opts.BuID)
		})

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Document type not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get document type")
		return nil, err
	}

	return entity, nil
}

// GetByIDs retrieves a list of document types by their IDs.
//
// Parameters:
//   - ctx: The context for the operation.
//   - docIDs: A slice of document IDs to retrieve.
//
// Returns:
//   - []*billing.DocumentType: A list of document types.
//   - error: An error if the operation fails.
func (dt *documentTypeRepository) GetByIDs(
	ctx context.Context,
	docIDs []string,
) ([]*billing.DocumentType, error) {
	dba, err := dt.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	log := dt.l.With().
		Str("operation", "GetByIDs").
		Logger()

	// Create an empty slice with the capacity to hold all potential results
	entities := make([]*billing.DocumentType, 0, len(docIDs))

	query := dba.NewSelect().Model(&entities).
		Where("dt.id IN (?)", bun.In(docIDs))

	if err = query.Scan(ctx); err != nil {
		log.Error().Err(err).Msg("failed to get document types")
		return nil, err
	}

	return entities, nil
}

// Create inserts a new document type into the database.
//
// Parameters:
//   - ctx: The context for the operation.
//   - entity: The document type to be created.
//
// Returns:
//   - *billing.DocumentType: The created document type.
//   - error: An error if the operation fails.
func (dt *documentTypeRepository) Create(
	ctx context.Context,
	entity *billing.DocumentType,
) (*billing.DocumentType, error) {
	dba, err := dt.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := dt.l.With().
		Str("operation", "Create").
		Str("orgID", entity.OrganizationID.String()).
		Str("buID", entity.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(entity).Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("documentType", entity).
				Msg("failed to insert document type")
			return iErr
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return entity, nil
}

// Update updates an existing document type in the database.
//
// Parameters:
//   - ctx: The context for the operation.
//   - entity: The document type to be updated.
//
// Returns:
//   - *billing.DocumentType: The updated document type.
//   - error: An error if the operation fails.
func (dt *documentTypeRepository) Update(
	ctx context.Context,
	entity *billing.DocumentType,
) (*billing.DocumentType, error) {
	dba, err := dt.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := dt.l.With().
		Str("operation", "Update").
		Str("id", entity.GetID()).
		Int64("version", entity.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := entity.Version

		entity.Version++

		results, rErr := tx.NewUpdate().
			Model(entity).
			WherePK().
			OmitZero().
			Where("dt.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("documentType", entity).
				Msg("failed to update document type")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("documentType", entity).
				Msg("failed to get rows affected")
			return roErr
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The Document Type (%s) has either been updated or deleted since the last request.",
					entity.ID.String(),
				),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update document type")
		return nil, err
	}

	return entity, nil
}
