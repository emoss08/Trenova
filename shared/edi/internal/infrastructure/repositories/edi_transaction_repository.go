package repositories

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/shared/edi/internal/core/domain"
	"github.com/emoss08/trenova/shared/edi/internal/core/ports"
	"github.com/emoss08/trenova/shared/edi/internal/infrastructure/database"
	"go.uber.org/fx"
)

type ediTransactionRepository struct {
	db *database.DB
}

type EDITransactionRepoParams struct {
	fx.In
	DB *database.DB
}

func NewEDITransactionRepository(params EDITransactionRepoParams) ports.EDITransactionRepository {
	return &ediTransactionRepository{
		db: params.DB,
	}
}

func (r *ediTransactionRepository) Create(ctx context.Context, tx *domain.EDITransaction) error {
	_, err := r.db.NewInsert().Model(tx).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create EDI transaction: %w", err)
	}
	return nil
}

func (r *ediTransactionRepository) GetByID(ctx context.Context, id string) (*domain.EDITransaction, error) {
	tx := new(domain.EDITransaction)
	err := r.db.NewSelect().
		Model(tx).
		Where("id = ?", id).
		Relation("Document").
		Relation("Shipment").
		Scan(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get EDI transaction by ID: %w", err)
	}
	return tx, nil
}

func (r *ediTransactionRepository) GetByDocumentID(ctx context.Context, documentID string) ([]*domain.EDITransaction, error) {
	var txs []*domain.EDITransaction
	err := r.db.NewSelect().
		Model(&txs).
		Where("document_id = ?", documentID).
		OrderExpr("created_at ASC").
		Scan(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get EDI transactions by document ID: %w", err)
	}
	return txs, nil
}

func (r *ediTransactionRepository) Update(ctx context.Context, tx *domain.EDITransaction) error {
	_, err := r.db.NewUpdate().
		Model(tx).
		WherePK().
		Exec(ctx)
	
	if err != nil {
		return fmt.Errorf("failed to update EDI transaction: %w", err)
	}
	return nil
}