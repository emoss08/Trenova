package receivabletools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/app/fiscalperiodservice"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptworkitemports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFiscalPeriods struct {
	period    *fiscalperiod.FiscalPeriod
	refusal   error
	guard     *agenttoolstest.WriteGuard
	requested *fiscalperiodservice.TransitionRequest
}

func (f *fakeFiscalPeriods) PlanTransition(
	_ context.Context,
	req *fiscalperiodservice.TransitionRequest,
	_ pulid.ID,
) (*fiscalperiodservice.TransitionPlan, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}
	after := *f.period
	switch req.Transition {
	case fiscalperiodservice.TransitionClose:
		after.Status = fiscalperiod.StatusClosed
	case fiscalperiodservice.TransitionLock:
		after.Status = fiscalperiod.StatusLocked
	case fiscalperiodservice.TransitionReopen:
		after.Status = fiscalperiod.StatusOpen
		after.ReopenReason = req.Reason
	case fiscalperiodservice.TransitionUnlock, fiscalperiodservice.TransitionActivate:
		after.Status = fiscalperiod.StatusOpen
	}

	return &fiscalperiodservice.TransitionPlan{Before: f.period, After: &after}, nil
}

func (f *fakeFiscalPeriods) Transition(
	_ context.Context,
	req *fiscalperiodservice.TransitionRequest,
	_ pulid.ID,
) (*fiscalperiod.FiscalPeriod, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.requested = req

	return f.period, nil
}

func septemberPeriod(status fiscalperiod.Status) *fiscalperiod.FiscalPeriod {
	return &fiscalperiod.FiscalPeriod{
		ID:        pulid.MustNew("fp_"),
		Name:      "September 2026",
		Status:    status,
		StartDate: 1_788_220_800,
		EndDate:   1_790_812_799,
		Version:   4,
	}
}

func TestFiscalPeriodTools_EachIsAPersonsDecisionUnderItsOwnPermission(t *testing.T) {
	t.Parallel()

	cases := []struct {
		move      *fiscalPeriodMove
		from      fiscalperiod.Status
		operation permission.Operation
		want      string
	}{
		{closeFiscalPeriodMove(), fiscalperiod.StatusLocked, permission.OpClose, "Closed"},
		{lockFiscalPeriodMove(), fiscalperiod.StatusOpen, permission.OpLock, "Locked"},
		{unlockFiscalPeriodMove(), fiscalperiod.StatusLocked, permission.OpUnlock, "Open"},
		{reopenFiscalPeriodMove(), fiscalperiod.StatusClosed, permission.OpReopen, "Open"},
		{openFiscalPeriodMove(), fiscalperiod.StatusInactive, permission.OpActivate, "Open"},
	}
	for _, tc := range cases {
		t.Run(tc.move.name, func(t *testing.T) {
			t.Parallel()

			periods := &fakeFiscalPeriods{
				period: septemberPeriod(tc.from),
				guard:  &agenttoolstest.WriteGuard{},
			}
			tool := newFiscalPeriodTool(periods, tc.move)
			params := agenttoolstest.ExecuteParams(map[string]any{
				paramFiscalPeriodID: periods.period.ID.String(),
				common.ParamReason:  "A late fuel invoice belongs in September",
			})

			preview := agenttoolstest.PreviewWithoutWrites(
				t,
				periods.guard,
				func() (*agent.ToolPreview, error) {
					return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
				},
			)
			assert.Contains(t, preview.Summary, "September 2026")
			assert.Equal(
				t,
				tc.want,
				agenttoolstest.FieldByPath(
					t,
					agenttoolstest.PreviewChange(t, preview, 0),
					"status",
				).After,
			)

			require.ErrorIs(t, tool.Execute(t.Context(), params), common.ErrNeedsAPersonsApproval)
			params.ProposalID = pulid.MustNew("ap_")
			require.NoError(t, tool.Execute(t.Context(), params))
			require.NotNil(t, periods.requested)
			assert.Equal(t, tc.move.transition, periods.requested.Transition)

			policy := tool.Policy()
			assert.Equal(t, permission.ResourceFiscalPeriod, policy.Resource)
			assert.Equal(t, tc.operation, policy.Operation)
			assert.Equal(t, agent.TierPropose, policy.MaxTier)
			assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)
			target, ok := tool.(agenttoolports.TargetedTool).Target(params.Params)
			require.True(t, ok)
			assert.Equal(t, permission.ResourceFiscalPeriod, target.Resource)
		})
	}
}

func TestReopenFiscalPeriod_NeedsAReason(t *testing.T) {
	t.Parallel()

	periods := &fakeFiscalPeriods{period: septemberPeriod(fiscalperiod.StatusClosed)}
	tool := newFiscalPeriodTool(periods, reopenFiscalPeriodMove())

	require.Error(
		t,
		tool.(agenttoolports.ToolValidator).Validate(t.Context(), agenttoolstest.ExecuteParams(
			map[string]any{paramFiscalPeriodID: periods.period.ID.String()},
		)),
	)
	assert.Contains(t, tool.ParamSchema()[toolschema.KeyRequired], common.ParamReason)
}

func TestCloseFiscalPeriod_ABlockedCloseIsAWouldFail(t *testing.T) {
	t.Parallel()

	periods := &fakeFiscalPeriods{
		period: septemberPeriod(fiscalperiod.StatusOpen),
		refusal: errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Cannot close period 9: period 8 is still open. Close periods sequentially.",
		),
	}
	tool := newFiscalPeriodTool(periods, closeFiscalPeriodMove())
	params := agenttoolstest.ExecuteParams(
		map[string]any{paramFiscalPeriodID: periods.period.ID.String()},
	)

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
}

type fakeMappingReviewer struct {
	rows      []*accountingsync.AccountingMapping
	guard     *agenttoolstest.WriteGuard
	confirmed *accountingmappingports.ConfirmAccountingMappingsRequest
	rejected  *accountingmappingports.AccountingMappingActionRequest
}

func proposedMapping(label, externalID, name string) *accountingsync.AccountingMapping {
	return &accountingsync.AccountingMapping{
		ID:           pulid.MustNew("acctm_"),
		TargetLabel:  label,
		ExternalID:   externalID,
		ExternalName: name,
		State:        accountingsync.MappingStateProposed,
		Source:       accountingsync.MappingSourceSuggested,
		Version:      2,
	}
}

func (f *fakeMappingReviewer) PlanConfirm(
	_ context.Context,
	req *accountingmappingports.ConfirmAccountingMappingsRequest,
) (*accountingmappingports.AccountingMappingConfirmPlan, error) {
	plan := &accountingmappingports.AccountingMappingConfirmPlan{}
	for _, row := range f.rows {
		after := row.Clone()
		after.Confirm(&accountingsync.Choice{
			ExternalID:   row.ExternalID,
			ExternalName: row.ExternalName,
			Source:       req.Source,
			ActorID:      req.UserID,
			At:           1_790_000_000,
		})
		plan.Changes = append(plan.Changes, &accountingmappingports.AccountingMappingChange{
			Before: row,
			After:  after,
		})
		plan.Confirmed = append(plan.Confirmed, after)
	}

	return plan, nil
}

func (f *fakeMappingReviewer) Confirm(
	_ context.Context,
	req *accountingmappingports.ConfirmAccountingMappingsRequest,
) ([]*accountingsync.AccountingMapping, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.confirmed = req

	return f.rows, nil
}

func (f *fakeMappingReviewer) PlanReject(
	context.Context,
	*accountingmappingports.AccountingMappingActionRequest,
) (*accountingmappingports.AccountingMappingChange, error) {
	after := f.rows[0].Clone()
	if err := after.Reject(); err != nil {
		return nil, err
	}

	return &accountingmappingports.AccountingMappingChange{Before: f.rows[0], After: after}, nil
}

func (f *fakeMappingReviewer) Reject(
	_ context.Context,
	req *accountingmappingports.AccountingMappingActionRequest,
) (*accountingsync.AccountingMapping, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.rejected = req

	return f.rows[0], nil
}

func TestConfirmAccountingMappingProposals_ConfirmsWhatWasShownAsTheAgent(t *testing.T) {
	t.Parallel()

	ar := proposedMapping("Accounts receivable", "1", "Accounts Receivable (A/R)")
	fuel := proposedMapping("Fuel surcharge", "91", "Fuel Surcharge")
	reviewer := &fakeMappingReviewer{rows: []*accountingsync.AccountingMapping{ar, fuel},
		guard: &agenttoolstest.WriteGuard{}}
	tool := accountingtools.NewConfirmAccountingMappingProposalsTool(reviewer)
	params := agenttoolstest.ExecuteParams(
		map[string]any{accountingtools.ParamMappingProposals: []any{
			map[string]any{
				accountingtools.ParamMappingID:  ar.ID.String(),
				accountingtools.ParamExternalID: "1",
			},
			map[string]any{
				accountingtools.ParamMappingID:  fuel.ID.String(),
				accountingtools.ParamExternalID: "91",
			},
		}},
	)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		reviewer.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "Would confirm 2 accounting mappings as proposed")
	require.Len(t, preview.Changes, 2)
	assert.Equal(
		t,
		"Confirmed",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), "state").After,
	)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, reviewer.confirmed)
	assert.Equal(t, accountingsync.MappingSourceAgent, reviewer.confirmed.Source)
	require.Len(t, reviewer.confirmed.Items, 2)
	assert.Equal(t, "91", reviewer.confirmed.Items[1].ExternalID)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceAccountingIntegration, policy.Resource)
	assert.Equal(t, permission.OpUpdate, policy.Operation)
	assert.Equal(t, agent.TierActWithApproval, policy.MaxTier)
}

func TestConfirmAccountingMappingProposals_RefusesARepeatedMapping(t *testing.T) {
	t.Parallel()

	ar := proposedMapping("Accounts receivable", "1", "A/R")
	tool := accountingtools.NewConfirmAccountingMappingProposalsTool(&fakeMappingReviewer{
		rows: []*accountingsync.AccountingMapping{ar},
	})

	require.Error(
		t,
		tool.(agenttoolports.ToolValidator).Validate(t.Context(), agenttoolstest.ExecuteParams(
			map[string]any{accountingtools.ParamMappingProposals: []any{
				map[string]any{
					accountingtools.ParamMappingID:  ar.ID.String(),
					accountingtools.ParamExternalID: "1",
				},
				map[string]any{
					accountingtools.ParamMappingID:  ar.ID.String(),
					accountingtools.ParamExternalID: "1",
				},
			}},
		)),
	)
}

func TestRejectAccountingMappingProposal_UnmatchesItAsTheAgent(t *testing.T) {
	t.Parallel()

	customer := proposedMapping("Acme Freight", "50", "Acme Freight LLC")
	reviewer := &fakeMappingReviewer{rows: []*accountingsync.AccountingMapping{customer},
		guard: &agenttoolstest.WriteGuard{}}
	tool := accountingtools.NewRejectAccountingMappingProposalTool(reviewer)
	params := agenttoolstest.ExecuteParams(
		map[string]any{accountingtools.ParamMappingID: customer.ID.String()},
	)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		reviewer.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "Would turn down Acme Freight LLC for Acme Freight")
	assert.Equal(
		t,
		"Unmatched",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), "state").After,
	)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, reviewer.rejected)
	assert.Equal(t, accountingsync.MappingSourceAgent, reviewer.rejected.Source)
}

type fakeSyncReleaser struct {
	records  map[pulid.ID]*accountingsync.AccountingSyncRecord
	backfill *accountingsync.AccountingBackfill
	guard    *agenttoolstest.WriteGuard
	released *accountingsyncports.ReleaseAccountingSyncRequest
	changed  *accountingsyncports.ChangeAccountingBackfillRequest
}

func (f *fakeSyncReleaser) Summary(
	context.Context,
	pagination.TenantInfo,
	integration.Type,
) (*accountingsyncports.AccountingSyncSummary, error) {
	start := int64(1_780_000_000)
	return &accountingsyncports.AccountingSyncSummary{
		ProviderName: "QuickBooks Online",
		Connection: &accountingsync.AccountingConnection{
			ID:            pulid.MustNew("acctc_"),
			Status:        accountingsync.ConnectionStatusConnected,
			SetupStep:     accountingsync.SetupStepComplete,
			SyncStartDate: &start,
			SyncEnabledAt: &start,
		},
		ActiveBackfill: f.backfill,
	}, nil
}

func (f *fakeSyncReleaser) GetRecord(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*accountingsync.AccountingSyncRecord, error) {
	record, ok := f.records[id]
	if !ok {
		return nil, errortypes.NewNotFoundError("Accounting sync record not found")
	}
	copied := *record

	return &copied, nil
}

func (f *fakeSyncReleaser) Release(
	_ context.Context,
	req *accountingsyncports.ReleaseAccountingSyncRequest,
) (int64, error) {
	if err := f.guard.Write(); err != nil {
		return 0, err
	}
	f.released = req

	return int64(len(req.IDs)), nil
}

func (f *fakeSyncReleaser) ChangeBackfill(
	_ context.Context,
	req *accountingsyncports.ChangeAccountingBackfillRequest,
) (*accountingsync.AccountingBackfill, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.changed = req

	return f.backfill, nil
}

func heldRecord(status accountingsync.SyncStatus) *accountingsync.AccountingSyncRecord {
	return &accountingsync.AccountingSyncRecord{
		ID:           pulid.MustNew("asr_"),
		ObjectType:   accountingsync.SyncObjectInvoice,
		ObjectNumber: "INV-1001",
		Status:       status,
		Version:      5,
	}
}

func TestReleaseAccountingSync_OnlyAPersonReleasesHeldDocuments(t *testing.T) {
	t.Parallel()

	held := heldRecord(accountingsync.SyncStatusAwaitingApproval)
	sync := &fakeSyncReleaser{
		records: map[pulid.ID]*accountingsync.AccountingSyncRecord{held.ID: held},
		guard:   &agenttoolstest.WriteGuard{},
	}
	tool := accountingtools.NewReleaseAccountingSyncTool(sync)
	params := agenttoolstest.ExecuteParams(map[string]any{
		accountingtools.ParamAccountingSystem: string(integration.TypeQuickBooksOnline),
		accountingtools.ParamSyncRecordIDs:    []any{held.ID.String()},
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		sync.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "Would release 1 document held for approval")
	assert.Equal(
		t,
		"Queued",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), "status").After,
	)

	require.ErrorIs(t, tool.Execute(t.Context(), params), common.ErrNeedsAPersonsApproval)
	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, sync.released)
	assert.Equal(t, []pulid.ID{held.ID}, sync.released.IDs)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceAccountingSync, policy.Resource)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)
	subset := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)[accountingtools.ParamSyncRecordIDs]
	assert.Equal(t, permission.ResourceAccountingSync.String(),
		toolschema.SubsetResource(subset.(map[string]any)))
}

func TestReleaseAccountingSync_RefusesADocumentThatIsNotHeld(t *testing.T) {
	t.Parallel()

	failed := heldRecord(accountingsync.SyncStatusDeadLettered)
	tool := accountingtools.NewReleaseAccountingSyncTool(&fakeSyncReleaser{
		records: map[pulid.ID]*accountingsync.AccountingSyncRecord{failed.ID: failed},
	})

	err := tool.(agenttoolports.ToolValidator).Validate(
		t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			accountingtools.ParamAccountingSystem: string(integration.TypeQuickBooksOnline),
			accountingtools.ParamSyncRecordIDs:    []any{failed.ID.String()},
		}),
	)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestChangeAccountingBackfill_ActsOnTheBackfillInProgress(t *testing.T) {
	t.Parallel()

	backfill := &accountingsync.AccountingBackfill{
		ID:     pulid.MustNew("acctbf_"),
		Status: accountingsync.BackfillStatusRunning,
	}
	sync := &fakeSyncReleaser{backfill: backfill, guard: &agenttoolstest.WriteGuard{}}
	tool := accountingtools.NewChangeAccountingBackfillTool(sync)
	params := agenttoolstest.PersonParams(map[string]any{
		accountingtools.ParamAccountingSystem: string(integration.TypeQuickBooksOnline),
		accountingtools.ParamBackfillAction:   string(accountingsyncports.AccountingBackfillPause),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		sync.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "Would pause the QuickBooks Online backfill (now running)")
	assert.Equal(
		t,
		"Paused",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), "status").After,
	)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, sync.changed)
	assert.Equal(t, backfill.ID, sync.changed.ID)
	assert.Equal(t, permission.OpManage, tool.Policy().Operation)
}

func TestChangeAccountingBackfill_RefusesAMoveTheBackfillCannotMake(t *testing.T) {
	t.Parallel()

	sync := &fakeSyncReleaser{backfill: &accountingsync.AccountingBackfill{
		ID:     pulid.MustNew("acctbf_"),
		Status: accountingsync.BackfillStatusRunning,
	}}
	tool := accountingtools.NewChangeAccountingBackfillTool(sync)

	resume := agenttoolstest.PersonParams(map[string]any{
		accountingtools.ParamAccountingSystem: string(integration.TypeQuickBooksOnline),
		accountingtools.ParamBackfillAction:   string(accountingsyncports.AccountingBackfillResume),
	})
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), resume))

	agentCall := agenttoolstest.AgentParamsFor(map[string]any{
		accountingtools.ParamAccountingSystem: string(integration.TypeQuickBooksOnline),
		accountingtools.ParamBackfillAction:   string(accountingsyncports.AccountingBackfillCancel),
	})
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), agentCall))
}

type fakeWorkItemRouter struct {
	item     *bankreceiptworkitem.WorkItem
	guard    *agenttoolstest.WriteGuard
	assigned *bankreceiptworkitemports.AssignBankReceiptWorkItemRequest
	reviewed bool
}

func (f *fakeWorkItemRouter) change(
	status bankreceiptworkitem.Status,
	assignee pulid.ID,
) *bankreceiptworkitemports.BankReceiptWorkItemChange {
	after := *f.item
	after.Status = status
	if assignee.IsNotNil() {
		after.AssignedToUserID = assignee
	}

	return &bankreceiptworkitemports.BankReceiptWorkItemChange{Before: f.item, After: &after}
}

func (f *fakeWorkItemRouter) PlanAssign(
	_ context.Context,
	req *bankreceiptworkitemports.AssignBankReceiptWorkItemRequest,
	_ *permissionports.RequestActor,
) (*bankreceiptworkitemports.BankReceiptWorkItemChange, error) {
	return f.change(bankreceiptworkitem.StatusAssigned, req.AssignedToUserID), nil
}

func (f *fakeWorkItemRouter) Assign(
	_ context.Context,
	req *bankreceiptworkitemports.AssignBankReceiptWorkItemRequest,
	_ *permissionports.RequestActor,
) (*bankreceiptworkitem.WorkItem, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.assigned = req

	return f.item, nil
}

func (f *fakeWorkItemRouter) PlanStartReview(
	context.Context,
	*bankreceiptworkitemports.GetBankReceiptWorkItemRequest,
	*permissionports.RequestActor,
) (*bankreceiptworkitemports.BankReceiptWorkItemChange, error) {
	return f.change(bankreceiptworkitem.StatusInReview, pulid.Nil), nil
}

func (f *fakeWorkItemRouter) StartReview(
	context.Context,
	*bankreceiptworkitemports.GetBankReceiptWorkItemRequest,
	*permissionports.RequestActor,
) (*bankreceiptworkitem.WorkItem, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.reviewed = true

	return f.item, nil
}

func TestTriageBankReceiptWorkItem_AssignsOrMarksItUnderReview(t *testing.T) {
	t.Parallel()

	items := &fakeWorkItemRouter{
		item: &bankreceiptworkitem.WorkItem{
			ID:      pulid.MustNew("brwi_"),
			Status:  bankreceiptworkitem.StatusOpen,
			Version: 1,
		},
		guard: &agenttoolstest.WriteGuard{},
	}
	tool := newTriageBankReceiptWorkItemTool(items)
	assignee := pulid.MustNew("usr_")
	assign := agenttoolstest.AgentParamsFor(map[string]any{
		paramWorkItemID:   items.item.ID.String(),
		paramWorkItemMove: string(workItemAssign),
		paramAssigneeID:   assignee.String(),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		items.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), assign)
		},
	)
	assert.Contains(t, preview.Summary, "Would assign the reconciliation item")
	assert.Equal(
		t,
		"Assigned",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), "status").After,
	)
	require.NoError(t, tool.Execute(t.Context(), assign))
	require.NotNil(t, items.assigned)
	assert.Equal(t, assignee, items.assigned.AssignedToUserID)

	review := agenttoolstest.AgentParamsFor(map[string]any{
		paramWorkItemID:   items.item.ID.String(),
		paramWorkItemMove: string(workItemStartReview),
	})
	require.NoError(t, tool.Execute(t.Context(), review))
	assert.True(t, items.reviewed)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceBankReceiptWorkItem, policy.Resource)
	assert.Equal(t, agent.TierAutoExecute, policy.MaxTier)
}

func TestTriageBankReceiptWorkItem_RefusesAMismatchedAssignee(t *testing.T) {
	t.Parallel()

	tool := newTriageBankReceiptWorkItemTool(&fakeWorkItemRouter{})
	id := pulid.MustNew("brwi_").String()

	for name, raw := range map[string]map[string]any{
		"assign without anyone": {
			paramWorkItemID:   id,
			paramWorkItemMove: string(workItemAssign),
		},
		"review with someone": {
			paramWorkItemID:   id,
			paramWorkItemMove: string(workItemStartReview),
			paramAssigneeID:   pulid.MustNew("usr_").String(),
		},
		"an unknown action": {
			paramWorkItemID:   id,
			paramWorkItemMove: "Resolve",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, tool.(agenttoolports.ToolValidator).Validate(
				t.Context(),
				agenttoolstest.ExecuteParams(raw),
			))
		})
	}
}
