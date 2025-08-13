package repositories

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/shared/edi/internal/core/domain"
	"github.com/emoss08/trenova/shared/edi/internal/core/ports"
	"github.com/emoss08/trenova/shared/edi/internal/infrastructure/database"
	"go.uber.org/fx"
)

type ediDocumentRepository struct {
	db *database.DB
}

type EDIDocumentRepoParams struct {
	fx.In
	DB *database.DB
}

func NewEDIDocumentRepository(params EDIDocumentRepoParams) ports.EDIDocumentRepository {
	return &ediDocumentRepository{
		db: params.DB,
	}
}

func (r *ediDocumentRepository) Create(ctx context.Context, doc *domain.EDIDocument) error {
	_, err := r.db.NewInsert().Model(doc).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create EDI document: %w", err)
	}
	return nil
}

func (r *ediDocumentRepository) GetByID(ctx context.Context, id string) (*domain.EDIDocument, error) {
	doc := new(domain.EDIDocument)
	err := r.db.NewSelect().
		Model(doc).
		Where("id = ?", id).
		Relation("Transactions").
		Relation("Acknowledgments").
		Scan(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get EDI document by ID: %w", err)
	}
	return doc, nil
}

func (r *ediDocumentRepository) GetByControlNumber(ctx context.Context, partnerID, controlNumber string) (*domain.EDIDocument, error) {
	doc := new(domain.EDIDocument)
	err := r.db.NewSelect().
		Model(doc).
		Where("partner_id = ? AND control_number = ?", partnerID, controlNumber).
		Scan(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get EDI document by control number: %w", err)
	}
	return doc, nil
}

func (r *ediDocumentRepository) Update(ctx context.Context, doc *domain.EDIDocument) error {
	_, err := r.db.NewUpdate().
		Model(doc).
		WherePK().
		Exec(ctx)
	
	if err != nil {
		return fmt.Errorf("failed to update EDI document: %w", err)
	}
	return nil
}

func (r *ediDocumentRepository) List(ctx context.Context, partnerID string, limit, offset int) ([]*domain.EDIDocument, error) {
	var docs []*domain.EDIDocument
	
	query := r.db.NewSelect().Model(&docs).OrderExpr("created_at DESC")
	
	if partnerID != "" {
		query = query.Where("partner_id = ?", partnerID)
	}
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	err := query.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list EDI documents: %w", err)
	}
	
	return docs, nil
}