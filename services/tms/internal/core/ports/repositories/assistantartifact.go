package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ListArtifactsRequest reads a thread's artifacts, pinned first then newest.
type ListArtifactsRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	// Limit bounds the read; zero means the repository's default.
	Limit int
}

type GetArtifactRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type SetArtifactPinnedRequest struct {
	ID         pulid.ID
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	Pinned     bool
}

// UpdateArtifactStatusRequest moves an artifact along, such as a draft that
// was sent or a run that finished, and may replace what it shows.
type UpdateArtifactStatusRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Status     assistantartifact.Status
	Payload    map[string]any
}

type AssistantArtifactRepository interface {
	// Upsert writes an artifact, replacing the one with the same thread,
	// source tool call and kind, or the same proposal or plan, when there is
	// one: a retried turn updates what it produced rather than adding a twin.
	Upsert(
		ctx context.Context,
		artifact *assistantartifact.Artifact,
	) (*assistantartifact.Artifact, error)
	ListByThread(
		ctx context.Context,
		req ListArtifactsRequest,
	) ([]*assistantartifact.Artifact, error)
	GetByID(ctx context.Context, req GetArtifactRequest) (*assistantartifact.Artifact, error)
	SetPinned(
		ctx context.Context,
		req SetArtifactPinnedRequest,
	) (*assistantartifact.Artifact, error)
	UpdateStatus(
		ctx context.Context,
		req UpdateArtifactStatusRequest,
	) (*assistantartifact.Artifact, error)
	// FindByProposal reads the artifact that views a proposal, for the
	// status to follow the decision.
	FindByProposal(
		ctx context.Context,
		tenant pagination.TenantInfo,
		proposalID pulid.ID,
	) (*assistantartifact.Artifact, error)
}
