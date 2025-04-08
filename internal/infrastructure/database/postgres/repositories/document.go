package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
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

// GetDocumentCountByResource gets the document count for every resource type.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - req: The request options for getting the document count by resource type.
//
// Returns:
//   - []*repositories.GetDocumentCountByResourceResponse: The document count for every resource type.
func (r *documentRepository) GetDocumentCountByResource(ctx context.Context, req *ports.TenantOptions) ([]*repositories.GetDocumentCountByResourceResponse, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetDocumentCountByResource").
		Logger()

	results := make([]*repositories.GetDocumentCountByResourceResponse, 0)

	q := dba.NewSelect().
		ColumnExpr("resource_type").
		ColumnExpr("COUNT(DISTINCT resource_id) as count").
		ColumnExpr("sum(file_size) as total_size").
		ColumnExpr("max(created_at) as last_modified").
		Model((*document.Document)(nil)).
		Where("doc.organization_id = ?", req.OrgID).
		Where("doc.business_unit_id = ?", req.BuID).
		Group("resource_type")

	if err = q.Scan(ctx, &results); err != nil {
		log.Error().Err(err).Msg("failed to get document count by resource")
		return nil, err
	}

	return results, nil
}

func (r *documentRepository) addJoinToResourceTable(q *bun.SelectQuery, req repositories.GetResourceSubFoldersRequest) *bun.SelectQuery {
	//nolint:exhaustive // not all cases are implemented
	switch req.ResourceType {
	case permission.ResourceShipment:
		q = q.Join("LEFT JOIN shipments as s ON s.id = doc.resource_id").
			ColumnExpr("s.pro_number as folder_name").
			Group("s.pro_number")
	case permission.ResourceWorker:
		q = q.Join("LEFT JOIN workers as w ON w.id = doc.resource_id").
			ColumnExpr("w.whole_name as folder_name").
			Group("w.whole_name")
	}

	return q
}

// GetResourceSubFolders gets the sub-folders for a resource
//
// Parameters:
//   - ctx: The context for the operation.
//   - req: The request containing resource details.
//
// Returns:
//   - []*repositories.GetResourceSubFoldersResponse: The sub-folders for a resource.
//   - error: An error if the operation fails.
func (r *documentRepository) GetResourceSubFolders(ctx context.Context, req repositories.GetResourceSubFoldersRequest) ([]*repositories.GetResourceSubFoldersResponse, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetResourceSubFolders").
		Str("resourceType", string(req.ResourceType)).
		Logger()

	results := make([]*repositories.GetResourceSubFoldersResponse, 0)

	q := dba.NewSelect().
		ColumnExpr("COUNT(doc.resource_id) as count").
		ColumnExpr("sum(doc.file_size) as total_size").
		ColumnExpr("max(doc.created_at) as last_modified").
		ColumnExpr("doc.resource_id as resource_id").
		Model((*document.Document)(nil)).
		Where("doc.organization_id = ?", req.OrgID).
		Where("doc.business_unit_id = ?", req.BuID).
		Where("doc.resource_type = ?", req.ResourceType).
		Group("doc.resource_id")

	q = r.addJoinToResourceTable(q, req)

	if err = q.Scan(ctx, &results); err != nil {
		log.Error().Err(err).Msg("failed to get resource sub-folders")
	}

	return results, nil
}

// GetByID gets a document by its ID
//
// Parameters:
//   - ctx: The context for the operation.
//   - req: The request containing document details.
//
// Returns:
//   - *document.Document: The document.
//   - error: An error if the operation fails.
func (r *documentRepository) GetByID(ctx context.Context, req repositories.GetDocumentByIDRequest) (*document.Document, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetDocumentByID").
		Str("docID", req.ID.String()).
		Str("orgID", req.OrgID.String()).
		Str("buID", req.BuID.String()).
		Logger()

	doc := new(document.Document)

	if err = dba.NewSelect().
		Model(doc).
		WherePK().
		Scan(ctx, doc); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Document not found")
		}

		log.Error().Err(err).Msg("failed to get document by ID")
		return nil, err
	}

	return doc, nil
}

func (r *documentRepository) filterResourceQuery(q *bun.SelectQuery, req *repositories.GetDocumentsByResourceIDRequest) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "doc",
		Filter:     req.Filter,
	})

	q = q.Where("doc.resource_id = ?", req.ResourceID).
		Where("doc.resource_type = ?", req.ResourceType)

	q = q.Order("doc.created_at ASC").
		Relation("UploadedBy")

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

// GetDocumentsByResourceID gets documents by resource ID
//
// Parameters:
//   - ctx: The context for the operation.
//   - req: The request containing document details.
//
// Returns:
//   - *ports.ListResult[*document.Document]: The documents.
//   - error: An error if the operation fails.
func (r *documentRepository) GetDocumentsByResourceID(ctx context.Context, req *repositories.GetDocumentsByResourceIDRequest) (*ports.ListResult[*document.Document], error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetDocumentsByResourceID").
		Str("resourceType", string(req.ResourceType)).
		Str("resourceID", req.ResourceID).
		Logger()

	entities := make([]*document.Document, 0)

	q := dba.NewSelect().Model(&entities)
	q = r.filterResourceQuery(q, req)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("No documents found for the given resource ID")
		}

		log.Error().Err(err).Msg("failed to get documents by resource ID")
		return nil, err
	}

	return &ports.ListResult[*document.Document]{
		Items: entities,
		Total: total,
	}, nil
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

// Update updates a document in the database
//
// Parameters:
//   - ctx: The context for the operation.
//   - doc: The document entity to be updated.
//
// Returns:
//   - *document.Document: The updated document.
//   - error: An error if the operation fails.
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

	//nolint:dupl // Service code is similar to each other
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

// Delete deletes a document from the database
//
// Parameters:
//   - ctx: The context for the operation.
//   - req: The request containing document details.
//
// Returns:
//   - error: An error if the operation fails.
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
