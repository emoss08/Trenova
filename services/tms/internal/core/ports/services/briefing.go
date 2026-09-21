package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/briefing"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// WriteBriefingRequest writes one organization's morning. Roles empty
// writes every role, which is what the nightly job does.
type WriteBriefingRequest struct {
	TenantInfo pagination.TenantInfo
	Roles      []briefing.RoleKey
	// Now anchors the whole sweep to one instant, so every role's briefing
	// for a day describes exactly the same window.
	Now int64
	// BriefingDate overrides the organization's current local day, for a
	// rerun of a day that has already turned.
	BriefingDate string
	// SkipNarration writes the computed page without spending a model
	// call, which is what a backfill of old days wants.
	SkipNarration bool
}

type WriteBriefingResult struct {
	Written  int
	Narrated int
	// Failed names the roles whose page could not be written. One role's
	// failure never costs the others their morning.
	Failed    []string
	Briefings []*briefing.Briefing
}

type GetBriefingRequest struct {
	TenantInfo pagination.TenantInfo
	RoleKey    briefing.RoleKey
	// BriefingDate is the organization's local day; empty is today.
	BriefingDate string
}

type ListBriefingsRequest struct {
	TenantInfo pagination.TenantInfo
	RoleKey    briefing.RoleKey
	Limit      int
}

type BriefingService interface {
	WriteForDay(ctx context.Context, req WriteBriefingRequest) (*WriteBriefingResult, error)
	// Today is the reader's page for the organization's current day, or
	// nil when the morning job has not run yet.
	Today(ctx context.Context, req GetBriefingRequest, actor *RequestActor) (*briefing.Briefing, error)
	Get(ctx context.Context, tenant pagination.TenantInfo, id pulid.ID, actor *RequestActor) (*briefing.Briefing, error)
	List(ctx context.Context, req ListBriefingsRequest, actor *RequestActor) ([]*briefing.Briefing, error)
	MarkRead(ctx context.Context, tenant pagination.TenantInfo, id pulid.ID, actor *RequestActor) (*briefing.Briefing, error)
	Regenerate(ctx context.Context, req GetBriefingRequest, actor *RequestActor) (*briefing.Briefing, error)
}
