package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/carriersettlementservice"
	"github.com/emoss08/trenova/internal/core/services/driversettlementservice"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	periodStartUnix = int64(1_788_220_800)
	periodEndUnix   = periodStartUnix + 7*secondsPerDay
	payDateUnix     = periodEndUnix + 3*secondsPerDay
)

func currentBounds() driversettlementservice.PeriodBounds {
	return driversettlementservice.PeriodBounds{
		PeriodStart: periodStartUnix,
		PeriodEnd:   periodEndUnix,
		PayDate:     payDateUnix,
	}
}

type fakeWorkerGenerator struct {
	plan      *driversettlementservice.GenerationPlan
	created   *driversettlement.Settlement
	planned   *driversettlementservice.GenerateForWorkerRequest
	generated *driversettlementservice.GenerateForWorkerRequest
}

func (f *fakeWorkerGenerator) CurrentPeriod(
	context.Context,
	pagination.TenantInfo,
) (driversettlementservice.PeriodBounds, error) {
	return currentBounds(), nil
}

func (f *fakeWorkerGenerator) PlanGenerateForWorker(
	_ context.Context,
	req *driversettlementservice.GenerateForWorkerRequest,
) (*driversettlementservice.GenerationPlan, error) {
	f.planned = req

	return f.plan, nil
}

func (f *fakeWorkerGenerator) GenerateOffCycle(
	_ context.Context,
	req *driversettlementservice.GenerateForWorkerRequest,
	_ *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	f.generated = req

	return f.created, nil
}

func TestGenerateDriverSettlement_DefaultsToTheCurrentPeriod(t *testing.T) {
	t.Parallel()

	draft := pendingDriverSettlement()
	draft.Status = driversettlement.StatusDraft
	draft.WorkerID = pulid.MustNew("wrk_")
	draft.HasExceptions = true
	draft.Exceptions = []driversettlement.Exception{{Code: "MissingPOD"}}
	generator := &fakeWorkerGenerator{
		plan:    &driversettlementservice.GenerationPlan{Settlement: draft, AutoApprove: true},
		created: draft,
	}
	tool := &generateDriverSettlementTool{settlements: generator}
	params := executeParams(map[string]any{paramWorkerID: draft.WorkerID.String()})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, periodStartUnix, generator.planned.PeriodStart)
	assert.Equal(t, periodEndUnix, generator.planned.PeriodEnd)
	assert.Equal(t, payDateUnix, generator.planned.PayDate)
	assert.Contains(t, preview.Summary, "It raises 1 exception for review.")
	assert.Contains(t, preview.Summary, "approves a clean settlement")
	change := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, change.Operation)
	require.NotNil(t, change.Money)

	result, err := tool.ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, draft.ID.String(), result.IDs[paramSettlementID])
	require.NotNil(t, result.Record)
	assert.Equal(t, tool.Policy().Artifact, result.Record.EntityType)

	generator.created = nil
	_, err = tool.ExecuteWithResult(t.Context(), params)
	require.ErrorIs(t, err, ErrNothingToSettle)
}

func TestGenerateDriverSettlement_ReadsAPeriodAsWholeDays(t *testing.T) {
	t.Parallel()

	generator := &fakeWorkerGenerator{
		plan: &driversettlementservice.GenerationPlan{AlreadySettled: true},
	}
	tool := &generateDriverSettlementTool{settlements: generator}
	workerID := pulid.MustNew("wrk_").String()

	half := executeParams(map[string]any{paramWorkerID: workerID, paramPeriodStart: "2026-09-01"})
	require.ErrorIs(t, tool.Validate(t.Context(), half), errHalfAPeriod)

	backwards := executeParams(map[string]any{
		paramWorkerID:    workerID,
		paramPeriodStart: "2026-09-07",
		paramPeriodEnd:   "2026-09-01",
	})
	require.Error(t, tool.Validate(t.Context(), backwards))

	params := executeParams(map[string]any{
		paramWorkerID:    workerID,
		paramPeriodStart: "2026-09-01",
		paramPeriodEnd:   "2026-09-07",
	})
	err := tool.Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already has a settlement for this period")
	assert.Equal(t, generator.planned.PeriodStart+7*secondsPerDay, generator.planned.PeriodEnd)
	assert.Equal(t, generator.planned.PeriodEnd+3*secondsPerDay, generator.planned.PayDate)

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
	assert.Contains(t, preview.Summary, "Sep 1, 2026 through Sep 7, 2026")
}

type fakeDriverBatch struct {
	plan      *driversettlementservice.BatchPlan
	generated *driversettlementservice.GenerateBatchRequest
}

func (f *fakeDriverBatch) PlanBatch(
	context.Context,
	*driversettlementservice.GenerateBatchRequest,
) (*driversettlementservice.BatchPlan, error) {
	return f.plan, nil
}

func (f *fakeDriverBatch) GenerateBatch(
	_ context.Context,
	req *driversettlementservice.GenerateBatchRequest,
	_ *serviceports.RequestActor,
) (*driversettlement.SettlementBatch, error) {
	f.generated = req

	return &driversettlement.SettlementBatch{}, nil
}

func TestGenerateDriverSettlementBatch_PreviewsEachDriverStillToSettle(t *testing.T) {
	t.Parallel()

	batch := &fakeDriverBatch{plan: &driversettlementservice.BatchPlan{
		Bounds: currentBounds(),
		ExistingBatch: &driversettlement.SettlementBatch{
			Name: "Pay period ending Sep 7",
		},
		Workers: []*repositories.UnsettledWorkerSummary{
			{WorkerID: pulid.MustNew("wrk_"), WorkerName: "Ana Ruiz", EventCount: 4,
				GrossAmountMinor: 250000},
			{WorkerID: pulid.MustNew("wrk_"), WorkerName: "Settled", EventCount: 2,
				HasSettlement: true},
			{WorkerID: pulid.MustNew("wrk_"), WorkerName: "All held", HeldCount: 1},
		},
	}}
	tool := &generateDriverBatchTool{settlements: batch}
	params := executeParams(map[string]any{paramBatchNotes: "  Labor Day week  "})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	assert.True(t, preview.Partial)
	require.Len(t, preview.Changes, 1)
	assert.Equal(t, "Ana Ruiz", previewChange(t, preview, 0).Label)
	assert.Contains(t, preview.Summary, "1 driver would get a draft settlement.")
	assert.Contains(t, preview.Summary, "join the open batch Pay period ending Sep 7")

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, "Labor Day week", batch.generated.Notes)
	assert.Zero(t, batch.generated.PeriodStart)

	batch.plan.Refusal = errortypes.NewBusinessError("The batch for this period is completed")
	preview, err = tool.Preview(t.Context(), params)
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
	require.Error(t, tool.Validate(t.Context(), params))
}

type fakeCarrierBatch struct {
	plan    *carriersettlementservice.BatchPlan
	request *carriersettlementservice.BatchPlanRequest
}

func (f *fakeCarrierBatch) PlanBatch(
	_ context.Context,
	req *carriersettlementservice.BatchPlanRequest,
) (*carriersettlementservice.BatchPlan, error) {
	f.request = req

	return f.plan, nil
}

func (f *fakeCarrierBatch) GenerateBatch(
	context.Context,
	*carriersettlementservice.GenerateBatchRequest,
	*serviceports.RequestActor,
) (*carriersettlement.CarrierSettlementBatch, error) {
	return &carriersettlement.CarrierSettlementBatch{}, nil
}

func TestGenerateCarrierSettlementBatch_CountsWhatThePreviewLeavesOut(t *testing.T) {
	t.Parallel()

	settlement := &carriersettlement.CarrierSettlement{
		CarrierID:       pulid.MustNew("car_"),
		Status:          carriersettlement.StatusDraft,
		GrossCostMinor:  120000,
		NetPayableMinor: 120000,
		CurrencyCode:    "USD",
	}
	batch := &fakeCarrierBatch{plan: &carriersettlementservice.BatchPlan{
		Bounds:       currentBounds(),
		Settlements:  []*carriersettlement.CarrierSettlement{settlement},
		CarrierCount: 5,
		SettledCount: 1,
	}}
	tool := &generateCarrierBatchTool{settlements: batch}
	assert.Equal(t, permission.ResourceCarrierSettlement, tool.Policy().Resource)

	preview, err := tool.Preview(t.Context(), executeParams(map[string]any{}))
	require.NoError(t, err)
	assert.Equal(t, maxBatchPreviewRecords, batch.request.Limit)
	assert.True(t, preview.Partial)
	assert.Equal(t, 3, preview.OmittedRecords)
	assert.Contains(t, preview.Summary, "Pending cost for the period covers 5 carriers")
	require.Len(t, preview.Changes, 1)
	assert.Equal(t, permission.ResourceCarrier,
		fieldByPath(t, previewChange(t, preview, 0), "carrierId").AfterRef.Resource)
}

type fakeInstantPayer struct {
	plan *driversettlementservice.InstantPayPlan
	paid *driversettlementservice.PayWorkerNowRequest
}

func (f *fakeInstantPayer) PlanPayWorkerNow(
	context.Context,
	*driversettlementservice.PayWorkerNowRequest,
) (*driversettlementservice.InstantPayPlan, error) {
	return f.plan, nil
}

func (f *fakeInstantPayer) PayWorkerNow(
	_ context.Context,
	req *driversettlementservice.PayWorkerNowRequest,
	_ *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	f.paid = req

	return f.plan.Settlement, nil
}

func TestPayWorkerNow_OnlyAPersonPaysSomeone(t *testing.T) {
	t.Parallel()

	settlement := pendingDriverSettlement()
	settlement.Status = driversettlement.StatusPaid
	settlement.PaymentMethod = "InstantPay"
	settlement.ShipmentCount = 2
	payer := &fakeInstantPayer{plan: &driversettlementservice.InstantPayPlan{
		Settlement: settlement,
		Journal: &settlementshared.JournalPlan{
			EntryStatus: "Posted",
			Lines: []settlementshared.JournalLine{
				{AccountID: pulid.MustNew("gla_"), DebitMinor: 174000},
				{AccountID: pulid.MustNew("gla_"), CreditMinor: 174000},
			},
		},
	}}
	tool := &payWorkerNowTool{settlements: payer}
	policy := tool.Policy()
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.True(t, policy.HasEgress(agent.EgressDriverVisible))

	event := pulid.MustNew("dpe_")
	params := executeParams(map[string]any{
		paramWorkerID:       pulid.MustNew("wrk_").String(),
		paramPaymentMethod:  "InstantPay",
		paramPayEventIDs:    []any{event.String()},
		paramApplyRecurring: true,
	})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "Would pay the driver now for 2 loads")
	assert.Contains(t, preview.Summary, "by InstantPay")
	require.Len(t, preview.Changes, 2)
	assert.Equal(t, permission.ResourceJournalEntry, previewChange(t, preview, 1).Resource)

	_, err = tool.ExecuteWithResult(t.Context(), params)
	require.ErrorIs(t, err, ErrSettlementNeedsAPerson)
	assert.Nil(t, payer.paid)

	params.ProposalID = pulid.MustNew("ap_")
	result, err := tool.ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, settlement.ID.String(), result.IDs[paramSettlementID])
	assert.Equal(t, []pulid.ID{event}, payer.paid.PayEventIDs)
	assert.True(t, payer.paid.ApplyRecurring)

	payer.plan.Refusal = errNoInstantPayFixture
	preview, err = tool.Preview(t.Context(), params)
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}

var errNoInstantPayFixture = errortypes.NewValidationError(
	"workerId",
	errortypes.ErrInvalidOperation,
	"The driver has no accrued, unheld pay to pay now",
)
