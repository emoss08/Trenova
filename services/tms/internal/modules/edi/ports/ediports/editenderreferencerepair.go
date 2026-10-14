package ediports

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
)

type TenderReferenceCandidate struct {
	MessageID      pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	TransferID     pulid.ID
	RawX12         string
	FileHead       string
	ShipmentID     string
	PaymentMethod  string
}

type ListTenderReferenceCandidatesRequest struct {
	OrganizationID pulid.ID
	AfterID        pulid.ID
	Limit          int
}

type CountUnreadableTendersRequest struct {
	OrganizationID pulid.ID
}

type RepairTenderReferenceRequest struct {
	Candidate     *TenderReferenceCandidate
	ShipmentID    string
	PaymentMethod string
}

type RepairTenderReferenceResult struct {
	Transfers int
	Shipments int
}

type TenderReferenceRepairRepository interface {
	ListTenderReferenceCandidates(
		ctx context.Context,
		req *ListTenderReferenceCandidatesRequest,
	) ([]*TenderReferenceCandidate, error)
	CountUnreadableTenders(
		ctx context.Context,
		req *CountUnreadableTendersRequest,
	) (int, error)
	RepairTenderReference(
		ctx context.Context,
		req *RepairTenderReferenceRequest,
	) (*RepairTenderReferenceResult, error)
}
