package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/manualjournal"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListManualJournalRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type ListManualJournalConnectionRequest struct {
	Filter               *pagination.QueryOptions `json:"filter"`
	Cursor               pagination.CursorInfo    `json:"-"`
	ManualJournalColumns []string                 `json:"-"`
}

type GetManualJournalByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ManualJournalRepository interface {
	List(
		ctx context.Context,
		req *ListManualJournalRequest,
	) (*pagination.ListResult[*manualjournal.Request], error)
	ListConnection(
		ctx context.Context,
		req *ListManualJournalConnectionRequest,
	) (*pagination.CursorListResult[*manualjournal.Request], error)
	GetByID(ctx context.Context, req GetManualJournalByIDRequest) (*manualjournal.Request, error)
	Create(ctx context.Context, entity *manualjournal.Request) (*manualjournal.Request, error)
	Update(ctx context.Context, entity *manualjournal.Request) (*manualjournal.Request, error)
}
