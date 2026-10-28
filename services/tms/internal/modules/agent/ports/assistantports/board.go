package assistantports

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type BoardFillRequest struct {
	Actor    *permissionports.RequestActor
	ThreadID pulid.ID
	Board    *board.Board
}

type BoardCardRequest struct {
	Thread     GetThreadRequest
	ArtifactID pulid.ID
	Actor      *permissionports.RequestActor
}

type BoardCard struct {
	ArtifactID pulid.ID
	ThreadID   pulid.ID
	Title      string
	Author     string
	Question   string
	AnsweredAt int64
	Filled     *board.Filled
	Markdown   string
}

type BoardFiller interface {
	Check(ctx context.Context, req *BoardFillRequest) error
	Fill(ctx context.Context, req *BoardFillRequest) (*board.Filled, error)
	Card(ctx context.Context, req *BoardCardRequest) (*BoardCard, error)
	Location(ctx context.Context, tenant pagination.TenantInfo) *time.Location
}
