package repositories

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/shared/edi/internal/core/domain"
	"github.com/emoss08/trenova/shared/edi/internal/core/ports"
	"github.com/emoss08/trenova/shared/edi/internal/infrastructure/database"
	"go.uber.org/fx"
)

type ediShipmentRepository struct {
	db *database.DB
}

type EDIShipmentRepoParams struct {
	fx.In
	DB *database.DB
}

func NewEDIShipmentRepository(params EDIShipmentRepoParams) ports.EDIShipmentRepository {
	return &ediShipmentRepository{
		db: params.DB,
	}
}

func (r *ediShipmentRepository) Create(ctx context.Context, shipment *domain.EDIShipment) error {
	_, err := r.db.NewInsert().Model(shipment).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create EDI shipment: %w", err)
	}
	return nil
}

func (r *ediShipmentRepository) GetByID(ctx context.Context, id string) (*domain.EDIShipment, error) {
	shipment := new(domain.EDIShipment)
	err := r.db.NewSelect().
		Model(shipment).
		Where("id = ?", id).
		Relation("Transaction").
		Relation("Stops").
		Scan(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get EDI shipment by ID: %w", err)
	}
	return shipment, nil
}

func (r *ediShipmentRepository) GetByShipmentID(ctx context.Context, shipmentID string) (*domain.EDIShipment, error) {
	shipment := new(domain.EDIShipment)
	err := r.db.NewSelect().
		Model(shipment).
		Where("shipment_id = ?", shipmentID).
		Relation("Stops").
		Scan(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get EDI shipment by shipment ID: %w", err)
	}
	return shipment, nil
}

func (r *ediShipmentRepository) GetByTransactionID(ctx context.Context, transactionID string) (*domain.EDIShipment, error) {
	shipment := new(domain.EDIShipment)
	err := r.db.NewSelect().
		Model(shipment).
		Where("transaction_id = ?", transactionID).
		Relation("Stops").
		Scan(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get EDI shipment by transaction ID: %w", err)
	}
	return shipment, nil
}

func (r *ediShipmentRepository) Update(ctx context.Context, shipment *domain.EDIShipment) error {
	_, err := r.db.NewUpdate().
		Model(shipment).
		WherePK().
		Exec(ctx)
	
	if err != nil {
		return fmt.Errorf("failed to update EDI shipment: %w", err)
	}
	return nil
}

func (r *ediShipmentRepository) CreateStop(ctx context.Context, stop *domain.EDIStop) error {
	_, err := r.db.NewInsert().Model(stop).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create EDI stop: %w", err)
	}
	return nil
}