package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/document"
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

// DocumentRepositoryParams defines dependencies required for initializing the DocumentRepository.
// This includes database connection, logger, and document repository.
type DocumentRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// documentRepository implements the DocumentRepository interface
// and provides methods to manage documents, including CRUD operations,
// status updates, and aggregation.
type documentRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewDocumentRepository initializes a new instance of documentRepository with its dependencies.
//
// Parameters:
//   - p: DocumentRepositoryParams containing dependencies.
//
// Returns:
//   - repositories.DocumentRepository: A ready-to-use document repository instance.
func NewDocumentRepository(p DocumentRepositoryParams) repositories.DocumentRepository {
	log := p.Logger.With().
		Str("repository", "document").
		Logger()

	return &documentRepository{db: p.DB, l: &log}
}

// addOptions adds options to the query
//
// Parameters:
//   - q: The query to add options to.
//   - req: The request options for adding options to the query.
//
// Returns:
//   - *bun.SelectQuery: The query with added options.
func (r *documentRepository) addOptions(q *bun.SelectQuery, req repositories.DocumentRequest) *bun.SelectQuery {
	if req.ExpandDocumentDetails {
		q = q.Relation("UploadedBy")
		q = q.Relation("ApprovedBy")
	}

	return q
}

// filterQuery filters the query based on the request options
//
// Parameters:
//   - q: The query to filter.
//   - req: The request options for filtering the query.
//
// Returns:
//   - *bun.SelectQuery: The filtered query.
func (r *documentRepository) filterQuery(q *bun.SelectQuery, req *repositories.ListDocumentsRequest) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "doc",
		Filter:     req.Filter,
	})

	if req.Filter.Query != "" {
		q = postgressearch.BuildSearchQuery(
			q,
			req.Filter.Query,
			(*document.Document)(nil),
		)
	}

	// * Filter by entity type and ID if provided
	if req.ResourceType != "" {
		q = q.Where("doc.resource_type = ?", req.ResourceType)
	}

	if req.ResourceID.IsNotNil() {
		q = q.Where("doc.resource_id = ?", req.ResourceID)
	}

	// * Filter by document type if provided
	if req.DocumentType != "" {
		q = q.Where("doc.document_type = ?", req.DocumentType)
	}

	// * filter by status if provided
	if len(req.Statuses) > 0 {
		q = q.Where("doc.status IN (?)", bun.In(req.Statuses))
	}

	// * Filter by tags if provided
	if len(req.Tags) > 0 {
		q = q.Where("doc.tags && ?", req.Tags)
	}

	// * Filter by expiration date if provided
	if req.ExpirationDateStart != nil {
		q = q.Where("doc.expiration_date >= ?", req.ExpirationDateStart)
	}

	if req.ExpirationDateEnd != nil {
		q = q.Where("doc.expiration_date <= ?", req.ExpirationDateEnd)
	}

	// * Filter by creation date range if provided
	if req.CreatedAtStart != nil {
		q = q.Where("doc.created_at >= ?", req.CreatedAtStart)
	}

	if req.CreatedAtEnd != nil {
		q = q.Where("doc.created_at <= ?", req.CreatedAtEnd)
	}

	q = r.addOptions(q, req.DocumentRequest)

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

// List lists documents
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - req: The request options for listing documents.
//
// Returns:
//   - *ports.ListResult[*document.Document]: The list of documents.
//   - error: If the operation fails.
func (r *documentRepository) List(ctx context.Context, req *repositories.ListDocumentsRequest) (*ports.ListResult[*document.Document], error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "List").
		Str("buID", req.Filter.TenantOpts.BuID.String()).
		Str("orgID", req.Filter.TenantOpts.OrgID.String()).
		Logger()

	entities := make([]*document.Document, 0)

	q := dba.NewSelect().Model(&entities)
	q = r.filterQuery(q, req)

	// Sort by creation date descending by default
	if req.SortBy == "" {
		q.Order("doc.created_at DESC")
	} else {
		q.Order(fmt.Sprintf("doc.%s %s", req.SortBy, req.SortDir))
	}

	total, err := q.ScanAndCount(ctx, &entities)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan and count documents")
		return nil, eris.Wrap(err, "scan and count")
	}

	return &ports.ListResult[*document.Document]{
		Items: entities,
		Total: total,
	}, nil
}

// GetByID gets a document by ID
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - req: The request options for getting a document by ID.
//
// Returns:
//   - *document.Document: The document found.
//   - error: If the operation fails.
func (r *documentRepository) GetByID(ctx context.Context, req repositories.GetDocumentByIDOptions) (*document.Document, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetByID").
		Str("documentID", req.ID.String()).
		Logger()

	doc := new(document.Document)

	q := dba.NewSelect().Model(doc).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("doc.id = ?", req.ID).
				Where("doc.organization_id = ?", req.OrgID).
				Where("doc.business_unit_id = ?", req.BuID)
		})

	q = r.addOptions(q, req.DocumentRequest)

	if err = q.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Document not found within your organization")
		}

		log.Error().Err(err).Msg("failed to scan document")
		return nil, err
	}

	return doc, nil
}

// FindByResourceID finds documents by resource ID
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - req: The request options for finding documents by resource ID.
//
// Returns:
//   - []*document.Document: The documents found.
//   - error: If the operation fails.
func (r *documentRepository) FindByResourceID(ctx context.Context, req *repositories.FindDocumentsByResourceRequest) ([]*document.Document, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "FindByResourceID").
		Str("resourceID", req.ResourceID.String()).
		Str("resourceType", string(req.ResourceType)).
		Logger()

	entities := make([]*document.Document, 0)

	q := dba.NewSelect().Model(&entities).
		Where("doc.resource_id = ?", req.ResourceID).
		Where("doc.entity_type = ?", req.ResourceType).
		Where("doc.organization_id = ?", req.OrgID).
		Where("doc.business_unit_id = ?", req.BuID)

		// * Filter by document type if provided
	if req.DocumentType != "" {
		q = q.Where("doc.document_type = ?", req.DocumentType)
	}

	// * Filter by status if provided
	if len(req.Statuses) > 0 {
		q = q.Where("doc.status IN (?)", bun.In(req.Statuses))
	}

	q = r.addOptions(q, req.DocumentRequest)

	// * Sort by creation date descending by default
	q = q.Order("doc.created_at DESC")

	if err = q.Scan(ctx, &entities); err != nil {
		log.Error().Err(err).Msg("failed to scan documents")
		return nil, err
	}

	return entities, nil
}

// Create inserts a new document into the database
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - doc: The document entity to be created.
//
// Returns:
//   - *document.Document: The created document.
//   - error: If insertion fails.
func (r *documentRepository) Create(ctx context.Context, doc *document.Document) (*document.Document, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Create").
		Str("orgID", doc.OrganizationID.String()).
		Str("buID", doc.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(doc).Returning("*").Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("document", doc).
				Msg("failed to insert document")
			return iErr
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create document")
		return nil, err
	}

	return doc, nil
}

func (r *documentRepository) Update(ctx context.Context, doc *document.Document) (*document.Document, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Update").
		Str("id", doc.GetID()).
		Int64("version", doc.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := doc.Version

		doc.Version++

		results, rErr := tx.NewUpdate().
			Model(doc).
			WherePK().
			Where("doc.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("document", doc).
				Msg("failed to update document")
			return eris.Wrap(rErr, "update document")
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("document", doc).
				Msg("failed to get rows affected")
			return eris.Wrap(roErr, "get affected rows")
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The Document (%s) has either been updated or deleted since the last request.", doc.GetID()),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update document")
		return nil, err
	}

	return doc, nil
}

func (r *documentRepository) Delete(ctx context.Context, req repositories.DeleteDocumentRequest) error {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Delete").
		Str("documentID", req.ID.String()).
		Str("orgID", req.OrgID.String()).
		Str("buID", req.BuID.String()).
		Logger()

	result, err := dba.NewDelete().
		Model((*document.Document)(nil)).
		Where("id = ?", req.ID).
		Where("organization_id = ?", req.OrgID).
		Where("business_unit_id = ?", req.BuID).
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete document")
		return eris.Wrap(err, "delete document")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return eris.Wrap(err, "get affected rows")
	}

	if rows == 0 {
		return errors.NewNotFoundError("Document not found or already deleted")
	}

	return nil
}

func (r *documentRepository) FindExpiringDocuments(ctx context.Context, req repositories.FindExpiringDocumentsRequest) ([]*document.Document, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "FindExpiringDocuments").
		Int64("threshold", req.ExpirationThreshold).
		Logger()

	entities := make([]*document.Document, 0)

	q := dba.NewSelect().Model(&entities).
		Where("doc.expiration_date IS NOT NULL").
		Where("doc.expiration_date <= ?", req.ExpirationThreshold).
		Where("doc.expiration_date > ?", 0). // Ensure it's not already expired
		Where("doc.status != ?", document.DocumentStatusExpired)

	// * Filter by organization if provided
	if !req.OrgID.IsNil() {
		q = q.Where("doc.organization_id = ?", req.OrgID)
	}

	// * Filter by business unit if provided
	if !req.BuID.IsNil() {
		q = q.Where("doc.business_unit_id = ?", req.BuID)
	}

	q = q.Order("doc.expiration_date ASC")

	if err = q.Scan(ctx); err != nil {
		log.Error().Err(err).Msg("failed to find expiring documents")
		return nil, err
	}

	return entities, nil
}

func (r *documentRepository) UpdateStatus(ctx context.Context, req repositories.UpdateDocumentStatusRequest) (*document.Document, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "UpdateStatus").
		Str("documentID", req.ID.String()).
		Str("status", string(req.Status)).
		Logger()

	// * Get the document first
	doc, err := r.GetByID(ctx, repositories.GetDocumentByIDOptions{
		ID:    req.ID,
		OrgID: req.OrgID,
		BuID:  req.BuID,
	})
	if err != nil {
		return nil, err
	}

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// * Update the document version
		ov := doc.Version
		doc.Version++

		results, rErr := tx.NewUpdate().Model(doc).
			WherePK().
			Where("doc.version = ?", ov).
			Set("status = ?", req.Status).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().Err(rErr).
				Interface("document", doc).
				Msg("failed to update document status")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).
				Interface("document", doc).
				Msg("failed to get rows affected")
			return roErr
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The document (%s) has been updated since your last request.", doc.GetID()),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).
			Interface("document", doc).
			Msg("failed to update document status")
		return nil, err
	}

	return doc, nil
}

func (r *documentRepository) FindByTags(ctx context.Context, req repositories.FindDocumentsByTagsRequest) ([]*document.Document, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "FindByTags").
		Interface("tags", req.Tags).
		Logger()

	entities := make([]*document.Document, 0)

	q := dba.NewSelect().Model(&entities).
		Where("doc.tags && ?", req.Tags)

	// * Filter by organization and business unit
	q = q.Where("doc.organization_id = ?", req.OrgID)
	q = q.Where("doc.business_unit_id = ?", req.BuID)

	// * Filter by document type if provided
	if req.DocumentType != "" {
		q = q.Where("doc.document_type = ?", req.DocumentType)
	}

	// * Filter by status if provided
	if len(req.Statuses) > 0 {
		q = q.Where("doc.status IN (?)", bun.In(req.Statuses))
	}

	// * Apply options
	q = r.addOptions(q, req.DocumentRequest)

	if err = q.Scan(ctx); err != nil {
		log.Error().Err(err).Msg("failed to find documents by tags")
		return nil, err
	}

	return entities, nil
}

func (r *documentRepository) FindByDocumentType(ctx context.Context, req repositories.FindDocumentsByTypeRequest) ([]*document.Document, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "FindByDocumentType").
		Str("documentType", string(req.DocumentType)).
		Logger()

	entities := make([]*document.Document, 0)

	q := dba.NewSelect().Model(&entities).
		Where("doc.document_type = ?", req.DocumentType).
		Where("doc.organization_id = ?", req.OrgID).
		Where("doc.business_unit_id = ?", req.BuID)

	// * Filter by entity type if provided
	if req.ResourceType != "" {
		q = q.Where("doc.resource_type = ?", req.ResourceType)
	}

	// * Filter by status if provided
	if len(req.Statuses) > 0 {
		q = q.Where("doc.status IN (?)", bun.In(req.Statuses))
	}

	q = r.addOptions(q, req.DocumentRequest)

	if err = q.Scan(ctx); err != nil {
		log.Error().Err(err).Msg("failed to find documents by type")
		return nil, err
	}

	return entities, nil
}

func (r *documentRepository) BulkUpdateStatus(ctx context.Context, req repositories.BulkUpdateDocumentStatusRequest) (int, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return 0, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "BulkUpdateStatus").
		Interface("documentIDs", req.IDs).
		Str("status", string(req.Status)).
		Logger()

	result, err := dba.NewUpdate().
		Model((*document.Document)(nil)).
		Set("status = ?", req.Status).
		Set("updated_at = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint").
		Set("version = version + 1").
		Where("id IN (?)", bun.In(req.IDs)).
		Where("organization_id = ?", req.OrgID).
		Where("business_unit_id = ?", req.BuID).
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to bulk update document status")
		return 0, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return 0, err
	}

	return int(rows), nil
}

func (r *documentRepository) CountDocuments(ctx context.Context, req repositories.CountDocumentsRequest) (map[document.DocumentType]int, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "CountDocuments").
		Str("entityType", string(req.ResourceType)).
		Logger()

	type countResult struct {
		DocumentType document.DocumentType `bun:"document_type"`
		Count        int                   `bun:"count"`
	}

	var results []countResult

	q := dba.NewSelect().
		Column("document_type").
		ColumnExpr("count(*) as count").
		Model((*document.Document)(nil)).
		Where("doc.organization_id = ?", req.OrgID).
		Where("doc.business_unit_id = ?", req.BuID).
		Group("document_type")

	// * Filter by entity type and ID if provided
	if req.ResourceType != "" {
		q = q.Where("doc.resource_type = ?", req.ResourceType)
	}

	if !req.ResourceID.IsNil() {
		q = q.Where("doc.resource_id = ?", req.ResourceID)
	}

	// * Filter by status if provided
	if len(req.Statuses) > 0 {
		q = q.Where("doc.status IN (?)", bun.In(req.Statuses))
	}

	if err = q.Scan(ctx, &results); err != nil {
		log.Error().Err(err).Msg("failed to count documents")
		return nil, err
	}

	// * Convert to map
	countsMap := make(map[document.DocumentType]int)
	for _, result := range results {
		countsMap[result.DocumentType] = result.Count
	}

	return countsMap, nil
}
