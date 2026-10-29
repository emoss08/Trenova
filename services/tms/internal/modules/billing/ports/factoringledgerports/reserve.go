package factoringledgerports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringledger"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListReserveEntryConnectionRequest struct {
	Filter             *pagination.QueryOptions           `json:"filter"`
	Cursor             pagination.CursorInfo              `json:"-"`
	FactoringCompanyID pulid.ID                           `json:"factoringCompanyId"`
	SubmissionID       pulid.ID                           `json:"submissionId"`
	InvoiceID          pulid.ID                           `json:"invoiceId"`
	Kinds              []factoringledger.ReserveEntryKind `json:"kinds"`
}

type GetReserveBalanceRequest struct {
	TenantInfo         pagination.TenantInfo `json:"-"`
	FactoringCompanyID pulid.ID              `json:"factoringCompanyId"`
}

type ListReserveEntriesForItemRequest struct {
	TenantInfo       pagination.TenantInfo `json:"-"`
	SubmissionItemID pulid.ID              `json:"submissionItemId"`
}

type ReserveBalance struct {
	FactoringCompanyID pulid.ID `json:"factoringCompanyId"`
	ReserveMinor       int64    `json:"reserveMinor"`
	LiabilityMinor     int64    `json:"liabilityMinor"`
	AdvancedMinor      int64    `json:"advancedMinor"`
	ReleasedMinor      int64    `json:"releasedMinor"`
	ChargedBackMinor   int64    `json:"chargedBackMinor"`
	FeeMinor           int64    `json:"feeMinor"`
	EntryCount         int      `json:"entryCount"`
	LastOccurredOn     *int64   `json:"lastOccurredOn"`
}

type ReserveEntryRepository interface {
	Create(
		ctx context.Context,
		entity *factoringledger.ReserveEntry,
	) (*factoringledger.ReserveEntry, error)
	ListConnection(
		ctx context.Context,
		req *ListReserveEntryConnectionRequest,
	) (*pagination.CursorListResult[*factoringledger.ReserveEntry], error)
	ListForItem(
		ctx context.Context,
		req ListReserveEntriesForItemRequest,
	) ([]*factoringledger.ReserveEntry, error)
	GetBalance(ctx context.Context, req GetReserveBalanceRequest) (*ReserveBalance, error)
}
