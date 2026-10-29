package settlementgenerationtools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/paytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/carriersettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const periodEndUnix = agenttoolstest.PeriodStartUnix + 7*toolparams.SecondsPerDay

const payDateUnix = periodEndUnix + 3*toolparams.SecondsPerDay

func currentBounds() driversettlementservice.PeriodBounds {
	return driversettlementservice.PeriodBounds{
		PeriodStart: agenttoolstest.PeriodStartUnix,
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
	_ *permissionports.RequestActor,
) (*driversettlement.Settlement, error) {
	f.generated = req

	return f.created, nil
}

func TestGenerateDriverSettlement_DefaultsToTheCurrentPeriod(t *testing.T) {
	t.Parallel()

	draft := agenttoolstest.PendingDriverSettlement()
	draft.Status = driversettlement.StatusDraft
	draft.WorkerID = pulid.MustNew("wrk_")
	draft.HasExceptions = true
	draft.Exceptions = []driversettlement.Exception{{Code: "MissingPOD"}}
	generator := &fakeWorkerGenerator{
		plan:    &driversettlementservice.GenerationPlan{Settlement: draft, AutoApprove: true},
		created: draft,
	}
	tool := &GenerateDriverSettlementTool{settlements: generator}
	params := agenttoolstest.ExecuteParams(
		map[string]any{toolparams.ParamWorkerID: draft.WorkerID.String()},
	)

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, agenttoolstest.PeriodStartUnix, generator.planned.PeriodStart)
	assert.Equal(t, periodEndUnix, generator.planned.PeriodEnd)
	assert.Equal(t, payDateUnix, generator.planned.PayDate)
	assert.Contains(t, preview.Summary, "It raises 1 exception for review.")
	assert.Contains(t, preview.Summary, "approves a clean settlement")
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agentpreview.PreviewOperationCreate, change.Operation)
	require.NotNil(t, change.Money)

	result, err := tool.ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, draft.ID.String(), result.IDs[settlementpreview.ParamSettlementID])
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
	tool := &GenerateDriverSettlementTool{settlements: generator}
	workerID := pulid.MustNew("wrk_").String()

	half := agenttoolstest.ExecuteParams(
		map[string]any{
			toolparams.ParamWorkerID:    workerID,
			toolparams.ParamPeriodStart: "2026-09-01",
		},
	)
	require.ErrorIs(t, tool.Validate(t.Context(), half), errHalfAPeriod)

	backwards := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamWorkerID:    workerID,
		toolparams.ParamPeriodStart: "2026-09-07",
		toolparams.ParamPeriodEnd:   "2026-09-01",
	})
	require.Error(t, tool.Validate(t.Context(), backwards))

	params := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamWorkerID:    workerID,
		toolparams.ParamPeriodStart: "2026-09-01",
		toolparams.ParamPeriodEnd:   "2026-09-07",
	})
	err := tool.Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already has a settlement for this period")
	assert.Equal(
		t,
		generator.planned.PeriodStart+7*toolparams.SecondsPerDay,
		generator.planned.PeriodEnd,
	)
	assert.Equal(
		t,
		generator.planned.PeriodEnd+3*toolparams.SecondsPerDay,
		generator.planned.PayDate,
	)

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
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
	_ *permissionports.RequestActor,
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
		Workers: []*driversettlementports.UnsettledWorkerSummary{
			{WorkerID: pulid.MustNew("wrk_"), WorkerName: "Ana Ruiz", EventCount: 4,
				GrossAmountMinor: 250000},
			{WorkerID: pulid.MustNew("wrk_"), WorkerName: "Settled", EventCount: 2,
				HasSettlement: true},
			{WorkerID: pulid.MustNew("wrk_"), WorkerName: "All held", HeldCount: 1},
		},
	}}
	tool := &GenerateDriverBatchTool{settlements: batch}
	params := agenttoolstest.ExecuteParams(
		map[string]any{toolparams.ParamNote: "  Labor Day week  "},
	)

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	assert.True(t, preview.Partial)
	require.Len(t, preview.Changes, 1)
	assert.Equal(t, "Ana Ruiz", agenttoolstest.PreviewChange(t, preview, 0).Label)
	assert.Contains(t, preview.Summary, "1 driver would get a draft settlement.")
	assert.Contains(t, preview.Summary, "join the open batch Pay period ending Sep 7")

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, "Labor Day week", batch.generated.Notes)
	assert.Zero(t, batch.generated.PeriodStart)

	batch.plan.Refusal = errortypes.NewBusinessError("The batch for this period is completed")
	preview, err = tool.Preview(t.Context(), params)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
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
	*permissionports.RequestActor,
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
	tool := &GenerateCarrierBatchTool{settlements: batch}
	assert.Equal(t, permission.ResourceCarrierSettlement, tool.Policy().Resource)

	preview, err := tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{}))
	require.NoError(t, err)
	assert.Equal(t, maxBatchPreviewRecords, batch.request.Limit)
	assert.True(t, preview.Partial)
	assert.Equal(t, 3, preview.OmittedRecords)
	assert.Contains(t, preview.Summary, "Pending cost for the period covers 5 carriers")
	require.Len(t, preview.Changes, 1)
	assert.Equal(
		t,
		permission.ResourceCarrier,
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			"carrierId",
		).AfterRef.Resource,
	)
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
	_ *permissionports.RequestActor,
) (*driversettlement.Settlement, error) {
	f.paid = req

	return f.plan.Settlement, nil
}

func TestPayWorkerNow_OnlyAPersonPaysSomeone(t *testing.T) {
	t.Parallel()

	settlement := agenttoolstest.PendingDriverSettlement()
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
	tool := &paytools.PayWorkerNowTool{Settlements: payer}
	policy := tool.Policy()
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.True(t, policy.HasEgress(agent.EgressDriverVisible))

	event := pulid.MustNew("dpe_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamWorkerID:             pulid.MustNew("wrk_").String(),
		settlementpreview.ParamPaymentMethod: "InstantPay",
		paytools.ParamPayEventIDs:            []any{event.String()},
		paytools.ParamApplyRecurring:         true,
	})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "Would pay the driver now for 2 loads")
	assert.Contains(t, preview.Summary, "by InstantPay")
	require.Len(t, preview.Changes, 2)
	assert.Equal(
		t,
		permission.ResourceJournalEntry,
		agenttoolstest.PreviewChange(t, preview, 1).Resource,
	)

	_, err = tool.ExecuteWithResult(t.Context(), params)
	require.ErrorIs(t, err, settlementpreview.ErrSettlementNeedsAPerson)
	assert.Nil(t, payer.paid)

	params.ProposalID = pulid.MustNew("ap_")
	result, err := tool.ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, settlement.ID.String(), result.IDs[settlementpreview.ParamSettlementID])
	assert.Equal(t, []pulid.ID{event}, payer.paid.PayEventIDs)
	assert.True(t, payer.paid.ApplyRecurring)

	payer.plan.Refusal = errNoInstantPayFixture
	preview, err = tool.Preview(t.Context(), params)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
}

var errNoInstantPayFixture = errortypes.NewValidationError(
	"workerId",
	errortypes.ErrInvalidOperation,
	"The driver has no accrued, unheld pay to pay now",
)
