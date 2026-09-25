package proposalpreviewservice

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func changeOf(t *testing.T, preview *agent.ProposalPreview) *agent.RecordChange {
	t.Helper()
	require.NotEmpty(t, preview.Changes)

	return &preview.Changes[0]
}

func fieldOf(t *testing.T, change *agent.RecordChange, path string) *agent.PreviewFieldChange {
	t.Helper()
	for i := range change.Fields {
		if change.Fields[i].Path == path {
			return &change.Fields[i]
		}
	}
	require.Failf(t, "no field", "%s not in %+v", path, change.Fields)

	return nil
}

func TestForProposal_PreviewsInAReadOnlySnapshotAndDigestsWhatWasShown(t *testing.T) {
	t.Parallel()

	f := newFixture()
	proposal := f.proposal(f.cancelParams())

	preview, err := f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal: proposal,
		Viewer:   f.viewer(),
	})
	require.NoError(t, err)

	require.Len(t, f.db.opts, 1)
	assert.True(t, f.db.opts[0].ReadOnly, "a preview never writes")
	assert.Equal(t, sql.LevelRepeatableRead, f.db.opts[0].Isolation)

	assert.Equal(t, agent.PreviewCoverageFull, preview.Coverage)
	assert.Equal(t, "Would set PRO PRO-100 to Cancelled.", preview.Summary)
	assert.False(t, preview.IsStale())
	require.NotNil(t, preview.TargetVersion)
	assert.Equal(t, int64(3), *preview.TargetVersion)
	assert.Len(t, preview.Digest, 64)
	assert.Zero(t, preview.WithheldCount)

	change := changeOf(t, preview)
	assert.Equal(t, "PRO-100", change.Label)
	status := fieldOf(t, change, "status")
	assert.Equal(t, "New", status.Before)
	assert.Equal(t, "Cancelled", status.After)

	again, err := f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal: proposal,
		Viewer:   f.viewer(),
	})
	require.NoError(t, err)
	assert.Equal(t, preview.Digest, again.Digest, "the same world digests alike")

	modified, err := f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal:      proposal,
		Modifications: map[string]any{"status": "Hold"},
		Viewer:        f.viewer(),
	})
	require.NoError(t, err)
	assert.Equal(t, "Hold", fieldOf(t, changeOf(t, modified), "status").After,
		"the approver's changes are what is previewed")
	assert.NotEqual(t, preview.Digest, modified.Digest)
	assert.Equal(t, "Hold", f.tool.asked[len(f.tool.asked)-1].Params["status"])
	assert.Equal(t, proposal.ID, f.tool.asked[0].ProposalID)
}

func TestForProposal_RefusesAReaderFromAnotherTenant(t *testing.T) {
	t.Parallel()

	f := newFixture()
	viewer := f.viewer()
	viewer.Actor.OrganizationID = pulid.MustNew("org_")

	_, err := f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal: f.proposal(f.cancelParams()),
		Viewer:   viewer,
	})

	require.ErrorIs(t, err, ErrTenantMismatch)
	assert.Empty(t, f.tool.asked, "the tool is never asked for another tenant's record")
}

func TestForProposal_LabelsTheRecordsItNames(t *testing.T) {
	t.Parallel()

	f := newFixture()
	other := pulid.MustNew("cus_")
	f.labeler.labels[permission.ResourceCustomer][other] = "Other Co"
	params := f.cancelParams()
	params["customerId"] = other.String()

	preview, err := f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal: f.proposal(params),
		Viewer:   f.viewer(),
	})
	require.NoError(t, err)

	change := changeOf(t, preview)
	require.NotNil(t, change.Record, "a shipment opens on its own page")
	assert.Equal(t, "shipment", change.Record.EntityType)
	customer := fieldOf(t, change, "customerId")
	require.NotNil(t, customer.BeforeRef)
	require.NotNil(t, customer.AfterRef)
	assert.Equal(t, "Acme Foods", customer.BeforeRef.Label)
	assert.Equal(t, "Other Co", customer.AfterRef.Label)
	require.Len(t, f.labeler.asked, 1, "one batch for every record the preview names")
	assert.ElementsMatch(t, []pulid.ID{f.customer, other},
		f.labeler.asked[0][permission.ResourceCustomer])
}

func TestForProposal_WithholdsWhatTheReaderMayNotSee(t *testing.T) {
	t.Parallel()

	customerChange := &agent.RecordChange{
		Resource:  permission.ResourceCustomer,
		EntityID:  pulid.MustNew("cus_"),
		Label:     "Acme Foods",
		Operation: agent.PreviewOperationSend,
		Message:   &agent.MessagePreview{Channel: agent.MessageChannelEmail, Body: "Hello"},
	}
	preview := &agent.ProposalPreview{Changes: []agent.RecordChange{
		{
			Resource:  permission.ResourceShipment,
			EntityID:  pulid.MustNew("shp_"),
			Label:     "PRO-100",
			Operation: agent.PreviewOperationUpdate,
			Fields: []agent.PreviewFieldChange{
				{
					Path:        "status",
					Before:      "New",
					After:       "Hold",
					Sensitivity: permission.SensitivityInternal,
				},
				{
					Path:        "rateAmount",
					Before:      "100",
					After:       "120",
					Sensitivity: permission.SensitivityRestricted,
				},
				{
					Path:   "customerId",
					Before: "cus_1",
					After:  "cus_2",
					AfterRef: &agent.PreviewRef{
						Resource: permission.ResourceCustomer,
						Label:    "Acme",
					},
				},
			},
			Money: &agent.MoneyPreview{Sensitivity: permission.SensitivityRestricted},
		},
		*customerChange,
	}}

	a := readerAccess{
		ceilings: fakeCeilings{levels: map[permission.Resource]permission.FieldSensitivity{
			permission.ResourceShipment: permission.SensitivityInternal,
		}},
		reads: fakeReads{denied: map[permission.Resource]bool{permission.ResourceCustomer: true}},
	}
	filter(t.Context(), preview, a)

	shipment := preview.Changes[0]
	assert.Equal(t, "New", shipment.Fields[0].Before, "an internal value is shown")
	assert.True(t, shipment.Fields[1].Withheld, "a restricted amount above the ceiling is not")
	assert.Nil(t, shipment.Fields[1].Before)
	assert.Nil(t, shipment.Fields[1].After)
	assert.True(
		t,
		shipment.Fields[2].AfterRef.Withheld,
		"a customer's name the reader may not read",
	)
	assert.Empty(t, shipment.Fields[2].AfterRef.Label)
	assert.True(t, shipment.Money.Withheld)

	customer := preview.Changes[1]
	assert.True(t, customer.Withheld, "a record of a resource the reader may not read goes whole")
	assert.Empty(t, customer.Label)
	assert.Nil(t, customer.Message)
	assert.True(t, customer.EntityID.IsNil())

	assert.Equal(t, 4, preview.WithheldCount)
	assert.True(t, preview.HasWarning(agent.PreviewWarningWithheld))
}

func TestForProposal_WithheldValuesAreNotInTheDigest(t *testing.T) {
	t.Parallel()

	f := newFixture()
	proposal := f.proposal(f.cancelParams())
	open, err := f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal: proposal,
		Viewer:   f.viewer(),
	})
	require.NoError(t, err)

	narrow := f.viewer()
	narrow.Ceilings = fakeCeilings{levels: map[permission.Resource]permission.FieldSensitivity{
		permission.ResourceShipment: permission.SensitivityPublic,
	}}
	hidden, err := f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal: proposal,
		Viewer:   narrow,
	})
	require.NoError(t, err)

	assert.Positive(t, hidden.WithheldCount)
	assert.NotEqual(t, open.Digest, hidden.Digest, "the digest says what this reader saw")
	assert.True(t, fieldOf(t, changeOf(t, hidden), "status").Withheld)
}

func TestForProposal_IsStaleWhenThePinnedRecordMoved(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.versions.versions[f.tool.state.ID] = 4
	f.tool.state.Status = "Hold"
	proposal := f.proposal(f.cancelParams())
	f.baselines.stored = []*agent.ProposalBaseline{{
		ProposalID: proposal.ID,
		Preview: &agent.ToolPreview{Changes: []agent.RecordChange{{
			Resource:  permission.ResourceShipment,
			EntityID:  f.tool.state.ID,
			Operation: agent.PreviewOperationUpdate,
			Fields: []agent.PreviewFieldChange{
				{Path: "status", Before: "New", After: "Cancelled"},
			},
		}}},
	}}

	preview, err := f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal: proposal,
		Viewer:   f.viewer(),
	})
	require.NoError(t, err)

	assert.True(t, preview.IsStale())
	assert.Equal(t, int64(3), preview.Staleness.ProposedVersion)
	assert.Equal(t, int64(4), preview.Staleness.CurrentVersion)
	assert.True(t, preview.HasWarning(agent.PreviewWarningTargetChanged))

	status := fieldOf(t, changeOf(t, preview), "status")
	assert.True(t, status.ChangedSinceProposed)
	assert.Equal(t, "New", status.ProposedBefore, "was New when proposed")
	assert.Equal(t, "Hold", status.Before)
}

func TestForProposal_IsStaleWhenThePinnedRecordIsGone(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.versions.missing[f.tool.state.ID] = true

	preview, err := f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal: f.proposal(f.cancelParams()),
		Viewer:   f.viewer(),
	})
	require.NoError(t, err)

	assert.True(t, preview.IsStale())
	assert.True(t, preview.Staleness.Missing)
	assert.True(t, preview.HasWarning(agent.PreviewWarningRecordMissing))
}

func TestForProposal_FallsBackToTheParametersWhenAToolCannotSay(t *testing.T) {
	t.Parallel()

	plain := &validatingTool{baseTool: baseTool{
		name:     "flag_for_manual_review",
		resource: permission.ResourceShipment,
	}}
	f := newFixture(plain)
	proposal := f.proposal(map[string]any{"shipmentId": f.tool.state.ID.String(), "status": "x"})
	proposal.ToolName = plain.name

	preview, err := f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal: proposal,
		Viewer:   f.viewer(),
	})
	require.NoError(t, err)

	assert.Equal(t, agent.PreviewCoverageUnavailable, preview.Coverage)
	assert.Contains(t, preview.Summary, "Would run flag_for_manual_review")
	change := changeOf(t, preview)
	assert.Equal(t, agent.PreviewOperationRun, change.Operation)
	assert.Equal(t, "x", fieldOf(t, change, "status").After)
	assert.False(t, preview.IsStale(), "staleness is still read without a preview")
	assert.Len(t, preview.Digest, 64)
}

func TestForProposal_ReadsASimulationAsAPartialPreview(t *testing.T) {
	t.Parallel()

	simulating := &simulatingTool{baseTool: baseTool{
		name:     "place_shipment_hold",
		resource: permission.ResourceShipment,
	}}
	f := newFixture(simulating)
	proposal := f.proposal(f.cancelParams())
	proposal.ToolName = simulating.name

	preview, err := f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal: proposal,
		Viewer:   f.viewer(),
	})
	require.NoError(t, err)

	assert.Equal(t, agent.PreviewCoveragePartial, preview.Coverage)
	assert.Equal(t, "Would put the shipment on hold.", preview.Summary)
	change := changeOf(t, preview)
	assert.Equal(t, f.tool.state.ID, change.EntityID)
	assert.Equal(t, "Hold", fieldOf(t, change, "status").After)
}

func TestForProposal_APreviewThatRunsOutOfTimeIsUnavailable(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.svc.readTimeout = 20 * time.Millisecond
	f.tool.block = true

	started := time.Now()
	preview, err := f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal: f.proposal(f.cancelParams()),
		Viewer:   f.viewer(),
	})
	require.NoError(t, err)

	assert.Less(t, time.Since(started), time.Second)
	assert.Equal(t, agent.PreviewCoverageUnavailable, preview.Coverage)
	require.True(t, preview.HasWarning(agent.PreviewWarningPreviewFailed))
	assert.Contains(t, preview.Warnings[0].Message, "took too long")
	assert.NotNil(t, preview.Staleness, "the pin was read before the preview ran out of time")
}

func TestForProposal_AFailingValidationIsAWarning(t *testing.T) {
	t.Parallel()

	failing := &validatingTool{
		baseTool: baseTool{name: "release_shipment_hold", resource: permission.ResourceShipment},
		err:      errors.New("the shipment has no hold"),
	}
	f := newFixture(failing)
	proposal := f.proposal(f.cancelParams())
	proposal.ToolName = failing.name

	preview, err := f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal: proposal,
		Viewer:   f.viewer(),
	})
	require.NoError(t, err)

	require.True(t, preview.HasWarning(agent.PreviewWarningWouldFail))
}

func TestForProposal_RefusesChangesThatCouldNotRun(t *testing.T) {
	t.Parallel()

	f := newFixture()
	refusal := errortypes.NewValidationError("shipmentId", errortypes.ErrForbidden, "no")
	f.checker.err = refusal

	_, err := f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal:      f.proposal(f.cancelParams()),
		Modifications: map[string]any{"shipmentId": pulid.MustNew("shp_").String()},
		Viewer:        f.viewer(),
	})

	require.ErrorIs(t, err, refusal)
}

func TestForProposal_ASelfScopedPreviewIsTheOwnersAlone(t *testing.T) {
	t.Parallel()

	personal := &previewingTool{baseTool: baseTool{
		name:     "add_home_widget",
		resource: permission.ResourceShipment,
		scope:    agent.ToolScopeSelf,
	}}
	f := newFixture(personal)
	personal.state = f.tool.state
	viewer := f.viewer()
	proposal := f.proposal(map[string]any{
		services.SelfScopeOwnerParam: pulid.MustNew("usr_").String(),
		"status":                     "Cancelled",
	})
	proposal.ToolName = personal.name

	_, err := f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal: proposal,
		Viewer:   viewer,
	})
	require.True(t, errortypes.IsNotFoundError(err))

	proposal.ToolParams[services.SelfScopeOwnerParam] = viewer.Actor.UserID.String()
	_, err = f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal: proposal,
		Viewer:   viewer,
	})
	require.NoError(t, err)
}

func TestForProposal_ReturnsTheRecordedPreviewOnceDecided(t *testing.T) {
	t.Parallel()

	f := newFixture()
	proposal := f.proposal(f.cancelParams())
	proposal.Status = agent.ProposalStatusExecuted
	recorded := &agent.ProposalPreview{
		Schema:     agent.PreviewSchemaVersion,
		ProposalID: proposal.ID,
		Summary:    "Would cancel PRO-100.",
		Coverage:   agent.PreviewCoverageFull,
		Changes: []agent.RecordChange{{
			Resource:  permission.ResourceShipment,
			EntityID:  f.tool.state.ID,
			Operation: agent.PreviewOperationUpdate,
			Fields: []agent.PreviewFieldChange{{
				Path: "status", Before: "New", After: "Cancelled",
				Sensitivity: permission.SensitivityInternal,
			}},
		}},
	}
	f.decisions.decisions = []*agent.AgentDecision{{
		ID:             pulid.MustNew("ad_"),
		OrganizationID: f.org,
		BusinessUnitID: f.bu,
		Preview:        recorded,
		PreviewDigest:  "d1",
	}}

	preview, err := f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal: proposal,
		Viewer:   f.viewer(),
	})
	require.NoError(t, err)
	assert.True(t, preview.Recorded)
	assert.Equal(t, "d1", preview.Digest)
	assert.Empty(t, f.tool.asked, "a decided proposal is never previewed again")

	narrow := f.viewer()
	narrow.Ceilings = fakeCeilings{levels: map[permission.Resource]permission.FieldSensitivity{
		permission.ResourceShipment: permission.SensitivityPublic,
	}}
	filtered, err := f.svc.ForProposal(t.Context(), &services.ProposalPreviewRequest{
		Proposal: proposal,
		Viewer:   narrow,
	})
	require.NoError(t, err)
	assert.True(t, filtered.Changes[0].Fields[0].Withheld, "filtered again for this reader")
	assert.Equal(t, "Cancelled", recorded.Changes[0].Fields[0].After,
		"the recorded preview itself is never changed")
}

func TestForPlan_ProjectsLaterStepsFromEarlierOnes(t *testing.T) {
	t.Parallel()

	f := newFixture()
	plan := &agent.AgentPlan{
		ID:             pulid.MustNew("apl_"),
		OrganizationID: f.org,
		BusinessUnitID: f.bu,
	}
	first := f.proposal(map[string]any{"shipmentId": f.tool.state.ID.String(), "status": "Hold"})
	first.PlanID, first.PlanStep = &plan.ID, 1
	second := f.proposal(f.cancelParams())
	second.PlanID, second.PlanStep = &plan.ID, 2
	f.steps.steps = []*agent.AgentProposal{second, first}

	preview, err := f.svc.ForPlan(t.Context(), &services.PlanPreviewRequest{
		Plan:   plan,
		Viewer: f.viewer(),
	})
	require.NoError(t, err)

	require.Len(t, preview.Steps, 2)
	assert.Equal(t, 1, preview.Steps[0].Step)
	later := preview.Steps[1].Preview
	status := fieldOf(t, changeOf(t, later), "status")
	assert.Equal(t, "Hold", status.Before, "step 2 starts from what step 1 leaves")
	assert.Equal(t, 1, status.ProjectedFromStep)
	assert.Equal(t, 1, changeOf(t, later).DependsOnStep)
	assert.True(t, later.HasWarning(agent.PreviewWarningDependsOnStep))

	want, err := agent.PlanPreviewDigest(plan.ID, preview.StepDigests())
	require.NoError(t, err)
	assert.Equal(t, want, preview.Digest)
	assert.False(t, preview.Stale)

	f.versions.versions[f.tool.state.ID] = 9
	stale, err := f.svc.ForPlan(t.Context(), &services.PlanPreviewRequest{
		Plan:   plan,
		Viewer: f.viewer(),
	})
	require.NoError(t, err)
	assert.True(t, stale.Stale, "a plan is stale when any step is")
}

func TestBaseline_PinsAndKeepsWhatTheWriteWouldDo(t *testing.T) {
	t.Parallel()

	f := newFixture()
	proposalID := pulid.MustNew("ap_")
	result := f.svc.Baseline(t.Context(), &services.ProposalBaselineRequest{
		ProposalID: proposalID,
		Tool:       f.tool,
		Params: services.ToolExecuteParams{
			OrganizationID: f.org,
			BusinessUnitID: f.bu,
			Params:         f.cancelParams(),
		},
		Persist: true,
	})

	require.NotNil(t, result.Target)
	assert.Equal(t, int64(3), result.Target.Version)
	assert.Equal(t, f.tool.state.ID, result.Target.ID)
	require.NotNil(t, result.Preview)
	require.Len(t, f.baselines.kept, 1)
	kept := f.baselines.kept[0]
	assert.Equal(t, proposalID, kept.ProposalID)
	assert.Equal(t, f.org, kept.OrganizationID)
	require.NotNil(t, kept.TargetVersion)
	assert.Equal(t, int64(3), *kept.TargetVersion)
	require.Len(t, f.db.opts, 1)
	assert.True(t, f.db.opts[0].ReadOnly)

	evaluation := f.svc.Baseline(t.Context(), &services.ProposalBaselineRequest{
		ProposalID: pulid.MustNew("ap_"),
		Tool:       f.tool,
		Params: services.ToolExecuteParams{
			OrganizationID: f.org,
			BusinessUnitID: f.bu,
			Params:         f.cancelParams(),
		},
	})
	require.NotNil(t, evaluation.Target)
	assert.Len(t, f.baselines.kept, 1, "an evaluation keeps nothing")
}

func TestBaseline_NeverFailsTheProposal(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.tool.failErr = errors.New("boom")

	result := f.svc.Baseline(t.Context(), &services.ProposalBaselineRequest{
		ProposalID: pulid.MustNew("ap_"),
		Tool:       f.tool,
		Params: services.ToolExecuteParams{
			OrganizationID: f.org,
			BusinessUnitID: f.bu,
			Params:         f.cancelParams(),
		},
		Persist: true,
	})

	require.NotNil(t, result.Target, "the pin survives a failed preview")
	assert.Nil(t, result.Preview)
	assert.Empty(t, f.baselines.kept)
}
