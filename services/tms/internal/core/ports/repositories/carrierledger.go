package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListCarrierLedgerEntriesRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	CarrierID  pulid.ID              `json:"carrierId"`
	Limit      int                   `json:"limit"`
}

type CarrierLedgerRepository interface {
	AppendEntries(ctx context.Context, entries []*carriersettlement.LedgerEntry) error
	ListByCarrier(
		ctx context.Context,
		req *ListCarrierLedgerEntriesRequest,
	) ([]*carriersettlement.LedgerEntry, error)
}
