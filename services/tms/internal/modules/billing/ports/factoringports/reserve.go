package factoringports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListReserveEntryConnectionRequest struct {
	Filter             *pagination.QueryOptions     `json:"filter"`
	Cursor             pagination.CursorInfo        `json:"-"`
	FactoringCompanyID pulid.ID                     `json:"factoringCompanyId"`
	SubmissionID       pulid.ID                     `json:"submissionId"`
	InvoiceID          pulid.ID                     `json:"invoiceId"`
	Kinds              []factoring.ReserveEntryKind `json:"kinds"`
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
	FactoringCompanyID pulid.ID `json:"factoringCompanyId" bun:"factoring_company_id"`
	ReserveMinor       int64    `json:"reserveMinor"       bun:"reserve_minor"`
	LiabilityMinor     int64    `json:"liabilityMinor"     bun:"liability_minor"`
	AdvancedMinor      int64    `json:"advancedMinor"      bun:"advanced_minor"`
	ReleasedMinor      int64    `json:"releasedMinor"      bun:"released_minor"`
	ChargedBackMinor   int64    `json:"chargedBackMinor"   bun:"charged_back_minor"`
	FeeMinor           int64    `json:"feeMinor"           bun:"fee_minor"`
	EntryCount         int      `json:"entryCount"         bun:"entry_count"`
	LastOccurredOn     *int64   `json:"lastOccurredOn"     bun:"last_occurred_on"`
}

type ReserveEntryRepository interface {
	Create(ctx context.Context, entity *factoring.ReserveEntry) (*factoring.ReserveEntry, error)
	ListConnection(
		ctx context.Context,
		req *ListReserveEntryConnectionRequest,
	) (*pagination.CursorListResult[*factoring.ReserveEntry], error)
	ListForItem(
		ctx context.Context,
		req ListReserveEntriesForItemRequest,
	) ([]*factoring.ReserveEntry, error)
	GetBalance(ctx context.Context, req GetReserveBalanceRequest) (*ReserveBalance, error)
}
