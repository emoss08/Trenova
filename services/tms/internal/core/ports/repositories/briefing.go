package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/briefing"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// GetBriefingForDayRequest reads one role's briefing for an organization's
// local day. A nil reader is the role's shared briefing.
type GetBriefingForDayRequest struct {
	TenantInfo   pagination.TenantInfo
	RoleKey      briefing.RoleKey
	UserID       *pulid.ID
	BriefingDate string
}

type GetBriefingByIDRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

// ListBriefingsRequest pages back through the week, newest day first.
type ListBriefingsRequest struct {
	TenantInfo pagination.TenantInfo
	RoleKey    briefing.RoleKey
	UserID     *pulid.ID
	Limit      int
}

// DeleteBriefingsBeforeRequest removes briefings for days before a cut-off,
// a batch at a time across every tenant.
type DeleteBriefingsBeforeRequest struct {
	// BeforeDate is an organization-independent YYYY-MM-DD cut-off: a day
	// string compares correctly as a string, so one cut-off serves every
	// timezone without converting anything.
	BeforeDate string
	Limit      int
}

type BriefingRepository interface {
	// Upsert writes the day's briefing, replacing the one already there for
	// that organization, role, reader and day. A rerun overwrites rather
	// than stacking a second copy.
	Upsert(ctx context.Context, entity *briefing.Briefing) (*briefing.Briefing, error)
	GetByID(ctx context.Context, req GetBriefingByIDRequest) (*briefing.Briefing, error)
	GetForDay(ctx context.Context, req GetBriefingForDayRequest) (*briefing.Briefing, error)
	List(ctx context.Context, req ListBriefingsRequest) ([]*briefing.Briefing, error)
	MarkRead(
		ctx context.Context,
		req GetBriefingByIDRequest,
		readAt int64,
	) (*briefing.Briefing, error)
	MarkEmailed(
		ctx context.Context,
		req GetBriefingByIDRequest,
		emailedAt int64,
	) (*briefing.Briefing, error)
	DeleteBefore(ctx context.Context, req DeleteBriefingsBeforeRequest) (int, error)
}
