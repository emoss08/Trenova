package workforcetools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePTORequests struct {
	guard       *agenttoolstest.WriteGuard
	pto         *worker.WorkerPTO
	autoApprove bool

	created *worker.WorkerPTO
	updated *worker.WorkerPTO
}

func newFakePTORequests() *fakePTORequests {
	return &fakePTORequests{
		guard: &agenttoolstest.WriteGuard{},
		pto: &worker.WorkerPTO{
			ID:        pulid.MustNew("wrkpto_"),
			WorkerID:  pulid.MustNew("wrk_"),
			Status:    worker.PTOStatusRequested,
			Type:      worker.PTOTypeVacation,
			StartDate: 1_791_072_000,
			EndDate:   1_791_331_200,
			Reason:    "Family trip",
			Days:      decimal.NewFromInt(4),
			Version:   1,
		},
	}
}

func (f *fakePTORequests) Get(
	context.Context,
	*workerports.GetPTOByIDRequest,
) (*worker.WorkerPTO, error) {
	copied := *f.pto
	return &copied, nil
}

func (f *fakePTORequests) PlanCreate(
	_ context.Context,
	entity *worker.WorkerPTO,
	userID pulid.ID,
) (*worker.WorkerPTO, error) {
	if entity.EndDate < entity.StartDate {
		return nil, errortypes.NewValidationError("endDate", errortypes.ErrInvalid,
			"End date must be after the start date")
	}
	planned := *entity
	planned.Days = decimal.NewFromInt((entity.EndDate-entity.StartDate)/86_400 + 1)
	if f.autoApprove {
		planned.Status = worker.PTOStatusApproved
		planned.ApproverID = userID
		planned.AutoApproved = true
	}
	return &planned, nil
}

func (f *fakePTORequests) Create(
	ctx context.Context,
	entity *worker.WorkerPTO,
	userID pulid.ID,
) (*worker.WorkerPTO, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	planned, err := f.PlanCreate(ctx, entity, userID)
	if err != nil {
		return nil, err
	}
	planned.ID = pulid.MustNew("wrkpto_")
	f.created = planned
	return planned, nil
}

func (f *fakePTORequests) PlanUpdate(
	_ context.Context,
	entity *worker.WorkerPTO,
) (*kernelports.RecordChange[worker.WorkerPTO], error) {
	if f.pto.Status != worker.PTOStatusRequested {
		return nil, errortypes.NewValidationError("status", errortypes.ErrInvalidOperation,
			"PTO is approved and can no longer be edited")
	}
	return &kernelports.RecordChange[worker.WorkerPTO]{Before: f.pto, After: entity}, nil
}

func (f *fakePTORequests) Update(
	_ context.Context,
	entity *worker.WorkerPTO,
	_ pulid.ID,
) (*worker.WorkerPTO, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.updated = entity
	return entity, nil
}

type fakePTOLedger struct {
	guard    *agenttoolstest.WriteGuard
	balance  decimal.Decimal
	adjusted *ptoledgerservice.AdjustRequest
}

func (f *fakePTOLedger) PlanAdjust(
	_ context.Context,
	req *ptoledgerservice.AdjustRequest,
) (*ptoledgerservice.AdjustPlan, error) {
	after := f.balance.Add(req.AmountDays)
	return &ptoledgerservice.AdjustPlan{
		Entry: &pto.WorkerPTOLedgerEntry{
			WorkerID:         req.WorkerID,
			PTOType:          req.PTOType,
			AmountDays:       req.AmountDays,
			BalanceAfterDays: after,
			Note:             req.Note,
		},
		BalanceBefore: f.balance,
		BalanceAfter:  after,
	}, nil
}

func (f *fakePTOLedger) Adjust(
	_ context.Context,
	req *ptoledgerservice.AdjustRequest,
) (*pto.WorkerPTOLedgerEntry, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.adjusted = req
	return &pto.WorkerPTOLedgerEntry{ID: pulid.MustNew("wpl_"), WorkerID: req.WorkerID}, nil
}

func TestRequestWorkerPTO_SaysWhenThePolicyBooksItAtOnce(t *testing.T) {
	t.Parallel()

	requests := newFakePTORequests()
	tool := newRequestWorkerPTOTool(requests)
	params := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamWorkerID: requests.pto.WorkerID.String(),
		paramPTOType:         "Vacation",
		paramPTOStart:        "2026-10-05",
		paramPTOEnd:          "2026-10-07",
		common.FieldReason:   "Wedding",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		requests.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "Would request 3 days of Vacation")
	assert.NotContains(t, preview.Summary, "booked at once")

	requests.autoApprove = true
	preview = agenttoolstest.PreviewWithoutWrites(
		t,
		requests.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "booked at once and the driver is told")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, requests.created)
	assert.Equal(t, worker.PTOTypeVacation, requests.created.Type)
	assert.Equal(t, "Wedding", requests.created.Reason)
	assert.Equal(t, []agent.EgressClass{agent.EgressDriverVisible}, tool.Policy().Egress)
	assert.Equal(t, permission.ResourceWorkerPTO, tool.Policy().Resource)

	backwards := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamWorkerID: requests.pto.WorkerID.String(),
		paramPTOType:         "Vacation",
		paramPTOStart:        "2026-10-07",
		paramPTOEnd:          "2026-10-05",
		common.FieldReason:   "Wedding",
	})
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), backwards))
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			common.ParamWorkerID: requests.pto.WorkerID.String(),
			paramPTOType:         "Sabbatical",
			paramPTOStart:        "2026-10-05",
			paramPTOEnd:          "2026-10-07",
			common.FieldReason:   "x",
		})))
}

func TestUpdateWorkerPTO_OnlyAPendingRequest(t *testing.T) {
	t.Parallel()

	requests := newFakePTORequests()
	tool := newUpdateWorkerPTOTool(requests)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"ptoId":            requests.pto.ID.String(),
		paramPTOEnd:        "2026-10-09",
		paramPTOType:       "Personal",
		common.FieldReason: "Extended trip",
		paramPTOStart:      "2026-10-05",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		requests.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(t, "Would change the time-off request.", preview.Summary)
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, requests.updated)
	assert.Equal(t, worker.PTOTypePersonal, requests.updated.Type)
	assert.Nil(t, requests.updated.Worker)

	partial := agenttoolstest.ExecuteParams(map[string]any{
		"ptoId":            requests.pto.ID.String(),
		common.FieldReason: "Only the reason",
	})
	require.NoError(t, tool.Execute(t.Context(), partial))
	assert.Equal(t, requests.pto.Type, requests.updated.Type)
	assert.Equal(t, requests.pto.StartDate, requests.updated.StartDate)

	requests.pto.Status = worker.PTOStatusApproved
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), partial))
}

func TestAdjustWorkerPTOBalance_IsAPersonsDecision(t *testing.T) {
	t.Parallel()

	ledger := &fakePTOLedger{guard: &agenttoolstest.WriteGuard{}, balance: decimal.NewFromInt(5)}
	tool := newAdjustWorkerPTOBalanceTool(ledger)
	raw := map[string]any{
		common.ParamWorkerID: pulid.MustNew("wrk_").String(),
		paramPTOType:         "Vacation",
		paramAmountDays:      "2.5",
		common.FieldNote:     "Carried over from the previous payroll system",
	}

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		ledger.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(
				t.Context(),
				agenttoolstest.ExecuteParams(raw),
			)
		},
	)
	assert.Contains(t, preview.Summary, "from 5 to 7.5")

	require.ErrorIs(
		t,
		tool.Execute(t.Context(), agenttoolstest.ExecuteParams(raw)),
		common.ErrNeedsAPersonsApproval,
	)
	assert.Nil(t, ledger.adjusted)
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ApprovedParams(raw)))
	require.NotNil(t, ledger.adjusted)
	assert.True(t, decimal.RequireFromString("2.5").Equal(ledger.adjusted.AmountDays))

	policy := tool.Policy()
	assert.Equal(t, permission.OpManage, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)

	for name, params := range map[string]map[string]any{
		"zero days": {
			common.ParamWorkerID: pulid.MustNew("wrk_").String(), paramPTOType: "Vacation",
			paramAmountDays: "0", common.FieldNote: "x",
		},
		"no note": {
			common.ParamWorkerID: pulid.MustNew("wrk_").String(), paramPTOType: "Vacation",
			paramAmountDays: "1",
		},
	} {
		require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
			agenttoolstest.ExecuteParams(params)), name)
	}
}
