package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/deskcase"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListCaseRecordsRequest struct {
	TenantInfo pagination.TenantInfo
	Refs       []deskcase.Ref
}

type GetShipmentCaseFactsRequest struct {
	TenantInfo pagination.TenantInfo
	ShipmentID pulid.ID
	// PODCode is the document type code a proof of delivery is filed under.
	PODCode string
}

type GetInvoiceCaseFactsRequest struct {
	TenantInfo pagination.TenantInfo
	InvoiceID  pulid.ID
}

type ListCasePartiesRequest struct {
	TenantInfo pagination.TenantInfo
	CustomerID pulid.ID
	CarrierIDs []pulid.ID
}

// AssistantCaseRepository reads what a Desk case shows about the record it
// is about, in as few queries as the rail and the case header need: one per
// kind of record for the rail, one for a shipment's checklist.
type AssistantCaseRepository interface {
	// ListRecords reads the records the cases name, keyed by id. A record
	// that is gone, or is not the organization's, is left out.
	ListRecords(
		ctx context.Context,
		req *ListCaseRecordsRequest,
	) (map[pulid.ID]*deskcase.Record, error)
	// ShipmentFacts reads what the ready-to-bill checklist needs beyond
	// billing readiness: the proof of delivery on file, the carriers' rate
	// confirmations, the open charge findings, the detention clocks, the
	// last emailed customer update and whether billing has the shipment.
	ShipmentFacts(
		ctx context.Context,
		req *GetShipmentCaseFactsRequest,
	) (*deskcase.ShipmentFacts, error)
	InvoiceFacts(ctx context.Context, req *GetInvoiceCaseFactsRequest) (*deskcase.InvoiceFacts, error)
	// Parties names the customer and carriers a case can wait on.
	Parties(ctx context.Context, req *ListCasePartiesRequest) ([]deskcase.Party, error)
}
