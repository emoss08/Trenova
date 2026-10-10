package agenttoolservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/fuelpurchaseservice"
	"github.com/emoss08/trenova/internal/core/services/fuelsurchargeservice"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var texasID = pulid.MustNew("ifj_")

type fakeJurisdictions struct{}

func (fakeJurisdictions) GetJurisdiction(_ context.Context, id pulid.ID) (*ifta.Jurisdiction, error) {
	if id != texasID {
		return nil, errortypes.NewNotFoundError("Jurisdiction not found")
	}

	return &ifta.Jurisdiction{ID: texasID, Code: "TX", Name: "Texas"}, nil
}

type fakeFuel struct {
	guard     *writeGuard
	stored    *fuelpurchase.FuelPurchase
	card      *fuelpurchase.FuelCard
	batch     *fuelpurchase.ImportBatch
	refusal   error
	created   *fuelpurchaseservice.CreatePurchaseRequest
	updated   *fuelpurchaseservice.UpdatePurchaseRequest
	deleted   *fuelpurchaseservice.DeletePurchaseRequest
	assigned  *fuelpurchaseservice.AssignCardRequest
	committed *fuelpurchaseservice.CommitRequest
	discarded *fuelpurchaseservice.DiscardRequest
}

func (f *fakeFuel) GetPurchase(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*fuelpurchase.FuelPurchase, error) {
	copied := *f.stored

	return &copied, nil
}

func plannedPurchase(purchase *fuelpurchase.FuelPurchase) *fuelpurchase.FuelPurchase {
	copied := *purchase
	copied.Gallons = copied.QuantityUnit.ToGallons(copied.Quantity)

	return &copied
}

func (f *fakeFuel) PlanCreatePurchase(
	_ context.Context,
	req *fuelpurchaseservice.CreatePurchaseRequest,
) (*fuelpurchase.FuelPurchase, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return plannedPurchase(req.Purchase), nil
}

func (f *fakeFuel) CreatePurchase(
	ctx context.Context,
	req *fuelpurchaseservice.CreatePurchaseRequest,
) (*fuelpurchase.FuelPurchase, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.created = req
	created, err := f.PlanCreatePurchase(ctx, req)
	if err != nil {
		return nil, err
	}
	created.ID = pulid.MustNew("fpur_")

	return created, nil
}

func (f *fakeFuel) PlanUpdatePurchase(
	_ context.Context,
	req *fuelpurchaseservice.UpdatePurchaseRequest,
) (*fuelpurchaseservice.PurchaseChange, error) {
	return &fuelpurchaseservice.PurchaseChange{Before: f.stored, After: plannedPurchase(req.Purchase)}, nil
}

func (f *fakeFuel) UpdatePurchase(
	_ context.Context,
	req *fuelpurchaseservice.UpdatePurchaseRequest,
) (*fuelpurchase.FuelPurchase, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.updated = req

	return req.Purchase, nil
}

func (f *fakeFuel) PlanDeletePurchase(
	context.Context,
	*fuelpurchaseservice.DeletePurchaseRequest,
) (*fuelpurchase.FuelPurchase, error) {
	return f.stored, nil
}

func (f *fakeFuel) DeletePurchase(
	_ context.Context,
	req *fuelpurchaseservice.DeletePurchaseRequest,
) error {
	if err := f.guard.write(); err != nil {
		return err
	}
	f.deleted = req

	return nil
}

func (f *fakeFuel) GetCard(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*fuelpurchase.FuelCard, error) {
	copied := *f.card

	return &copied, nil
}

func (f *fakeFuel) PlanAssignCard(
	_ context.Context,
	req *fuelpurchaseservice.AssignCardRequest,
) (*fuelpurchaseservice.CardChange, error) {
	after := *f.card
	after.AssignedTractorID = req.AssignedTractorID
	after.Status = fuelpurchase.CardStatusActive

	return &fuelpurchaseservice.CardChange{Before: f.card, After: &after}, nil
}

func (f *fakeFuel) AssignCard(
	_ context.Context,
	req *fuelpurchaseservice.AssignCardRequest,
) (*fuelpurchase.FuelCard, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.assigned = req

	return f.card, nil
}

func (f *fakeFuel) GetImport(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*fuelpurchase.ImportBatch, error) {
	copied := *f.batch

	return &copied, nil
}

func (f *fakeFuel) PlanCommit(
	context.Context,
	*fuelpurchaseservice.CommitRequest,
) (*fuelpurchaseservice.CommitPlan, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return &fuelpurchaseservice.CommitPlan{
		Batch:     f.batch,
		Purchases: []*fuelpurchase.FuelPurchase{texasPurchase(), texasPurchase()},
	}, nil
}

func (f *fakeFuel) Commit(
	_ context.Context,
	req *fuelpurchaseservice.CommitRequest,
) (*fuelpurchase.ImportBatch, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.committed = req

	return f.batch, nil
}

func (f *fakeFuel) PlanResolveRows(
	context.Context,
	*fuelpurchaseservice.ResolveRowsRequest,
) (*fuelpurchaseservice.ResolveRowsPlan, error) {
	return &fuelpurchaseservice.ResolveRowsPlan{Batch: f.batch, Reviewed: 3, Resolved: 2, Queued: 1}, nil
}

func (f *fakeFuel) ResolveRows(
	context.Context,
	*fuelpurchaseservice.ResolveRowsRequest,
) (*fuelpurchaseservice.ResolveRowsResult, error) {
	return &fuelpurchaseservice.ResolveRowsResult{}, f.guard.write()
}

func (f *fakeFuel) PlanDiscard(
	context.Context,
	*fuelpurchaseservice.DiscardRequest,
) (*fuelpurchaseservice.ImportBatchChange, error) {
	after := *f.batch
	after.Status = fuelpurchase.ImportStatusDiscarded

	return &fuelpurchaseservice.ImportBatchChange{Before: f.batch, After: &after}, nil
}

func (f *fakeFuel) Discard(
	_ context.Context,
	req *fuelpurchaseservice.DiscardRequest,
) (*fuelpurchase.ImportBatch, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.discarded = req

	return f.batch, nil
}

func texasPurchase() *fuelpurchase.FuelPurchase {
	return &fuelpurchase.FuelPurchase{
		ID:                   pulid.MustNew("fpur_"),
		TractorID:            pulid.MustNew("trac_"),
		JurisdictionID:       texasID,
		PurchasedAt:          1_784_000_000,
		FuelType:             domaintypes.IFTAFuelTypeDiesel,
		Quantity:             decimal.RequireFromString("100"),
		QuantityUnit:         fuelpurchase.QuantityUnitGallon,
		Gallons:              decimal.RequireFromString("100"),
		TotalAmountMinor:     38_990,
		CurrencyCode:         "USD",
		TaxPaid:              true,
		TransactionReference: "T-100",
		Version:              2,
	}
}

func purchaseParams() map[string]any {
	return map[string]any{
		paramTractorID:      pulid.MustNew("trac_").String(),
		paramJurisdictionID: texasID.String(),
		paramPurchasedAt:    "2026-09-14T08:30:00-05:00",
		paramFuelType:       "Diesel",
		paramQuantity:       "100.5",
		paramTotalAmount:    "391.85",
	}
}

func TestRecordFuelPurchase_PreviewsTheReceiptAndRecordsIt(t *testing.T) {
	t.Parallel()

	fuel := &fakeFuel{guard: &writeGuard{}}
	tool := newRecordFuelPurchaseTool(fuel, fakeJurisdictions{})
	params := executeParams(purchaseParams())

	preview := previewWithoutWrites(t, fuel.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would record 100.500 gallons of Diesel bought in TX")
	change := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, change.Operation)
	assert.Equal(t, "TX (Texas)", fieldByPath(t, change, "jurisdiction").After)

	result, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, fuel.created)
	assert.Equal(t, int64(39_185), fuel.created.Purchase.TotalAmountMinor)
	assert.True(t, fuel.created.Purchase.TaxPaid)
	assert.Equal(t, fuelpurchase.QuantityUnitGallon, fuel.created.Purchase.QuantityUnit)
	assert.Equal(t, "recorded", result.Action)
	assert.NotEmpty(t, result.IDs[paramFuelPurchaseID])

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceFuelPurchase, policy.Resource)
	assert.Equal(t, permission.OpCreate, policy.Operation)
	assert.Equal(t, agent.TierActWithApproval, policy.MaxTier)
	require.NotNil(t, policy.TaintHold)
}

func TestRecordFuelPurchase_RefusesWhatItCannotRecord(t *testing.T) {
	t.Parallel()

	tool := newRecordFuelPurchaseTool(&fakeFuel{}, fakeJurisdictions{})
	for name, change := range map[string]func(map[string]any){
		"a local time without an offset": func(p map[string]any) { p[paramPurchasedAt] = "2026-09-14 08:30" },
		"a fuel IFTA does not know":      func(p map[string]any) { p[paramFuelType] = "Kerosene" },
		"a zero quantity":                func(p map[string]any) { p[paramQuantity] = "0" },
		"an amount that is not money":    func(p map[string]any) { p[paramTotalAmount] = "a lot" },
		"a bad currency":                 func(p map[string]any) { p[paramCurrencyCode] = "DOLLARS" },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			params := purchaseParams()
			change(params)
			require.Error(t, tool.(serviceports.ToolValidator).Validate(
				t.Context(),
				executeParams(params),
			))
		})
	}

	refused := newRecordFuelPurchaseTool(&fakeFuel{refusal: errortypes.NewValidationError(
		"transactionReference", errortypes.ErrDuplicate, "Already on file",
	)}, fakeJurisdictions{})
	preview, err := refused.(serviceports.ToolPreviewer).Preview(
		t.Context(),
		executeParams(purchaseParams()),
	)
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestCorrectFuelPurchase_KeepsWhatItIsNotGiven(t *testing.T) {
	t.Parallel()

	fuel := &fakeFuel{guard: &writeGuard{}, stored: texasPurchase()}
	tool := newCorrectFuelPurchaseTool(fuel, fakeJurisdictions{})
	params := executeParams(map[string]any{
		paramFuelPurchaseID: fuel.stored.ID.String(),
		paramQuantity:       "120",
	})

	preview := previewWithoutWrites(t, fuel.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "120.000", fieldByPath(t, previewChange(t, preview, 0), "gallons").After)

	_, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, fuel.updated)
	assert.Equal(t, "120", fuel.updated.Purchase.Quantity.String())
	assert.Equal(t, fuel.stored.TotalAmountMinor, fuel.updated.Purchase.TotalAmountMinor)
	assert.Equal(t, fuel.stored.Version, fuel.updated.Purchase.Version)

	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceFuelPurchase, target.Resource)

	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), executeParams(
		map[string]any{paramFuelPurchaseID: fuel.stored.ID.String()},
	)))
}

func TestDeleteFuelPurchase_StopsAtAProposalWithTheVersionItRead(t *testing.T) {
	t.Parallel()

	fuel := &fakeFuel{guard: &writeGuard{}, stored: texasPurchase()}
	tool := newDeleteFuelPurchaseTool(fuel, fakeJurisdictions{})
	params := executeParams(map[string]any{paramFuelPurchaseID: fuel.stored.ID.String()})

	preview := previewWithoutWrites(t, fuel.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, agent.PreviewOperationDelete, previewChange(t, preview, 0).Operation)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, fuel.deleted)
	assert.Equal(t, fuel.stored.Version, fuel.deleted.Version)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)
	assert.False(t, tool.Policy().Reversible)
}

func TestAssignFuelCard_SaysWhenItActivatesTheCard(t *testing.T) {
	t.Parallel()

	fuel := &fakeFuel{guard: &writeGuard{}, card: &fuelpurchase.FuelCard{
		ID:       pulid.MustNew("fcard_"),
		Provider: fuelpurchase.CardProviderComdata,
		LastFour: "4411",
		Status:   fuelpurchase.CardStatusSuspended,
		Version:  1,
	}}
	tool := newAssignFuelCardTool(fuel)
	tractorID := pulid.MustNew("trac_")
	params := executeParams(map[string]any{
		paramFuelCardID:        fuel.card.ID.String(),
		paramAssignedTractorID: tractorID.String(),
	})

	preview := previewWithoutWrites(t, fuel.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "and make it active")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, fuel.assigned)
	assert.Equal(t, int64(1), fuel.assigned.Version)
	assert.Equal(t, tractorID, *fuel.assigned.AssignedTractorID)

	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), executeParams(
		map[string]any{paramFuelCardID: fuel.card.ID.String()},
	)))
}

func fuelBatch() *fuelpurchase.ImportBatch {
	return &fuelpurchase.ImportBatch{
		ID:              pulid.MustNew("fpib_"),
		Provider:        fuelpurchase.CardProviderEFS,
		Origin:          fuelpurchase.ImportOriginUpload,
		FileName:        "september.csv",
		Status:          fuelpurchase.ImportStatusParsed,
		DefaultCurrency: "USD",
		RowCount:        2,
		Version:         5,
	}
}

func TestFuelImportTools_PlanWithTheVersionTheyReadAndWriteOnlyOnExecute(t *testing.T) {
	t.Parallel()

	fuel := &fakeFuel{guard: &writeGuard{}, batch: fuelBatch()}
	params := executeParams(map[string]any{
		paramFuelImportID: fuel.batch.ID.String(),
		paramReason:       "Loaded twice",
	})

	commit := newCommitFuelPurchaseImportTool(fuel)
	preview := previewWithoutWrites(t, fuel.guard, func() (*agent.ToolPreview, error) {
		return commit.(serviceports.ToolPreviewer).Preview(t.Context(),
			executeParams(map[string]any{paramFuelImportID: fuel.batch.ID.String()}))
	})
	assert.Contains(t, preview.Summary, "Would commit 2 fuel purchases from september.csv")
	require.NoError(t, commit.Execute(t.Context(),
		executeParams(map[string]any{paramFuelImportID: fuel.batch.ID.String()})))
	assert.Equal(t, int64(5), fuel.committed.Version)

	resolve := newResolveFuelPurchaseImportRowsTool(fuel)
	preview = previewWithoutWrites(t, fuel.guard, func() (*agent.ToolPreview, error) {
		return resolve.(serviceports.ToolPreviewer).Preview(t.Context(),
			executeParams(map[string]any{paramFuelImportID: fuel.batch.ID.String()}))
	})
	assert.Contains(t, preview.Summary, "2 now resolve and 1 still wait")

	discard := newDiscardFuelPurchaseImportTool(fuel)
	preview = previewWithoutWrites(t, fuel.guard, func() (*agent.ToolPreview, error) {
		return discard.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "Discarded", fieldByPath(t, previewChange(t, preview, 0), "status").After)
	require.NoError(t, discard.Execute(t.Context(), params))
	assert.Equal(t, "Loaded twice", fuel.discarded.Reason)

	refused := newCommitFuelPurchaseImportTool(&fakeFuel{
		batch:   fuelBatch(),
		refusal: errortypes.NewValidationError("status", errortypes.ErrInvalid, "Committed"),
	})
	require.Error(t, refused.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{paramFuelImportID: fuel.batch.ID.String()})))
}

type fakeFuelPrices struct {
	guard    *writeGuard
	stored   *fuelsurcharge.FuelIndexPrice
	added    *fuelsurcharge.FuelIndexPrice
	replaced *fuelsurcharge.FuelIndexPrice
}

func (f *fakeFuelPrices) PlanAddManualPrice(
	_ context.Context,
	entity *fuelsurcharge.FuelIndexPrice,
	_ pulid.ID,
) (*fuelsurcharge.FuelIndexPrice, error) {
	planned := *entity
	planned.Currency = "USD"
	planned.IsManual = true

	return &planned, nil
}

func (f *fakeFuelPrices) AddManualPrice(
	_ context.Context,
	entity *fuelsurcharge.FuelIndexPrice,
	_ pulid.ID,
) (*fuelsurcharge.FuelIndexPrice, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.added = entity

	return entity, nil
}

func (f *fakeFuelPrices) PlanUpdateManualPrice(
	_ context.Context,
	entity *fuelsurcharge.FuelIndexPrice,
	_ pulid.ID,
) (*fuelsurchargeservice.PriceChange, error) {
	after := *entity
	after.Currency = f.stored.Currency

	return &fuelsurchargeservice.PriceChange{Before: f.stored, After: &after}, nil
}

func (f *fakeFuelPrices) UpdateManualPrice(
	_ context.Context,
	entity *fuelsurcharge.FuelIndexPrice,
	_ pulid.ID,
) (*fuelsurcharge.FuelIndexPrice, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.replaced = entity

	return entity, nil
}

func TestFuelIndexPriceTools_OnlyAPersonMovesAPrice(t *testing.T) {
	t.Parallel()

	prices := &fakeFuelPrices{guard: &writeGuard{}, stored: &fuelsurcharge.FuelIndexPrice{
		ID:        pulid.MustNew("fip_"),
		PriceDate: "2026-09-21",
		Price:     decimal.RequireFromString("3.799"),
		Currency:  "USD",
	}}

	record := newRecordFuelIndexPriceTool(prices)
	params := executeParams(map[string]any{
		paramFuelIndexID: pulid.MustNew("fidx_").String(),
		paramPriceDate:   "2026-09-28",
		paramPrice:       "3.899",
	})
	preview := previewWithoutWrites(t, prices.guard, func() (*agent.ToolPreview, error) {
		return record.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would record 3.899 USD a gallon for 2026-09-28")
	require.ErrorIs(t, record.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	assert.Nil(t, prices.added)
	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, record.Execute(t.Context(), params))
	require.NotNil(t, prices.added)

	correct := newCorrectFuelIndexPriceTool(prices)
	fix := executeParams(map[string]any{
		paramFuelIndexPriceID: prices.stored.ID.String(),
		paramPriceDate:        "2026-09-21",
		paramPrice:            "3.829",
	})
	preview = previewWithoutWrites(t, prices.guard, func() (*agent.ToolPreview, error) {
		return correct.(serviceports.ToolPreviewer).Preview(t.Context(), fix)
	})
	assert.Contains(t, preview.Summary, "from 3.799 to 3.829")

	require.Error(t, correct.(serviceports.ToolValidator).Validate(t.Context(), executeParams(
		map[string]any{
			paramFuelIndexPriceID: prices.stored.ID.String(),
			paramPriceDate:        "Sept 21",
			paramPrice:            "3.829",
		},
	)))

	for _, tool := range []serviceports.AgentTool{record, correct} {
		policy := tool.Policy()
		assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress, policy.Name)
		assert.Equal(t, agent.TierPropose, policy.MaxTier, policy.Name)
		assert.Equal(t, permission.ResourceFuelSurchargeProgram, policy.Resource, policy.Name)
	}
}

func TestRecordFuelPurchase_WorksOutTheTotalFromTheUnitPrice(t *testing.T) {
	t.Parallel()

	fuel := &fakeFuel{guard: &writeGuard{}}
	tool := newRecordFuelPurchaseTool(fuel, fakeJurisdictions{})
	params := purchaseParams()
	delete(params, paramTotalAmount)
	params[paramQuantity] = "112"
	params[paramUnitPrice] = "3.89"

	_, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), executeParams(params))
	require.NoError(t, err)
	require.NotNil(t, fuel.created)
	assert.Equal(t, int64(43_568), fuel.created.Purchase.TotalAmountMinor)

	delete(params, paramUnitPrice)
	err = tool.(serviceports.ToolValidator).Validate(t.Context(), executeParams(params))
	require.Error(t, err)
	assert.Contains(t, err.Error(), paramTotalAmount)
}

func TestRecordFuelPurchase_RecordsADayAloneAtNoonInTheOrganizationsZone(t *testing.T) {
	t.Parallel()

	fuel := &fakeFuel{guard: &writeGuard{}}
	tool := newRecordFuelPurchaseTool(fuel, fakeJurisdictions{})
	params := purchaseParams()
	params[paramPurchasedAt] = "2026-10-08"
	execute := executeParams(params)
	execute.Timezone = "America/Chicago"

	_, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), execute)
	require.NoError(t, err)
	require.NotNil(t, fuel.created)

	recorded := time.Unix(fuel.created.Purchase.PurchasedAt, 0).In(time.FixedZone("CDT", -5*3600))
	assert.Equal(t, "2026-10-08 12:00", recorded.Format("2006-01-02 15:04"))
}
