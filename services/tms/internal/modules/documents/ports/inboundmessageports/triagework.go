package inboundmessageports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type TriageShipmentPeeker interface {
	ShipmentPeeks(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentIDs []pulid.ID,
	) (map[pulid.ID]*ShipmentPeek, error)
	CustomerShipments(
		ctx context.Context,
		req *TriageCustomerShipmentsRequest,
	) (*TriageCustomerShipments, error)
}

type TriageCustomerShipmentsRequest struct {
	TenantInfo pagination.TenantInfo
	CustomerID pulid.ID
	WeekStart  int64
	Limit      int
}

type TriageCustomerShipments struct {
	ShipmentIDs []pulid.ID
	ThisWeek    int
}

type TriageStopTimesReader interface {
	StopTimes(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	) (*StopTimes, error)
}

type TriageLoadFieldValue struct {
	Index int
	Key   inboundmessage.LoadFieldKey
	Value string
}

type CreateTriageLoadRequest struct {
	TenantInfo       pagination.TenantInfo
	Actor            *permissionports.RequestActor
	CustomerID       pulid.ID
	Fields           []TriageLoadFieldValue
	SourceDocumentID pulid.ID
	IdempotencyKey   string
}

type CreatedTriageLoad struct {
	ShipmentID pulid.ID
	ProNumber  string
}

type TriageLoadCreator interface {
	CreateLoad(ctx context.Context, req *CreateTriageLoadRequest) (*CreatedTriageLoad, error)
}

type LogTriageExceptionRequest struct {
	TenantInfo pagination.TenantInfo
	Actor      *permissionports.RequestActor
	ShipmentID pulid.ID
	Kind       inboundmessage.ExceptionKind
	Quantity   int
	Unit       string
	Note       string
	Source     string
}

type TriageExceptionLogger interface {
	LogException(ctx context.Context, req *LogTriageExceptionRequest) error
}

type SettleTriageDetentionRequest struct {
	TenantInfo pagination.TenantInfo
	ShipmentID pulid.ID
	Decision   inboundmessage.StopsDecision
	Note       string
	UserID     pulid.ID
}

type SettledTriageDetention struct {
	Amount   string
	Currency string
	Settled  bool
}

type TriageDetentionSettler interface {
	SettleDetention(
		ctx context.Context,
		req *SettleTriageDetentionRequest,
	) (*SettledTriageDetention, error)
}

type QueueCarrierInvoiceRequest struct {
	TenantInfo             pagination.TenantInfo
	Actor                  *permissionports.RequestActor
	ShipmentID             pulid.ID
	CarrierID              pulid.ID
	DocumentAIExtractionID pulid.ID
	InvoiceNumber          string
	TotalMinor             int64
}

type TriageCarrierPayQueuer interface {
	QueueCarrierInvoice(ctx context.Context, req *QueueCarrierInvoiceRequest) error
}

type AlternateAddressRequest struct {
	TenantInfo pagination.TenantInfo
	Domain     string
	Exclude    string
	Since      int64
	Limit      int
}

type CustomerTriageRequest struct {
	TenantInfo pagination.TenantInfo
	CustomerID pulid.ID
	Limit      int
}

type CustomerTriage struct {
	Waiting int
	Recent  []*Sample
}

type TriageWorkRepository interface {
	ContentsByDocumentIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		documentIDs []pulid.ID,
	) (map[pulid.ID]*documentcontent.Content, error)
	DocumentTypesOnShipments(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentIDs []pulid.ID,
	) (map[pulid.ID][]string, error)
	CurrentShipmentDocuments(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
		documentTypeID pulid.ID,
	) ([]pulid.ID, error)
	LatestExtractionID(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		documentID pulid.ID,
	) (pulid.ID, error)
	AlternateAddresses(ctx context.Context, req *AlternateAddressRequest) ([]string, error)
	CustomerTriage(ctx context.Context, req *CustomerTriageRequest) (*CustomerTriage, error)
}

type GetInboundAttachmentRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type GetInboundAttachmentsRequest struct {
	IDs        []pulid.ID
	TenantInfo pagination.TenantInfo
}
