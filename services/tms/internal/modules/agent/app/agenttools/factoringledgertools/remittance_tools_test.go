package factoringledgertools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/factoringkit"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringremittanceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRemittances struct {
	factoringremittanceports.RemittanceService

	guard     agenttoolstest.WriteGuard
	ledger    *fakeLedger
	rem       *factoringremittance.Remittance
	line      *factoringremittance.RemittanceLine
	updated   *factoringremittanceports.UpdateRemittanceLineRequest
	applied   *factoringremittanceports.RemittanceVersionRequest
	rematched *factoringremittanceports.RemittanceVersionRequest
	discarded *factoringremittanceports.DiscardRemittanceRequest
}

func newFakeRemittances() *fakeRemittances {
	ledger := newFakeLedger()
	line := &factoringremittance.RemittanceLine{
		ID:            pulid.MustNew("facrml_"),
		RowNumber:     4,
		Kind:          factoringremittance.RemittanceLineFunding,
		InvoiceNumber: "INV-7",
		MatchStatus:   factoringremittance.RemittanceLineSuggested,
	}
	return &fakeRemittances{
		ledger: ledger,
		line:   line,
		rem: &factoringremittance.Remittance{
			ID:             pulid.MustNew("facrem_"),
			FileName:       "rts-funding.csv",
			Status:         factoringremittance.RemittanceStatusParsed,
			CurrencyCode:   "USD",
			LineCount:      2,
			MatchedCount:   1,
			ExceptionCount: 1,
			Version:        5,
			Lines:          []*factoringremittance.RemittanceLine{line},
		},
	}
}

func (f *fakeRemittances) check(id pulid.ID, version int64) error {
	if id != f.rem.ID {
		return errortypes.NewNotFoundError("Factoring remittance not found")
	}
	if version != f.rem.Version {
		return errortypes.NewConflictError("Remittance changed since you opened it")
	}
	return nil
}

func (f *fakeRemittances) Get(
	_ context.Context,
	req factoringremittanceports.GetRemittanceRequest,
) (*factoringremittance.Remittance, error) {
	if req.ID != f.rem.ID {
		return nil, errortypes.NewNotFoundError("Factoring remittance not found")
	}
	return f.rem, nil
}

func (f *fakeRemittances) PlanUpdateLine(
	_ context.Context,
	req *factoringremittanceports.UpdateRemittanceLineRequest,
	_ *permissionports.RequestActor,
) (*factoringremittanceports.RemittanceLinePlan, error) {
	if err := f.check(req.RemittanceID, req.Version); err != nil {
		return nil, err
	}
	after := *f.line
	after.MatchStatus = factoringremittance.RemittanceLineMatched
	after.SubmissionItemID = req.ItemID
	afterRem := *f.rem
	afterRem.MatchedCount, afterRem.ExceptionCount = 2, 0
	return &factoringremittanceports.RemittanceLinePlan{
		Before:     f.rem,
		After:      &afterRem,
		LineBefore: f.line,
		LineAfter:  &after,
		Comment:    "Row 4 matched to invoice INV-7",
	}, nil
}

func (f *fakeRemittances) UpdateLine(
	_ context.Context,
	req *factoringremittanceports.UpdateRemittanceLineRequest,
	_ *permissionports.RequestActor,
) (*factoringremittance.Remittance, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.updated = req
	return f.rem, nil
}

func (f *fakeRemittances) Rematch(
	_ context.Context,
	req *factoringremittanceports.RemittanceVersionRequest,
	_ *permissionports.RequestActor,
) (*factoringremittance.Remittance, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.rematched = req
	return f.rem, nil
}

func (f *fakeRemittances) PlanApply(
	_ context.Context,
	req *factoringremittanceports.RemittanceVersionRequest,
	_ *permissionports.RequestActor,
) (*factoringremittanceports.RemittancePlan, error) {
	if err := f.check(req.ID, req.Version); err != nil {
		return nil, err
	}
	return &factoringremittanceports.RemittancePlan{
		Remittance: f.rem,
		Lines: []*factoringremittanceports.PlannedRemittanceLine{{
			LineID:        f.line.ID,
			RowNumber:     f.line.RowNumber,
			Kind:          factoringremittance.RemittanceLineFunding,
			InvoiceNumber: "INV-7",
			Movement:      f.ledger.planned(factoringledger.ReserveEntryFunding),
		}},
		FundedCount:   1,
		LeftOpenCount: 1,
		AdvanceMinor:  88_000,
		FeeMinor:      2_500,
	}, nil
}

func (f *fakeRemittances) Apply(
	_ context.Context,
	req *factoringremittanceports.RemittanceVersionRequest,
	_ *permissionports.RequestActor,
) (*factoringremittanceports.ApplyRemittanceResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.applied = req
	return &factoringremittanceports.ApplyRemittanceResult{Remittance: f.rem}, nil
}

func (f *fakeRemittances) PlanDiscard(
	_ context.Context,
	req *factoringremittanceports.DiscardRemittanceRequest,
	_ *permissionports.RequestActor,
) (*factoringremittanceports.RemittanceDiscardPlan, error) {
	if err := f.check(req.ID, req.Version); err != nil {
		return nil, err
	}
	after := *f.rem
	after.Status = factoringremittance.RemittanceStatusDiscarded
	after.DiscardReason = req.Reason
	return &factoringremittanceports.RemittanceDiscardPlan{Before: f.rem, After: &after}, nil
}

func (f *fakeRemittances) Discard(
	_ context.Context,
	req *factoringremittanceports.DiscardRemittanceRequest,
	_ *permissionports.RequestActor,
) (*factoringremittance.Remittance, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.discarded = req
	return f.rem, nil
}

func TestRemittanceToolPolicies(t *testing.T) {
	t.Parallel()

	remittances := newFakeRemittances()
	apply := NewApplyFactoringRemittanceTool(remittances, remittances.ledger, nil).Policy()
	assert.Equal(t, permission.ResourceFactoringRemittance, apply.Resource)
	assert.Equal(t, permission.OpImport, apply.Operation)
	assert.Equal(t, agent.TierPropose, apply.MaxTier, "booking a remittance is a person's")
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, apply.Egress)

	for _, tool := range []agenttoolports.AgentTool{
		NewResolveFactoringRemittanceLineTool(remittances),
		NewRematchFactoringRemittanceTool(remittances),
		NewDiscardFactoringRemittanceTool(remittances),
	} {
		policy := tool.Policy()
		assert.Equal(t, permission.ResourceFactoringRemittance, policy.Resource, tool.Name())
		assert.Equal(t, agent.TierPropose, policy.DefaultTier, tool.Name())
		assert.Equal(t, agent.TierActWithApproval, policy.MaxTier, tool.Name())
		assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress, tool.Name())
	}
}

func TestResolveFactoringRemittanceLine_PreviewsTheMatch(t *testing.T) {
	t.Parallel()

	remittances := newFakeRemittances()
	tool := NewResolveFactoringRemittanceLineTool(remittances)
	itemID := remittances.ledger.item.ID
	params := agenttoolstest.PersonCallParams(map[string]any{
		paramRemittanceID:         remittances.rem.ID.String(),
		factoringkit.ParamVersion: float64(remittances.rem.Version),
		paramLineID:               remittances.line.ID.String(),
		paramLineAction:           string(factoringremittanceports.RemittanceLineActionMatch),
		paramItemID:               itemID.String(),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&remittances.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	require.Len(t, preview.Changes, 2)
	assert.Contains(t, preview.Summary, "Row 4 matched to invoice INV-7")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, remittances.updated)
	assert.Equal(t, itemID, remittances.updated.ItemID)
	assert.Equal(t, remittances.line.ID, remittances.updated.LineID)

	missingItem := agenttoolstest.PersonCallParams(map[string]any{
		paramRemittanceID:         remittances.rem.ID.String(),
		factoringkit.ParamVersion: float64(remittances.rem.Version),
		paramLineID:               remittances.line.ID.String(),
		paramLineAction:           string(factoringremittanceports.RemittanceLineActionMatch),
	})
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), missingItem))

	badAction := agenttoolstest.PersonCallParams(map[string]any{
		paramRemittanceID:         remittances.rem.ID.String(),
		factoringkit.ParamVersion: float64(remittances.rem.Version),
		paramLineID:               remittances.line.ID.String(),
		paramLineAction:           "Book",
	})
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), badAction))
}

func TestApplyFactoringRemittance_PreviewsTheJournalAndRunsOnlyWhenApproved(t *testing.T) {
	t.Parallel()

	remittances := newFakeRemittances()
	tool := NewApplyFactoringRemittanceTool(remittances, remittances.ledger, nil)
	params := agenttoolstest.PersonCallParams(map[string]any{
		paramRemittanceID2:        remittances.rem.ID.String(),
		factoringkit.ParamVersion: float64(remittances.rem.Version),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&remittances.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	require.Len(t, preview.Changes, 1)
	assert.Equal(t, permission.ResourceJournalEntry, preview.Changes[0].Resource)
	assert.Contains(t, preview.Summary, "rts-funding.csv")
	assert.Contains(t, preview.Summary, "1 invoice funded")
	assert.Contains(t, preview.Summary, "1 line still to resolve stay open")

	require.ErrorIs(t, tool.Execute(t.Context(), params), toolspec.ErrNeedsAPersonsApproval)
	assert.Nil(t, remittances.applied)
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.Approved(params)))
	require.NotNil(t, remittances.applied)
	assert.Equal(t, int64(5), remittances.applied.Version)

	stale := agenttoolstest.PersonCallParams(map[string]any{
		paramRemittanceID2:        remittances.rem.ID.String(),
		factoringkit.ParamVersion: float64(4),
	})
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), stale))
}

func TestDiscardAndRematchFactoringRemittance(t *testing.T) {
	t.Parallel()

	remittances := newFakeRemittances()
	discard := NewDiscardFactoringRemittanceTool(remittances)
	params := agenttoolstest.PersonCallParams(map[string]any{
		paramRemittanceID2:        remittances.rem.ID.String(),
		factoringkit.ParamVersion: float64(remittances.rem.Version),
	})
	require.Error(t, discard.(agenttoolports.ToolValidator).Validate(t.Context(), params),
		"a discard says why")

	params.Params[factoringkit.ParamReason] = "A duplicate of yesterday's report"
	preview, err := discard.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Len(t, preview.Changes, 1)
	require.NoError(t, discard.Execute(t.Context(), params))
	assert.Equal(t, "A duplicate of yesterday's report", remittances.discarded.Reason)

	rematch := NewRematchFactoringRemittanceTool(remittances)
	rematchParams := agenttoolstest.PersonCallParams(map[string]any{
		paramRemittanceID2:        remittances.rem.ID.String(),
		factoringkit.ParamVersion: float64(remittances.rem.Version),
	})
	require.NoError(t, rematch.(agenttoolports.ToolValidator).Validate(t.Context(), rematchParams))
	require.NoError(t, rematch.Execute(t.Context(), rematchParams))
	require.NotNil(t, remittances.rematched)

	remittances.rem.Status = factoringremittance.RemittanceStatusApplied
	require.Error(t, rematch.(agenttoolports.ToolValidator).Validate(t.Context(), rematchParams),
		"an applied remittance cannot be matched again")
}
