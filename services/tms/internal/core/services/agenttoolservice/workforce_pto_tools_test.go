package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ptoledgerservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePTORequests struct {
	guard       *writeGuard
	pto         *worker.WorkerPTO
	autoApprove bool

	created *worker.WorkerPTO
	updated *worker.WorkerPTO
}

func newFakePTORequests() *fakePTORequests {
	return &fakePTORequests{
		guard: &writeGuard{},
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
	*repositories.GetPTOByIDRequest,
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
	if err := f.guard.write(); err != nil {
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
) (*serviceports.RecordChange[worker.WorkerPTO], error) {
	if f.pto.Status != worker.PTOStatusRequested {
		return nil, errortypes.NewValidationError("status", errortypes.ErrInvalidOperation,
			"PTO is approved and can no longer be edited")
	}
	return &serviceports.RecordChange[worker.WorkerPTO]{Before: f.pto, After: entity}, nil
}

func (f *fakePTORequests) Update(
	_ context.Context,
	entity *worker.WorkerPTO,
	_ pulid.ID,
) (*worker.WorkerPTO, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.updated = entity
	return entity, nil
}

type fakePTOLedger struct {
	guard    *writeGuard
	balance  decimal.Decimal
	adjusted *ptoledgerservice.AdjustRequest
}

func (f *fakePTOLedger) PlanAdjust(
	_ context.Context,
	req *ptoledgerservice.AdjustRequest,
) (*ptoledgerservice.AdjustPlan, error) {
	after := f.balance.Add(req.AmountDays)
	return &ptoledgerservice.AdjustPlan{
		Entry: &worker.WorkerPTOLedgerEntry{
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
) (*worker.WorkerPTOLedgerEntry, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.adjusted = req
	return &worker.WorkerPTOLedgerEntry{ID: pulid.MustNew("wpl_"), WorkerID: req.WorkerID}, nil
}

func TestRequestWorkerPTO_SaysWhenThePolicyBooksItAtOnce(t *testing.T) {
	t.Parallel()

	requests := newFakePTORequests()
	tool := newRequestWorkerPTOTool(requests)
	params := executeParams(map[string]any{
		paramWorkerID: requests.pto.WorkerID.String(),
		paramPTOType:  "Vacation",
		paramPTOStart: "2026-10-05",
		paramPTOEnd:   "2026-10-07",
		fieldReason:   "Wedding",
	})

	preview := previewWithoutWrites(t, requests.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would request 3 days of Vacation")
	assert.NotContains(t, preview.Summary, "booked at once")

	requests.autoApprove = true
	preview = previewWithoutWrites(t, requests.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "booked at once and the driver is told")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, requests.created)
	assert.Equal(t, worker.PTOTypeVacation, requests.created.Type)
	assert.Equal(t, "Wedding", requests.created.Reason)
	assert.Equal(t, []agent.EgressClass{agent.EgressDriverVisible}, tool.Policy().Egress)
	assert.Equal(t, permission.ResourceWorkerPTO, tool.Policy().Resource)

	backwards := executeParams(map[string]any{
		paramWorkerID: requests.pto.WorkerID.String(),
		paramPTOType:  "Vacation",
		paramPTOStart: "2026-10-07",
		paramPTOEnd:   "2026-10-05",
		fieldReason:   "Wedding",
	})
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), backwards))
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramWorkerID: requests.pto.WorkerID.String(),
			paramPTOType:  "Sabbatical",
			paramPTOStart: "2026-10-05",
			paramPTOEnd:   "2026-10-07",
			fieldReason:   "x",
		})))
}

func TestUpdateWorkerPTO_OnlyAPendingRequest(t *testing.T) {
	t.Parallel()

	requests := newFakePTORequests()
	tool := newUpdateWorkerPTOTool(requests)
	params := executeParams(map[string]any{
		"ptoId":       requests.pto.ID.String(),
		paramPTOEnd:   "2026-10-09",
		paramPTOType:  "Personal",
		fieldReason:   "Extended trip",
		paramPTOStart: "2026-10-05",
	})

	preview := previewWithoutWrites(t, requests.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "Would change the time-off request.", preview.Summary)
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, requests.updated)
	assert.Equal(t, worker.PTOTypePersonal, requests.updated.Type)
	assert.Nil(t, requests.updated.Worker)

	partial := executeParams(map[string]any{
		"ptoId":     requests.pto.ID.String(),
		fieldReason: "Only the reason",
	})
	require.NoError(t, tool.Execute(t.Context(), partial))
	assert.Equal(t, requests.pto.Type, requests.updated.Type)
	assert.Equal(t, requests.pto.StartDate, requests.updated.StartDate)

	requests.pto.Status = worker.PTOStatusApproved
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), partial))
}

func TestAdjustWorkerPTOBalance_IsAPersonsDecision(t *testing.T) {
	t.Parallel()

	ledger := &fakePTOLedger{guard: &writeGuard{}, balance: decimal.NewFromInt(5)}
	tool := newAdjustWorkerPTOBalanceTool(ledger)
	raw := map[string]any{
		paramWorkerID:   pulid.MustNew("wrk_").String(),
		paramPTOType:    "Vacation",
		paramAmountDays: "2.5",
		fieldNote:       "Carried over from the previous payroll system",
	}

	preview := previewWithoutWrites(t, ledger.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), executeParams(raw))
	})
	assert.Contains(t, preview.Summary, "from 5 to 7.5")

	require.ErrorIs(t, tool.Execute(t.Context(), executeParams(raw)), ErrNeedsAPersonsApproval)
	assert.Nil(t, ledger.adjusted)
	require.NoError(t, tool.Execute(t.Context(), approvedParams(raw)))
	require.NotNil(t, ledger.adjusted)
	assert.True(t, decimal.RequireFromString("2.5").Equal(ledger.adjusted.AmountDays))

	policy := tool.Policy()
	assert.Equal(t, permission.OpManage, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)

	for name, params := range map[string]map[string]any{
		"zero days": {
			paramWorkerID: pulid.MustNew("wrk_").String(), paramPTOType: "Vacation",
			paramAmountDays: "0", fieldNote: "x",
		},
		"no note": {
			paramWorkerID: pulid.MustNew("wrk_").String(), paramPTOType: "Vacation",
			paramAmountDays: "1",
		},
	} {
		require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
			executeParams(params)), name)
	}
}
