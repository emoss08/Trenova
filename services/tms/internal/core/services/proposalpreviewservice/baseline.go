package proposalpreviewservice

import (
	"bytes"
	"context"
	"maps"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// Baseline pins the target and previews the write in one read-only
// snapshot, when a proposal is filed. Whatever cannot be read is left out:
// a baseline never fails the proposal. A kept baseline holds every value but
// a Confidential one; it is read only to tell which values moved since.
func (s *Service) Baseline(
	ctx context.Context,
	req *services.ProposalBaselineRequest,
) *services.ProposalBaselineResult {
	result := &services.ProposalBaselineResult{}
	if req == nil || req.Tool == nil {
		return result
	}

	ctx, span := aitrace.StartPreview(ctx, &aitrace.PreviewSpec{
		ToolName:       req.Tool.Name(),
		ProposalID:     req.ProposalID,
		OrganizationID: req.Params.OrganizationID,
		BusinessUnitID: req.Params.BusinessUnitID,
		Purpose:        aitrace.PreviewPurposeBaseline,
	})
	defer span.End()

	in := &snapshotInput{tool: req.Tool, params: req.Params, timeout: s.fileTimeout}
	target, targeted := targetOf(req.Tool, req.Params.Params)
	if targeted {
		in.pin = &pin{target: target}
	}

	found, err := s.snapshot(ctx, in)
	if err != nil {
		s.l.Warn("could not take a proposal's baseline",
			zap.String("tool", req.Tool.Name()), zap.Error(err))
		aitrace.MarkFailed(span, aitrace.OutcomeFailed)
		result.PreviewErr = err

		return result
	}

	if found.version != nil {
		result.Target = &services.ProposalTarget{
			Resource: target.Resource,
			ID:       target.ID,
			Version:  *found.version,
		}
	}
	if found.previewErr != nil {
		result.PreviewErr = found.previewErr
		s.l.Info("a proposal was filed without a baseline: its preview failed",
			zap.String("tool", req.Tool.Name()), zap.Error(found.previewErr))
	}
	if found.previewErr == nil && found.preview != nil {
		result.Preview = found.preview
	}

	coverage := agent.PreviewCoverageUnavailable
	if result.Preview != nil {
		coverage = agent.PreviewCoverageFull
		if result.Preview.Partial {
			coverage = agent.PreviewCoveragePartial
		}
	}
	aitrace.FinishPreview(span, &aitrace.PreviewResult{
		Coverage: string(coverage),
		Records:  previewRecords(result.Preview),
	})

	if req.Persist {
		s.keepBaseline(ctx, req, result)
	}

	return result
}

func previewRecords(preview *agent.ToolPreview) int {
	if preview == nil {
		return 0
	}

	return len(preview.Changes)
}

func (s *Service) keepBaseline(
	ctx context.Context,
	req *services.ProposalBaselineRequest,
	result *services.ProposalBaselineResult,
) {
	if s.baselines == nil || result.Preview == nil || req.ProposalID.IsNil() {
		return
	}

	baseline := &agent.ProposalBaseline{
		ProposalID:     req.ProposalID,
		OrganizationID: req.Params.OrganizationID,
		BusinessUnitID: req.Params.BusinessUnitID,
		ToolName:       req.Tool.Name(),
		Preview:        result.Preview,
	}
	if result.Target != nil {
		version := result.Target.Version
		baseline.TargetVersion = &version
	}

	multiErr := errortypes.NewMultiError()
	baseline.Validate(multiErr)
	if multiErr.HasErrors() {
		s.l.Info("a proposal's baseline was not kept",
			zap.String("tool", req.Tool.Name()),
			zap.String("proposal", req.ProposalID.String()),
			zap.Error(multiErr),
		)

		return
	}

	if err := s.baselines.Create(ctx, baseline); err != nil {
		s.l.Warn("could not keep a proposal's baseline",
			zap.String("tool", req.Tool.Name()),
			zap.String("proposal", req.ProposalID.String()),
			zap.Error(err),
		)
	}
}

type valueKey struct {
	resource permission.Resource
	id       pulid.ID
	path     string
}

// markChangedSinceProposed compares each value's current "before" with what
// it was when the proposal was filed, and says which moved: "was 40,000 when
// proposed". A proposal filed before baselines were kept, or whose baseline
// could not be taken, is compared with nothing.
func (s *Service) markChangedSinceProposed(
	ctx context.Context,
	tenant pagination.TenantInfo,
	drafts []*draft,
) {
	if s.baselines == nil || len(drafts) == 0 {
		return
	}

	ids := make([]pulid.ID, 0, len(drafts))
	for _, d := range drafts {
		ids = append(ids, d.proposal.ID)
	}
	baselines, err := s.baselines.ListByProposals(ctx, repositories.ListProposalBaselinesRequest{
		ProposalIDs: ids,
		TenantInfo:  tenant,
	})
	if err != nil {
		s.l.Warn("could not read the baselines of proposals being previewed", zap.Error(err))

		return
	}

	byProposal := make(map[pulid.ID]*agent.ToolPreview, len(baselines))
	for _, baseline := range baselines {
		if baseline != nil && baseline.Preview != nil {
			byProposal[baseline.ProposalID] = baseline.Preview
		}
	}

	for _, d := range drafts {
		if baseline, ok := byProposal[d.proposal.ID]; ok {
			markAgainst(d.preview, baseline)
		}
	}
}

func markAgainst(preview *agent.ProposalPreview, baseline *agent.ToolPreview) {
	proposed := make(map[valueKey]any, 16)
	for i := range baseline.Changes {
		change := &baseline.Changes[i]
		if change.EntityID.IsNil() || change.Operation == agent.PreviewOperationCreate {
			continue
		}
		for j := range change.Fields {
			field := &change.Fields[j]
			proposed[valueKey{change.Resource, change.EntityID, field.Path}] = field.Before
		}
	}

	for i := range preview.Changes {
		change := &preview.Changes[i]
		if change.EntityID.IsNil() || change.Operation == agent.PreviewOperationCreate {
			continue
		}
		for j := range change.Fields {
			field := &change.Fields[j]
			before, known := proposed[valueKey{change.Resource, change.EntityID, field.Path}]
			if !known || sameValue(before, field.Before) {
				continue
			}
			field.ChangedSinceProposed = true
			field.ProposedBefore = before
		}
	}
}

func sameValue(a, b any) bool {
	left, err := jsonutils.CanonicalMarshal(a)
	if err != nil {
		return false
	}
	right, err := jsonutils.CanonicalMarshal(b)
	if err != nil {
		return false
	}

	return bytes.Equal(left, right)
}

// chain projects a plan's pending steps from one another, in order.
func (s *Service) chain(drafts []*draft) {
	if len(drafts) < 2 {
		return
	}

	steps := make([]toolpreview.ChainStep, 0, len(drafts))
	for _, d := range drafts {
		steps = append(steps, toolpreview.ChainStep{Step: d.proposal.PlanStep, Preview: d.preview})
	}
	toolpreview.Chain(steps)
}

// recordedFor is a decided proposal's preview: the one recorded when it was
// decided, filtered for this reader, or, when none was recorded, the
// parameters it was decided on.
func (s *Service) recordedFor(
	ctx context.Context,
	proposal *agent.AgentProposal,
	a readerAccess,
) (*agent.ProposalPreview, error) {
	if s.decisions != nil {
		decisions, err := s.decisions.ListByProposals(
			ctx,
			repositories.ListAgentDecisionsByProposalsRequest{
				ProposalIDs: []pulid.ID{proposal.ID},
				TenantInfo:  tenantOf(proposal),
			},
		)
		if err != nil {
			return nil, err
		}
		for _, decision := range decisions {
			if decision == nil || decision.Preview == nil {
				continue
			}
			if recorded := s.recorded(ctx, decision, a); recorded != nil {
				return recorded, nil
			}
		}
	}

	return s.unrecorded(ctx, proposal, a), nil
}

func (s *Service) unrecorded(
	ctx context.Context,
	proposal *agent.AgentProposal,
	a readerAccess,
) *agent.ProposalPreview {
	preview := &agent.ProposalPreview{
		Schema:     agent.PreviewSchemaVersion,
		ProposalID: proposal.ID,
		Tool:       proposal.ToolName,
		Coverage:   agent.PreviewCoverageUnavailable,
		Recorded:   true,
		ComputedAt: s.now(),
		Summary: "No preview was recorded when this was decided; " +
			"the parameters it was proposed with are shown.",
	}

	if tool, ok := s.tools.Get(proposal.ToolName); ok && !services.IsSelfScoped(tool) {
		params := toolpreview.Parameters(
			proposal.ToolName,
			tool.Policy().Resource,
			maps.Clone(proposal.ToolParams),
		)
		preview.Changes = params.Changes
	}
	filter(ctx, preview, a)

	return preview
}
