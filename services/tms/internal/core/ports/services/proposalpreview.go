package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ResourceReadAccess answers whether one reader may read a resource at all.
// An answer that cannot be had is no.
type ResourceReadAccess interface {
	MayRead(ctx context.Context, resource permission.Resource) bool
}

// PreviewViewer is the person a preview is filtered for. Ceilings and Reads
// are the request's own caches when there are any; the service makes its own
// when they are nil.
type PreviewViewer struct {
	Actor    *RequestActor
	Ceilings FieldCeilings
	Reads    ResourceReadAccess
}

// ProposalPreviewRequest asks what one proposal's write would do, with the
// approver's changes when they have made some.
type ProposalPreviewRequest struct {
	Proposal      *agent.AgentProposal
	Modifications map[string]any
	Viewer        *PreviewViewer
	// Params are the parameters as they would run, already checked, from a
	// caller that settled the approver's changes itself; they stand in for
	// Modifications. Never taken from a request.
	Params map[string]any
	// Deciding says the preview is being computed for a decision rather
	// than for a person to read.
	Deciding bool
}

// PlanPreviewRequest asks what every pending step of a plan would do, in
// order.
type PlanPreviewRequest struct {
	Plan   *agent.AgentPlan
	Steps  []*agent.AgentProposal
	Viewer *PreviewViewer
}

// ProposalBaselineRequest asks for the baseline of a write being proposed:
// its target's version and what it would do, read in one snapshot. Persist
// keeps the baseline for the proposal; an evaluation or a simulated write
// keeps nothing.
type ProposalBaselineRequest struct {
	ProposalID pulid.ID
	Tool       AgentTool
	Params     ToolExecuteParams
	Persist    bool
}

// ProposalBaselineResult is what a baseline found. Target is nil when the
// tool names no single record or its version could not be read; Preview is
// nil when the tool has no preview or it failed, and PreviewErr then says
// why it failed.
type ProposalBaselineResult struct {
	Target     *ProposalTarget
	Preview    *agent.ToolPreview
	PreviewErr error
}

// RecordLabels are the words each record is known by, per resource and id.
// A record that is gone, or of a resource no label is kept for, is absent.
type RecordLabels map[permission.Resource]map[pulid.ID]string

// Label is one record's label, or "" when there is none.
func (l RecordLabels) Label(resource permission.Resource, id pulid.ID) string {
	return l[resource][id]
}

// RecordLabeler reads the labels of many records at once, one query per
// resource, inside the tenant.
type RecordLabeler interface {
	Labels(
		ctx context.Context,
		tenant pagination.TenantInfo,
		refs map[permission.Resource][]pulid.ID,
	) (RecordLabels, error)
}

// ProposalPreviewService says what a proposed write would do, for one
// reader, from the world as it is now.
type ProposalPreviewService interface {
	// ForProposal previews a pending proposal as its reader would approve it,
	// or returns the preview recorded when it was decided, filtered for this
	// reader.
	ForProposal(ctx context.Context, req *ProposalPreviewRequest) (*agent.ProposalPreview, error)
	// ForPlan previews a plan's pending steps, each projected from the ones
	// before it.
	ForPlan(ctx context.Context, req *PlanPreviewRequest) (*agent.PlanPreview, error)
	// ForDecision is a decision's recorded preview, filtered for the reader.
	ForDecision(
		ctx context.Context,
		decision *agent.AgentDecision,
		viewer *PreviewViewer,
	) *agent.ProposalPreview
	// Baseline pins the target and previews the write in one read-only
	// snapshot. It never fails the proposal: whatever could not be read is
	// left out.
	Baseline(ctx context.Context, req *ProposalBaselineRequest) *ProposalBaselineResult
}
