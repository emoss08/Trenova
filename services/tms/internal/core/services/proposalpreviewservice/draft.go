package proposalpreviewservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/proposalexecutor"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type draftInput struct {
	proposal      *agent.AgentProposal
	modifications map[string]any
	params        map[string]any
	actor         *services.RequestActor
	timeout       time.Duration
}

// draft is one proposal's preview before it is filtered for a reader: every
// value the tool reported, labelled and compared with the pinned record.
type draft struct {
	proposal *agent.AgentProposal
	params   map[string]any
	preview  *agent.ProposalPreview
}

func (s *Service) draft(ctx context.Context, in *draftInput) (*draft, error) {
	proposal := in.proposal
	d := &draft{
		proposal: proposal,
		preview: &agent.ProposalPreview{
			Schema:     agent.PreviewSchemaVersion,
			ProposalID: proposal.ID,
			Tool:       proposal.ToolName,
			ComputedAt: s.now(),
		},
	}

	tool, ok := s.tools.Get(proposal.ToolName)
	if !ok {
		d.params = maps.Clone(proposal.ToolParams)
		s.toolRemoved(d)

		return d, s.pinOutside(ctx, d)
	}

	if err := assertOwner(tool, proposal, in.actor); err != nil {
		return nil, err
	}

	params, warnings, err := s.settleParams(ctx, tool, in)
	if err != nil {
		return nil, err
	}
	d.params = params

	policy := tool.Policy()
	found, err := s.snapshot(ctx, &snapshotInput{
		tool:     tool,
		params:   proposalexecutor.ExecutionParams(proposal, &policy, params, in.actor),
		pin:      pinOf(proposal),
		timeout:  in.timeout,
		labelled: true,
	})
	if err != nil {
		return nil, err
	}

	s.assemble(d, tool, found)
	d.preview.Staleness = found.staleness(proposal)
	d.preview.TargetVersion = found.version
	if proposal.TargetID.IsNil() {
		if _, targeted := targetOf(tool, params); targeted {
			d.preview.AddWarning(agent.PreviewWarning{
				Code: agent.PreviewWarningUnpinned,
				Message: "The record this change is for was not pinned when it was proposed, " +
					"so whether it has changed since cannot be told.",
			})
		}
	}
	if d.preview.IsStale() {
		d.preview.AddWarning(staleWarning(d.preview.Staleness, proposal.TargetResource))
	}
	for _, warning := range warnings {
		d.preview.AddWarning(warning)
	}

	return d, nil
}

// settleParams is the parameters the write would run with. Changes an
// approver has in mind are checked the way a decision checks them, and one
// that could not run, or that points the write at another record, is
// refused. Without changes, a tool that checks its own arguments is asked,
// and a call that would fail is a warning rather than a refusal: the person
// is shown what was proposed and why it would not run.
func (s *Service) settleParams(
	ctx context.Context,
	tool services.AgentTool,
	in *draftInput,
) (map[string]any, []agent.PreviewWarning, error) {
	proposal := in.proposal
	if in.params != nil {
		return in.params, nil, nil
	}

	changed := toolschema.Changed(proposal.ToolParams, in.modifications)
	if len(changed) > 0 {
		if s.checker == nil {
			return nil, nil, errortypes.NewBusinessError(
				"Changes to a proposal cannot be checked right now. Try again shortly",
			)
		}
		params, err := s.checker.CheckModifications(ctx, proposal, changed, in.actor)
		if err != nil {
			return nil, nil, err
		}

		return params, nil, nil
	}

	params := maps.Clone(proposal.ToolParams)
	validator, validates := tool.(services.ToolValidator)
	if !validates {
		return params, nil, nil
	}
	policy := tool.Policy()
	if err := validator.Validate(
		ctx,
		proposalexecutor.ExecutionParams(proposal, &policy, params, in.actor),
	); err != nil {
		warnings := []agent.PreviewWarning{toolpreview.WouldFail(err)}
		toolpreview.LocateReasons(warnings, tool.ParamSchema())

		//nolint:nilerr // a call that would fail is shown with a warning, not refused
		return params, warnings, nil
	}

	return params, nil, nil
}

// assertOwner keeps a self-scoped proposal's preview to the person it is
// for: it is about their own records, and nobody else's queue shows it.
func assertOwner(
	tool services.AgentTool,
	proposal *agent.AgentProposal,
	actor *services.RequestActor,
) error {
	if !services.IsSelfScoped(tool) {
		return nil
	}

	owner, _ := proposal.ToolParams[services.SelfScopeOwnerParam].(string)
	if actor == nil || actor.UserID.IsNil() || owner != actor.UserID.String() {
		return errortypes.NewNotFoundError("That proposal is not one you can preview")
	}

	return nil
}

type pin struct {
	target  services.ToolTarget
	version int64
}

func pinOf(proposal *agent.AgentProposal) *pin {
	if proposal.TargetID.IsNil() {
		return nil
	}

	return &pin{
		target: services.ToolTarget{
			Resource: permission.Resource(proposal.TargetResource),
			ID:       proposal.TargetID,
		},
		version: proposal.TargetVersion,
	}
}

func targetOf(tool services.AgentTool, params map[string]any) (services.ToolTarget, bool) {
	targeted, ok := tool.(services.TargetedTool)
	if !ok {
		return services.ToolTarget{}, false
	}

	return targeted.Target(params)
}

type snapshotInput struct {
	tool     services.AgentTool
	params   services.ToolExecuteParams
	pin      *pin
	timeout  time.Duration
	labelled bool
}

// snapshotResult is what one read-only snapshot found: the target's version
// and the tool's preview, read from the same state, and the labels of the
// records the preview names.
type snapshotResult struct {
	done       bool
	pinRead    bool
	version    *int64
	missing    bool
	preview    *agent.ToolPreview
	previewErr error
	labels     services.RecordLabels
}

// staleness compares the pinned version with the one read now. It is
// unknown, and nil, when the record's version could not be read: the
// executor still refuses a changed record when the write runs.
func (r *snapshotResult) staleness(proposal *agent.AgentProposal) *agent.PreviewStaleness {
	if proposal.TargetID.IsNil() || !r.pinRead {
		return nil
	}

	staleness := &agent.PreviewStaleness{
		Pinned:          true,
		ProposedVersion: proposal.TargetVersion,
		Missing:         r.missing,
	}
	if r.version != nil {
		staleness.CurrentVersion = *r.version
	}

	return staleness
}

// snapshot reads the target's version and runs the tool's preview in one
// read-only, repeatable-read transaction. A write the preview attempted
// fails there. The version is read first, so a preview that runs out of time
// still leaves the staleness known.
func (s *Service) snapshot(ctx context.Context, in *snapshotInput) (*snapshotResult, error) {
	result := &snapshotResult{}
	err := s.db.WithTx(ctx, ports.TxOptions{
		ReadOnly:  true,
		Isolation: sql.LevelRepeatableRead,
	}, func(txCtx context.Context, _ bun.Tx) error {
		if err := s.readPin(txCtx, in, result); err != nil {
			return err
		}

		if previewer, ok := in.tool.(services.ToolPreviewer); ok {
			previewCtx, cancel := context.WithTimeout(txCtx, in.timeout)
			result.preview, result.previewErr = runPreview(previewCtx, previewer, &in.params)
			cancel()
			if result.preview != nil {
				toolpreview.LocateReasons(result.preview.Warnings, in.tool.ParamSchema())
			}
		}

		if in.labelled && result.previewErr == nil && result.preview != nil {
			result.labels = s.labels(txCtx, &in.params, result.preview)
		}
		result.done = true

		return nil
	})
	if err != nil && !result.done {
		return nil, err
	}
	if err != nil {
		s.l.Debug("preview snapshot ended without a clean commit", zap.Error(err))
	}

	return result, nil
}

func (s *Service) readPin(ctx context.Context, in *snapshotInput, result *snapshotResult) error {
	if in.pin == nil || s.versions == nil {
		return nil
	}

	version, err := s.versions.Version(ctx, tenantOfParams(&in.params), in.pin.target)
	switch {
	case err == nil:
		result.version = &version
		result.pinRead = true
	case errors.Is(err, sql.ErrNoRows):
		result.missing = true
		result.pinRead = true
	case errors.Is(err, services.ErrRecordVersionUnsupported):
	default:
		return fmt.Errorf("read the %s this change is for: %w", in.pin.target.Resource, err)
	}

	return nil
}

// runPreview asks the tool what its write would do. A tool that panics has
// failed its preview, never the request.
func runPreview(
	ctx context.Context,
	previewer services.ToolPreviewer,
	params *services.ToolExecuteParams,
) (preview *agent.ToolPreview, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			preview = nil
			err = fmt.Errorf("the preview failed: %v", recovered)
		}
	}()

	preview, err = previewer.Preview(ctx, *params)
	if err == nil && preview == nil {
		err = errors.New("the tool returned no preview")
	}

	return preview.Bounded(), err
}

func tenantOfParams(params *services.ToolExecuteParams) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: params.OrganizationID, BuID: params.BusinessUnitID}
}

// assemble turns the tool's preview into the proposal's: its coverage, its
// changes named by their labels, or the fallback when it had nothing to say.
func (s *Service) assemble(d *draft, tool services.AgentTool, found *snapshotResult) {
	preview := d.preview
	switch {
	case found.previewErr != nil:
		s.l.Warn("a tool's preview failed; showing its parameters instead",
			zap.String("tool", tool.Name()),
			zap.String("proposal", d.proposal.ID.String()),
			zap.Error(found.previewErr),
		)
		s.fallback(d, tool)
		preview.AddWarning(agent.PreviewWarning{
			Code: agent.PreviewWarningPreviewFailed,
			Args: []string{previewFailure(found.previewErr)},
			Message: "What this change would do could not be worked out: " + previewFailure(
				found.previewErr,
			),
		})

		return
	case found.preview == nil:
		s.fallback(d, tool)

		return
	}

	preview.Summary = found.preview.Summary
	preview.Changes = found.preview.Changes
	preview.OmittedRecords = found.preview.OmittedRecords
	for _, warning := range found.preview.Warnings {
		preview.AddWarning(warning)
	}
	preview.Coverage = agent.PreviewCoverageFull
	if found.preview.Partial {
		preview.Coverage = agent.PreviewCoveragePartial
	}
	applyLabels(preview.Changes, found.labels)
}

// fallback is a tool that cannot say what it would change: the parameters it
// would run with, without the ones its policy holds Confidential.
func (s *Service) fallback(d *draft, tool services.AgentTool) {
	params := toolpreview.Parameters(d.proposal.ToolName, tool.Policy().Resource, d.params)
	d.preview.Summary = params.Summary
	d.preview.Changes = params.Changes
	d.preview.Coverage = agent.PreviewCoverageUnavailable
}

func (s *Service) toolRemoved(d *draft) {
	d.preview.Summary = "The tool " + d.proposal.ToolName + " that this change would run " +
		"no longer exists, so it cannot run."
	d.preview.Coverage = agent.PreviewCoverageUnavailable
	d.preview.AddWarning(agent.PreviewWarning{
		Code:    agent.PreviewWarningToolRemoved,
		Args:    []string{d.proposal.ToolName},
		Message: "The tool " + d.proposal.ToolName + " no longer exists.",
	})
}

// pinOutside reads the staleness of a proposal whose tool is gone, which no
// preview snapshot will read.
func (s *Service) pinOutside(ctx context.Context, d *draft) error {
	p := pinOf(d.proposal)
	if p == nil || s.versions == nil {
		return nil
	}

	result := &snapshotResult{}
	if err := s.readPin(ctx, &snapshotInput{
		pin: p,
		params: services.ToolExecuteParams{
			OrganizationID: d.proposal.OrganizationID,
			BusinessUnitID: d.proposal.BusinessUnitID,
		},
	}, result); err != nil {
		return err
	}
	d.preview.Staleness = result.staleness(d.proposal)
	d.preview.TargetVersion = result.version

	return nil
}

func previewFailure(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "it took too long"
	}

	return err.Error()
}

func staleWarning(staleness *agent.PreviewStaleness, resource string) agent.PreviewWarning {
	if staleness.Missing {
		return agent.PreviewWarning{
			Code:    agent.PreviewWarningRecordMissing,
			Args:    []string{resource},
			Message: "The " + resource + " this change is for no longer exists.",
		}
	}

	return agent.PreviewWarning{
		Code: agent.PreviewWarningTargetChanged,
		Args: []string{
			resource,
			strconv.FormatInt(staleness.ProposedVersion, 10),
			strconv.FormatInt(staleness.CurrentVersion, 10),
		},
		Message: "The " + resource + " has changed since this was proposed: it was at version " +
			strconv.FormatInt(staleness.ProposedVersion, 10) + " and is now at " +
			strconv.FormatInt(staleness.CurrentVersion, 10) + ".",
	}
}

// labels reads the label of every record the preview names, in one query per
// resource, inside the same snapshot. A label that cannot be read leaves the
// record named by nothing, never by its id.
func (s *Service) labels(
	ctx context.Context,
	params *services.ToolExecuteParams,
	preview *agent.ToolPreview,
) services.RecordLabels {
	if s.labeler == nil {
		return nil
	}

	refs := make(map[permission.Resource][]pulid.ID, 4)
	add := func(resource permission.Resource, id pulid.ID) {
		if resource != "" && id.IsNotNil() {
			refs[resource] = append(refs[resource], id)
		}
	}
	for i := range preview.Changes {
		change := &preview.Changes[i]
		if change.Label == "" {
			add(change.Resource, change.EntityID)
		}
		for j := range change.Fields {
			if ref := change.Fields[j].BeforeRef; ref != nil {
				add(ref.Resource, ref.ID)
			}
			if ref := change.Fields[j].AfterRef; ref != nil {
				add(ref.Resource, ref.ID)
			}
		}
	}
	if len(refs) == 0 {
		return nil
	}

	labels, err := s.labeler.Labels(ctx, tenantOfParams(params), refs)
	if err != nil {
		s.l.Warn("could not read the labels of the records a preview names", zap.Error(err))

		return nil
	}

	return labels
}

func applyLabels(changes []agent.RecordChange, labels services.RecordLabels) {
	for i := range changes {
		change := &changes[i]
		if change.Label == "" {
			change.Label = labels.Label(change.Resource, change.EntityID)
		}
		if change.Record == nil {
			change.Record = recordLink(change.Resource, change.EntityID)
		}
		for j := range change.Fields {
			labelRef(change.Fields[j].BeforeRef, labels)
			labelRef(change.Fields[j].AfterRef, labels)
		}
	}
}

func labelRef(ref *agent.PreviewRef, labels services.RecordLabels) {
	if ref == nil {
		return
	}
	if ref.Label == "" {
		ref.Label = labels.Label(ref.Resource, ref.ID)
	}
	if ref.Record == nil {
		ref.Record = recordLink(ref.Resource, ref.ID)
	}
}

// recordLink is where a record opens, when the record-link registry has a
// page for its kind.
func recordLink(resource permission.Resource, id pulid.ID) *agent.RecordRef {
	if resource == "" || id.IsNil() {
		return nil
	}
	if _, ok := productguide.Default.Record(resource.String()); !ok {
		return nil
	}

	return (&agent.RecordRef{EntityType: resource.String(), ID: id.String()}).Bounded()
}
