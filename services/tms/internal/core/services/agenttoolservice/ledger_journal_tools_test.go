package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/journalreversal"
	"github.com/emoss08/trenova/internal/core/domain/manualjournal"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	fuelAccountID    = pulid.MustNew("gla_")
	accruedAccountID = pulid.MustNew("gla_")
)

type fakeLedgerAccounts struct{}

func (fakeLedgerAccounts) GetByIDs(
	_ context.Context,
	req repositories.GetGLAccountsByIDsRequest,
) ([]*glaccount.GLAccount, error) {
	names := map[pulid.ID][2]string{
		fuelAccountID:    {"6100", "Fuel expense"},
		accruedAccountID: {"2100", "Accrued liabilities"},
	}
	out := make([]*glaccount.GLAccount, 0, len(req.GLAccountIDs))
	for _, id := range req.GLAccountIDs {
		if name, ok := names[id]; ok {
			out = append(out, &glaccount.GLAccount{ID: id, AccountCode: name[0], Name: name[1]})
		}
	}

	return out, nil
}

type fakeManualJournals struct {
	current   *manualjournal.Request
	refusal   error
	guard     *writeGuard
	created   *serviceports.CreateManualJournalRequest
	updated   *serviceports.UpdateManualJournalDraftRequest
	submitted bool
	cancelled *serviceports.CancelManualJournalRequest
	posted    bool
	approves  bool
}

func journalLines() []*manualjournal.Line {
	return []*manualjournal.Line{
		{GLAccountID: fuelAccountID, Description: "Fuel", DebitAmount: 2500},
		{GLAccountID: accruedAccountID, Description: "Accrual", CreditAmount: 2500},
	}
}

func draftJournal() *manualjournal.Request {
	entity := &manualjournal.Request{
		ID:                      pulid.MustNew("mjr_"),
		RequestNumber:           "MJR-7",
		Status:                  manualjournal.StatusDraft,
		Description:             "Accrue fuel",
		AccountingDate:          1_790_000_000,
		RequestedFiscalPeriodID: pulid.MustNew("fp_"),
		CurrencyCode:            "USD",
		Version:                 3,
		Lines:                   journalLines(),
	}
	entity.SyncTotals()

	return entity
}

func (f *fakeManualJournals) change(status manualjournal.Status) *serviceports.ManualJournalChange {
	before := *f.current
	after := *f.current
	after.Status = status

	return &serviceports.ManualJournalChange{Before: &before, After: &after}
}

func (f *fakeManualJournals) Get(
	context.Context,
	*serviceports.GetManualJournalRequest,
) (*manualjournal.Request, error) {
	copied := *f.current

	return &copied, nil
}

func (f *fakeManualJournals) PlanCreateDraft(
	_ context.Context,
	req *serviceports.CreateManualJournalRequest,
	_ *serviceports.RequestActor,
) (*manualjournal.Request, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}
	planned := &manualjournal.Request{
		Status:         manualjournal.StatusDraft,
		Description:    req.Description,
		AccountingDate: req.AccountingDate,
		CurrencyCode:   "USD",
	}
	for _, line := range req.Lines {
		planned.Lines = append(planned.Lines, &manualjournal.Line{
			GLAccountID:  line.GLAccountID,
			Description:  line.Description,
			DebitAmount:  line.DebitAmount,
			CreditAmount: line.CreditAmount,
		})
	}
	planned.SyncTotals()

	return planned, nil
}

func (f *fakeManualJournals) CreateDraft(
	ctx context.Context,
	req *serviceports.CreateManualJournalRequest,
	actor *serviceports.RequestActor,
) (*manualjournal.Request, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.created = req
	planned, err := f.PlanCreateDraft(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	planned.ID = pulid.MustNew("mjr_")
	planned.RequestNumber = "MJR-8"

	return planned, nil
}

func (f *fakeManualJournals) PlanUpdateDraft(
	_ context.Context,
	req *serviceports.UpdateManualJournalDraftRequest,
	_ *serviceports.RequestActor,
) (*serviceports.ManualJournalChange, error) {
	change := f.change(manualjournal.StatusDraft)
	change.After.Description = req.Description
	change.After.Lines = nil
	for _, line := range req.Lines {
		change.After.Lines = append(change.After.Lines, &manualjournal.Line{
			GLAccountID:  line.GLAccountID,
			Description:  line.Description,
			DebitAmount:  line.DebitAmount,
			CreditAmount: line.CreditAmount,
		})
	}
	change.After.SyncTotals()

	return change, nil
}

func (f *fakeManualJournals) UpdateDraft(
	_ context.Context,
	req *serviceports.UpdateManualJournalDraftRequest,
	_ *serviceports.RequestActor,
) (*manualjournal.Request, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.updated = req

	return f.current, nil
}

func (f *fakeManualJournals) PlanSubmit(
	context.Context,
	*serviceports.GetManualJournalRequest,
	*serviceports.RequestActor,
) (*serviceports.ManualJournalChange, error) {
	if f.approves {
		return f.change(manualjournal.StatusApproved), nil
	}

	return f.change(manualjournal.StatusPendingApproval), nil
}

func (f *fakeManualJournals) Submit(
	context.Context,
	*serviceports.GetManualJournalRequest,
	*serviceports.RequestActor,
) (*manualjournal.Request, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.submitted = true

	return f.current, nil
}

func (f *fakeManualJournals) PlanCancel(
	context.Context,
	*serviceports.CancelManualJournalRequest,
	*serviceports.RequestActor,
) (*serviceports.ManualJournalChange, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return f.change(manualjournal.StatusCancelled), nil
}

func (f *fakeManualJournals) Cancel(
	_ context.Context,
	req *serviceports.CancelManualJournalRequest,
	_ *serviceports.RequestActor,
) (*manualjournal.Request, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.cancelled = req

	return f.current, nil
}

func (f *fakeManualJournals) PlanPost(
	context.Context,
	*serviceports.GetManualJournalRequest,
	*serviceports.RequestActor,
) (*serviceports.ManualJournalChange, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}
	change := f.change(manualjournal.StatusPosted)
	change.Journal = &serviceports.JournalPreview{
		AccountingDate: f.current.AccountingDate + 86_400,
		FiscalPeriodID: pulid.MustNew("fp_"),
		EntryStatus:    "Posted",
		Lines: []serviceports.JournalLinePreview{
			{GLAccountID: fuelAccountID, DebitMinor: 2500},
			{GLAccountID: accruedAccountID, CreditMinor: 2500},
		},
	}

	return change, nil
}

func (f *fakeManualJournals) Post(
	context.Context,
	*serviceports.GetManualJournalRequest,
	*serviceports.RequestActor,
) (*manualjournal.Request, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.posted = true

	return f.current, nil
}

func draftParams() map[string]any {
	return map[string]any{
		paramJournalDescription: "Accrue September fuel",
		paramReason:             "Fuel card statement arrives after close",
		paramAccountingDate:     "2026-09-30",
		paramJournalLines: []any{
			map[string]any{
				paramGLAccountID:        fuelAccountID.String(),
				paramJournalDescription: "Fuel",
				paramJournalLineDebit:   "25.00",
			},
			map[string]any{
				paramGLAccountID:        accruedAccountID.String(),
				paramJournalDescription: "Accrual",
				paramJournalLineCredit:  "25.00",
			},
		},
	}
}

func TestDraftManualJournal_PreviewNamesTheAccountsAndDraftsWithoutPosting(t *testing.T) {
	t.Parallel()

	journals := &fakeManualJournals{guard: &writeGuard{}}
	tool := newDraftManualJournalTool(journals, fakeLedgerAccounts{})
	params := executeParams(draftParams())

	preview := previewWithoutWrites(t, journals.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})

	assert.Contains(t, preview.Summary, "Would draft the manual journal \"Accrue September fuel\"")
	assert.Contains(t, preview.Summary, "Its debits and credits balance.")
	change := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, change.Operation)
	assert.Equal(t, permission.ResourceManualJournal, change.Resource)
	require.NotNil(t, change.Money)
	require.Len(t, change.Money.Lines, 2)
	assert.Equal(t, "6100 Fuel expense (debit)", change.Money.Lines[0].Label)
	assert.Equal(t, "2100 Accrued liabilities (credit)", change.Money.Lines[1].Label)
	assert.True(t, change.Money.TotalAfter.Decimal.IsZero())

	result, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, journals.created)
	assert.Equal(t, int64(2500), journals.created.Lines[0].DebitAmount)
	assert.Equal(t, int64(2500), journals.created.Lines[1].CreditAmount)
	assert.Equal(t, "drafted", result.Action)
	assert.NotEmpty(t, result.IDs[paramManualJournalID])

	policy := tool.Policy()
	assert.Equal(t, permission.OpCreate, policy.Operation)
	assert.Equal(t, agent.TierActWithApproval, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress)
	require.NotNil(t, policy.TaintHold)
}

func TestDraftManualJournal_RefusesLinesItCannotBook(t *testing.T) {
	t.Parallel()

	tool := newDraftManualJournalTool(&fakeManualJournals{}, fakeLedgerAccounts{})
	both := draftParams()
	both[paramJournalLines].([]any)[0].(map[string]any)[paramJournalLineCredit] = "25.00"
	neither := draftParams()
	delete(neither[paramJournalLines].([]any)[1].(map[string]any), paramJournalLineCredit)
	single := draftParams()
	single[paramJournalLines] = single[paramJournalLines].([]any)[:1]
	negative := draftParams()
	negative[paramJournalLines].([]any)[0].(map[string]any)[paramJournalLineDebit] = "-5"
	badDate := draftParams()
	badDate[paramAccountingDate] = "30 Sept"

	for name, raw := range map[string]map[string]any{
		"a debit and a credit on one line": both,
		"a line with no amount":            neither,
		"one line":                         single,
		"a negative amount":                negative,
		"a bad date":                       badDate,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, tool.(serviceports.ToolValidator).Validate(
				t.Context(),
				executeParams(raw),
			))
		})
	}
}

func TestReviseManualJournalDraft_KeepsWhatItIsNotGiven(t *testing.T) {
	t.Parallel()

	journals := &fakeManualJournals{current: draftJournal(), guard: &writeGuard{}}
	tool := newReviseManualJournalDraftTool(journals, fakeLedgerAccounts{})
	params := executeParams(map[string]any{
		paramManualJournalID:    journals.current.ID.String(),
		paramJournalDescription: "Accrue fuel for September",
	})

	preview := previewWithoutWrites(t, journals.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would revise draft manual journal MJR-7")
	assert.Equal(t, "Accrue fuel for September",
		fieldByPath(t, previewChange(t, preview, 0), "description").After)

	_, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, journals.updated)
	assert.Equal(t, "Accrue fuel for September", journals.updated.Description)
	assert.Equal(t, journals.current.AccountingDate, journals.updated.AccountingDate)
	require.Len(t, journals.updated.Lines, 2)
	assert.Equal(t, fuelAccountID, journals.updated.Lines[0].GLAccountID)

	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceManualJournal, target.Resource)
}

func TestSubmitManualJournal_AnAgentNeverApprovesByTheBackDoor(t *testing.T) {
	t.Parallel()

	journals := &fakeManualJournals{current: draftJournal(), approves: true}
	tool := newSubmitManualJournalTool(journals, fakeLedgerAccounts{})
	raw := map[string]any{paramManualJournalID: journals.current.ID.String()}

	require.ErrorIs(t, tool.Execute(t.Context(), agentParamsFor(raw)), errAgentCannotApproveJournal)
	assert.False(t, journals.submitted)

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), executeParams(raw))
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "approval is turned off")

	require.NoError(t, tool.Execute(t.Context(), executeParams(raw)))
	assert.True(t, journals.submitted)
	assert.Equal(t, permission.OpSubmit, tool.Policy().Operation)
}

func TestSubmitManualJournal_AnAgentSubmitsForApproval(t *testing.T) {
	t.Parallel()

	journals := &fakeManualJournals{current: draftJournal(), guard: &writeGuard{}}
	tool := newSubmitManualJournalTool(journals, fakeLedgerAccounts{})
	params := agentParamsFor(map[string]any{paramManualJournalID: journals.current.ID.String()})

	preview := previewWithoutWrites(t, journals.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would submit manual journal MJR-7 for approval")
	assert.Equal(t, "PendingApproval", fieldByPath(t, previewChange(t, preview, 0), "status").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.True(t, journals.submitted)
}

func TestCancelManualJournal_NeedsAReason(t *testing.T) {
	t.Parallel()

	journals := &fakeManualJournals{current: draftJournal(), guard: &writeGuard{}}
	tool := newCancelManualJournalTool(journals, fakeLedgerAccounts{})

	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), executeParams(
		map[string]any{paramManualJournalID: journals.current.ID.String()},
	)))

	params := executeParams(map[string]any{
		paramManualJournalID: journals.current.ID.String(),
		paramReason:          "Entered twice",
	})
	preview := previewWithoutWrites(t, journals.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would cancel manual journal MJR-7 (now Draft)")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, journals.cancelled)
	assert.Equal(t, "Entered twice", journals.cancelled.Reason)
}

func TestPostManualJournal_OnlyAPersonPostsAndThePreviewShowsTheEntry(t *testing.T) {
	t.Parallel()

	approved := draftJournal()
	approved.Status = manualjournal.StatusApproved
	journals := &fakeManualJournals{current: approved, guard: &writeGuard{}}
	tool := newPostManualJournalTool(journals, fakeLedgerAccounts{})
	params := executeParams(map[string]any{paramManualJournalID: approved.ID.String()})

	preview := previewWithoutWrites(t, journals.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would post manual journal MJR-7 to the general ledger")
	assert.Contains(t, preview.Summary, "falls in a closed period")
	require.Len(t, preview.Changes, 2)
	assert.Equal(t, "Posted", fieldByPath(t, previewChange(t, preview, 0), "status").After)
	entry := previewChange(t, preview, 1)
	assert.Equal(t, permission.ResourceJournalEntry, entry.Resource)
	require.NotNil(t, entry.Money)
	assert.Equal(t, "6100 Fuel expense (debit)", entry.Money.Lines[0].Label)

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	assert.False(t, journals.posted)
	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.True(t, journals.posted)

	policy := tool.Policy()
	assert.Equal(t, permission.OpApprove, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)
}

func TestPostManualJournal_ARefusalIsAWouldFail(t *testing.T) {
	t.Parallel()

	journals := &fakeManualJournals{
		current: draftJournal(),
		refusal: errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Only approved manual journals can be posted",
		),
	}
	tool := newPostManualJournalTool(journals, fakeLedgerAccounts{})
	params := executeParams(map[string]any{paramManualJournalID: journals.current.ID.String()})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}

type fakeJournalReversals struct {
	change    *serviceports.JournalReversalChange
	guard     *writeGuard
	created   *serviceports.CreateJournalReversalRequest
	cancelled *serviceports.CancelJournalReversalRequest
	posted    bool
}

func reversalChange(from, to journalreversal.Status) *serviceports.JournalReversalChange {
	entry := &journalentry.JournalEntry{
		ID:          pulid.MustNew("je_"),
		EntryNumber: "JE-104",
		Status:      journalentry.StatusPosted,
		Version:     2,
	}
	before := &journalreversal.Reversal{
		ID:                      pulid.MustNew("jrev_"),
		OriginalJournalEntryID:  entry.ID,
		Status:                  from,
		RequestedAccountingDate: 1_790_000_000,
		ResolvedFiscalPeriodID:  pulid.MustNew("fp_"),
		ReasonCode:              "WrongAccount",
		ReasonText:              "Posted to fuel instead of tolls",
		Version:                 1,
	}
	after := *before
	after.Status = to

	return &serviceports.JournalReversalChange{
		Before:        before,
		After:         &after,
		OriginalEntry: entry,
		CurrencyCode:  "USD",
		Journal: &serviceports.JournalPreview{
			AccountingDate: before.RequestedAccountingDate,
			FiscalPeriodID: before.ResolvedFiscalPeriodID,
			EntryStatus:    "Posted",
			Lines: []serviceports.JournalLinePreview{
				{GLAccountID: fuelAccountID, CreditMinor: 2500},
				{GLAccountID: accruedAccountID, DebitMinor: 2500},
			},
		},
	}
}

func (f *fakeJournalReversals) PlanCreate(
	context.Context,
	*serviceports.CreateJournalReversalRequest,
	*serviceports.RequestActor,
) (*serviceports.JournalReversalChange, error) {
	change := *f.change
	change.Before = nil
	after := *f.change.After
	after.ID = pulid.Nil
	change.After = &after

	return &change, nil
}

func (f *fakeJournalReversals) Create(
	_ context.Context,
	req *serviceports.CreateJournalReversalRequest,
	_ *serviceports.RequestActor,
) (*journalreversal.Reversal, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.created = req

	return f.change.After, nil
}

func (f *fakeJournalReversals) PlanCancel(
	context.Context,
	*serviceports.CancelJournalReversalRequest,
	*serviceports.RequestActor,
) (*serviceports.JournalReversalChange, error) {
	return f.change, nil
}

func (f *fakeJournalReversals) Cancel(
	_ context.Context,
	req *serviceports.CancelJournalReversalRequest,
	_ *serviceports.RequestActor,
) (*journalreversal.Reversal, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.cancelled = req

	return f.change.After, nil
}

func (f *fakeJournalReversals) PlanPost(
	context.Context,
	*serviceports.GetJournalReversalRequest,
	*serviceports.RequestActor,
) (*serviceports.JournalReversalChange, error) {
	return f.change, nil
}

func (f *fakeJournalReversals) Post(
	context.Context,
	*serviceports.GetJournalReversalRequest,
	*serviceports.RequestActor,
) (*journalreversal.Reversal, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.posted = true

	return f.change.After, nil
}

func TestRequestJournalReversal_OnlyAPersonRequestsOne(t *testing.T) {
	t.Parallel()

	reversals := &fakeJournalReversals{
		change: reversalChange(journalreversal.StatusApproved, journalreversal.StatusApproved),
		guard:  &writeGuard{},
	}
	tool := newRequestJournalReversalTool(reversals, fakeLedgerAccounts{})
	params := executeParams(map[string]any{
		paramJournalEntryID:     reversals.change.OriginalEntry.ID.String(),
		paramAccountingDate:     "2026-09-21",
		paramReversalReasonCode: "WrongAccount",
		paramReversalReasonText: "Posted to fuel instead of tolls",
	})

	preview := previewWithoutWrites(t, reversals.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would request a reversal of journal entry JE-104")
	assert.Contains(t, preview.Summary, "approved at once")
	change := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, change.Operation)
	require.NotNil(t, change.Money)
	assert.Equal(t, "6100 Fuel expense (credit)", change.Money.Lines[0].Label)

	_, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.ErrorIs(t, err, ErrNeedsAPersonsApproval)
	params.ProposalID = pulid.MustNew("ap_")
	result, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, reversals.created)
	assert.Equal(t, "requested", result.Action)

	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceJournalEntry, target.Resource)
	assert.Equal(t, permission.OpCreate, tool.Policy().Operation)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)
}

func TestCancelJournalReversal_KeepsTheOriginalEntry(t *testing.T) {
	t.Parallel()

	reversals := &fakeJournalReversals{
		change: reversalChange(
			journalreversal.StatusPendingApproval,
			journalreversal.StatusCancelled,
		),
		guard: &writeGuard{},
	}
	tool := newCancelJournalReversalTool(reversals, fakeLedgerAccounts{})
	params := executeParams(map[string]any{
		paramJournalReversalID: reversals.change.Before.ID.String(),
		paramReason:            "The entry was right after all",
	})

	preview := previewWithoutWrites(t, reversals.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "the original entry stands as posted")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, reversals.cancelled)
	assert.Equal(t, agent.TierActWithApproval, tool.Policy().MaxTier)
}

func TestPostJournalReversal_ShowsTheEntryAndTheOriginalReversed(t *testing.T) {
	t.Parallel()

	reversals := &fakeJournalReversals{
		change: reversalChange(journalreversal.StatusApproved, journalreversal.StatusPosted),
		guard:  &writeGuard{},
	}
	tool := newPostJournalReversalTool(reversals, fakeLedgerAccounts{})
	params := executeParams(map[string]any{
		paramJournalReversalID: reversals.change.Before.ID.String(),
	})

	preview := previewWithoutWrites(t, reversals.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	require.Len(t, preview.Changes, 3)
	assert.Equal(t, permission.ResourceJournalEntry, previewChange(t, preview, 1).Resource)
	assert.Equal(t, "Reversed", fieldByPath(t, previewChange(t, preview, 2), "status").After)

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.True(t, reversals.posted)
	assert.Equal(t, permission.OpApprove, tool.Policy().Operation)
}
