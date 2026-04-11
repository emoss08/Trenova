package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/journalreversal"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetJournalReversalByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListJournalReversalsRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type JournalReversalRepository interface {
	List(ctx context.Context, req *ListJournalReversalsRequest) (*pagination.ListResult[*journalreversal.Reversal], error)
	GetByID(ctx context.Context, req GetJournalReversalByIDRequest) (*journalreversal.Reversal, error)
	Create(ctx context.Context, entity *journalreversal.Reversal) (*journalreversal.Reversal, error)
	Update(ctx context.Context, entity *journalreversal.Reversal) (*journalreversal.Reversal, error)
}
