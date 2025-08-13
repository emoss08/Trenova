package ports

import (
	"context"

	"github.com/emoss08/trenova/shared/edi/internal/core/domain"
)

type EDIDocumentRepository interface {
	Create(ctx context.Context, doc *domain.EDIDocument) error
	GetByID(ctx context.Context, id string) (*domain.EDIDocument, error)
	GetByControlNumber(ctx context.Context, partnerID, controlNumber string) (*domain.EDIDocument, error)
	Update(ctx context.Context, doc *domain.EDIDocument) error
	List(ctx context.Context, partnerID string, limit, offset int) ([]*domain.EDIDocument, error)
}

type EDITransactionRepository interface {
	Create(ctx context.Context, tx *domain.EDITransaction) error
	GetByID(ctx context.Context, id string) (*domain.EDITransaction, error)
	GetByDocumentID(ctx context.Context, documentID string) ([]*domain.EDITransaction, error)
	Update(ctx context.Context, tx *domain.EDITransaction) error
}

type EDIShipmentRepository interface {
	Create(ctx context.Context, shipment *domain.EDIShipment) error
	GetByID(ctx context.Context, id string) (*domain.EDIShipment, error)
	GetByShipmentID(ctx context.Context, shipmentID string) (*domain.EDIShipment, error)
	GetByTransactionID(ctx context.Context, transactionID string) (*domain.EDIShipment, error)
	Update(ctx context.Context, shipment *domain.EDIShipment) error
	CreateStop(ctx context.Context, stop *domain.EDIStop) error
}

type EDIAcknowledgmentRepository interface {
	Create(ctx context.Context, ack *domain.EDIAcknowledgment) error
	GetByDocumentID(ctx context.Context, documentID string) ([]*domain.EDIAcknowledgment, error)
	Update(ctx context.Context, ack *domain.EDIAcknowledgment) error
}

type EDIPartnerProfileRepository interface {
	Create(ctx context.Context, profile *domain.EDIPartnerProfile) error
	GetByPartnerID(ctx context.Context, partnerID string) (*domain.EDIPartnerProfile, error)
	Update(ctx context.Context, profile *domain.EDIPartnerProfile) error
	List(ctx context.Context, active bool) ([]*domain.EDIPartnerProfile, error)
	Delete(ctx context.Context, partnerID string) error
}