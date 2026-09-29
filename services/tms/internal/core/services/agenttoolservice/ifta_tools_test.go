package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/iftaservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeIFTA struct {
	fakeJurisdictions
	guard       *writeGuard
	stored      *ifta.Return
	entry       *ifta.JurisdictionMileageEntry
	refusal     error
	generated   *iftaservice.GenerateReturnRequest
	recomputed  *iftaservice.ReturnActionRequest
	amended     *iftaservice.AmendReturnRequest
	deleted     *iftaservice.ReturnActionRequest
	createdMile *ifta.JurisdictionMileageEntry
	updatedMile *ifta.JurisdictionMileageEntry
	deletedMile *iftaservice.DeleteMileageEntryRequest
}

func draftReturn(status ifta.ReturnStatus) *ifta.Return {
	return &ifta.Return{
		ID:           pulid.MustNew("ifr_"),
		Year:         2026,
		Quarter:      3,
		Status:       status,
		TotalMiles:   decimal.RequireFromString("48210.5"),
		TotalGallons: decimal.RequireFromString("7200.25"),
		NetDueMinor:  41_220,
		CurrencyCode: "USD",
		Version:      4,
	}
}

func (f *fakeIFTA) Get(context.Context, pagination.TenantInfo, pulid.ID) (*ifta.Return, error) {
	copied := *f.stored

	return &copied, nil
}

func (f *fakeIFTA) PlanGenerate(
	context.Context,
	*iftaservice.GenerateReturnRequest,
) (*ifta.Return, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return draftReturn(ifta.ReturnStatusDraft), nil
}

func (f *fakeIFTA) Generate(
	_ context.Context,
	req *iftaservice.GenerateReturnRequest,
) (*ifta.Return, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.generated = req

	return draftReturn(ifta.ReturnStatusDraft), nil
}

func (f *fakeIFTA) PlanRecompute(
	context.Context,
	*iftaservice.ReturnActionRequest,
) (*iftaservice.ReturnChange, error) {
	after := *f.stored
	after.TotalMiles = decimal.RequireFromString("48900")

	return &iftaservice.ReturnChange{Before: f.stored, After: &after}, nil
}

func (f *fakeIFTA) Recompute(
	_ context.Context,
	req *iftaservice.ReturnActionRequest,
) (*ifta.Return, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.recomputed = req

	return f.stored, nil
}

func (f *fakeIFTA) PlanAmend(
	context.Context,
	*iftaservice.AmendReturnRequest,
) (*iftaservice.ReturnAmendment, error) {
	amendment := draftReturn(ifta.ReturnStatusDraft)
	amendment.AmendmentNumber = 1

	return &iftaservice.ReturnAmendment{Filed: f.stored, Draft: amendment}, nil
}

func (f *fakeIFTA) Amend(
	_ context.Context,
	req *iftaservice.AmendReturnRequest,
) (*ifta.Return, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.amended = req

	return f.stored, nil
}

func (f *fakeIFTA) PlanDelete(
	context.Context,
	*iftaservice.ReturnActionRequest,
) (*ifta.Return, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return f.stored, nil
}

func (f *fakeIFTA) Delete(_ context.Context, req *iftaservice.ReturnActionRequest) error {
	if err := f.guard.write(); err != nil {
		return err
	}
	f.deleted = req

	return nil
}

func (f *fakeIFTA) PeriodInfo(
	_ context.Context,
	_ pagination.TenantInfo,
	year, quarter int,
) (iftaservice.PeriodInfo, error) {
	return iftaservice.PeriodInfo{
		Period: ifta.NewPeriod(year, quarter),
		Start:  1_782_864_000,
		End:    1_790_812_799,
	}, nil
}

func (f *fakeIFTA) GetMileageEntry(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*ifta.JurisdictionMileageEntry, error) {
	copied := *f.entry

	return &copied, nil
}

func withPeriod(entry *ifta.JurisdictionMileageEntry) *ifta.JurisdictionMileageEntry {
	planned := *entry
	planned.Year, planned.Quarter = 2026, 3

	return &planned
}

func (f *fakeIFTA) PlanCreateMileageEntry(
	_ context.Context,
	entity *ifta.JurisdictionMileageEntry,
	_ pulid.ID,
) (*ifta.JurisdictionMileageEntry, error) {
	return withPeriod(entity), nil
}

func (f *fakeIFTA) CreateMileageEntry(
	_ context.Context,
	entity *ifta.JurisdictionMileageEntry,
	_ pulid.ID,
) (*ifta.JurisdictionMileageEntry, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.createdMile = entity

	return entity, nil
}

func (f *fakeIFTA) PlanUpdateMileageEntry(
	_ context.Context,
	entity *ifta.JurisdictionMileageEntry,
	_ pulid.ID,
) (*iftaservice.MileageEntryChange, error) {
	return &iftaservice.MileageEntryChange{Before: f.entry, After: withPeriod(entity)}, nil
}

func (f *fakeIFTA) UpdateMileageEntry(
	_ context.Context,
	entity *ifta.JurisdictionMileageEntry,
	_ pulid.ID,
) (*ifta.JurisdictionMileageEntry, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.updatedMile = entity

	return entity, nil
}

func (f *fakeIFTA) PlanDeleteMileageEntry(
	context.Context,
	*iftaservice.DeleteMileageEntryRequest,
) (*ifta.JurisdictionMileageEntry, error) {
	return f.entry, nil
}

func (f *fakeIFTA) DeleteMileageEntry(
	_ context.Context,
	req *iftaservice.DeleteMileageEntryRequest,
) error {
	if err := f.guard.write(); err != nil {
		return err
	}
	f.deletedMile = req

	return nil
}

type fakeMileRouter struct {
	guard        *writeGuard
	recalculated *serviceports.RecalculateMoveJurisdictionMilesRequest
	backfills    []serviceports.BackfillJurisdictionMilesRequest
	unattributed int
}

func (f *fakeMileRouter) PlanMoveJurisdictionMiles(
	_ context.Context,
	req *serviceports.RecalculateMoveJurisdictionMilesRequest,
) (*serviceports.MoveJurisdictionMilesPlan, error) {
	return &serviceports.MoveJurisdictionMilesPlan{Move: &shipment.ShipmentMove{
		ID:                req.ShipmentMoveID,
		Stops:             []*shipment.Stop{{}, {}},
		JurisdictionMiles: []*shipment.ShipmentMoveJurisdictionMile{{}, {}, {}},
	}}, nil
}

func (f *fakeMileRouter) RecalculateMoveJurisdictionMiles(
	_ context.Context,
	req serviceports.RecalculateMoveJurisdictionMilesRequest,
) ([]*shipment.ShipmentMoveJurisdictionMile, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.recalculated = &req

	return nil, nil
}

func (f *fakeMileRouter) BackfillJurisdictionMiles(
	_ context.Context,
	req serviceports.BackfillJurisdictionMilesRequest,
) (*serviceports.BackfillJurisdictionMilesResult, error) {
	if !req.DryRun {
		if err := f.guard.write(); err != nil {
			return nil, err
		}
	}
	f.backfills = append(f.backfills, req)

	return &serviceports.BackfillJurisdictionMilesResult{
		DryRun:            req.DryRun,
		Started:           !req.DryRun,
		UnattributedMoves: f.unattributed,
		UnattributedMiles: decimal.RequireFromString("1234.56"),
	}, nil
}

func TestGenerateIFTAReturn_PreviewsTheDraftAndFilesNothing(t *testing.T) {
	t.Parallel()

	returns := &fakeIFTA{guard: &writeGuard{}}
	tool := newGenerateIFTAReturnTool(returns)
	params := executeParams(map[string]any{paramYear: 2026, paramQuarter: 3})

	preview := previewWithoutWrites(t, returns.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would generate the Q3 2026 IFTA return as a draft")
	assert.Contains(t, preview.Summary, "net due 412.20 USD")
	assert.Equal(t, agent.PreviewOperationCreate, previewChange(t, preview, 0).Operation)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, ifta.NewPeriod(2026, 3), returns.generated.Period)

	for name, bad := range map[string]map[string]any{
		"a fifth quarter": {paramYear: 2026, paramQuarter: 5},
		"no year":         {paramQuarter: 1},
	} {
		require.Error(t, tool.(serviceports.ToolValidator).Validate(
			t.Context(),
			executeParams(bad),
		), name)
	}

	blocked := newGenerateIFTAReturnTool(&fakeIFTA{refusal: errortypes.NewBusinessError(
		"A return is already filed for the quarter",
	)})
	preview, err := blocked.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
	require.Error(t, blocked.(serviceports.ToolValidator).Validate(t.Context(), params))
}

func TestIFTAReturnTools_ActOnTheVersionTheyRead(t *testing.T) {
	t.Parallel()

	returns := &fakeIFTA{guard: &writeGuard{}, stored: draftReturn(ifta.ReturnStatusDraft)}
	params := executeParams(map[string]any{paramIFTAReturnID: returns.stored.ID.String()})

	recompute := newRecomputeIFTAReturnTool(returns)
	preview := previewWithoutWrites(t, returns.guard, func() (*agent.ToolPreview, error) {
		return recompute.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "48900.00",
		fieldByPath(t, previewChange(t, preview, 0), "totalMiles").After)
	require.NoError(t, recompute.Execute(t.Context(), params))
	assert.Equal(t, returns.stored.Version, returns.recomputed.Version)
	assert.Equal(t, agent.TierAutoExecute, recompute.Policy().MaxTier)

	remove := newDeleteIFTAReturnTool(returns)
	preview = previewWithoutWrites(t, returns.guard, func() (*agent.ToolPreview, error) {
		return remove.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "Would delete the draft Q3 2026 IFTA return.", preview.Summary)
	require.NoError(t, remove.Execute(t.Context(), params))
	assert.Equal(t, returns.stored.Version, returns.deleted.Version)

	target, ok := remove.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceIFTAReturn, target.Resource)
}

func TestAmendIFTAReturn_IsOnlyEverProposedWithAReason(t *testing.T) {
	t.Parallel()

	returns := &fakeIFTA{guard: &writeGuard{}, stored: draftReturn(ifta.ReturnStatusFiled)}
	tool := newAmendIFTAReturnTool(returns)
	params := executeParams(map[string]any{
		paramIFTAReturnID: returns.stored.ID.String(),
		paramReason:       "Texas purchases from the September statement were left out",
	})

	preview := previewWithoutWrites(t, returns.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would open amendment 1 of the filed Q3 2026 IFTA return")
	assert.Contains(t, preview.Summary, "The filed return is not changed.")

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, "Texas purchases from the September statement were left out",
		returns.amended.Reason)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)

	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), executeParams(
		map[string]any{paramIFTAReturnID: returns.stored.ID.String()},
	)))
}

func texasMiles() *ifta.JurisdictionMileageEntry {
	return &ifta.JurisdictionMileageEntry{
		ID:             pulid.MustNew("ifme_"),
		TractorID:      pulid.MustNew("trac_"),
		JurisdictionID: texasID,
		TraveledAt:     1_784_000_000,
		Year:           2026,
		Quarter:        3,
		Miles:          decimal.RequireFromString("212.5"),
		Loaded:         true,
		Source:         ifta.MileageSourceManual,
		Version:        6,
	}
}

func TestIFTAMileageTools_RecordCorrectAndProposeDeletingMiles(t *testing.T) {
	t.Parallel()

	entries := &fakeIFTA{guard: &writeGuard{}, entry: texasMiles()}

	record := newRecordIFTAMileageEntryTool(entries)
	params := executeParams(map[string]any{
		paramTractorID:      pulid.MustNew("trac_").String(),
		paramJurisdictionID: texasID.String(),
		paramTraveledAt:     "2026-08-03T10:00:00Z",
		paramMiles:          "212.5",
		paramNotes:          "Trip sheet 4411",
	})
	preview := previewWithoutWrites(t, entries.guard, func() (*agent.ToolPreview, error) {
		return record.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "Would record 212.50 miles in TX (Texas), counted in Q3 2026.",
		preview.Summary)
	require.NoError(t, record.Execute(t.Context(), params))
	assert.True(t, entries.createdMile.Loaded)
	assert.Equal(t, ifta.MileageSourceManual, entries.createdMile.Source)
	require.NotNil(t, record.Policy().TaintHold)

	params.Params[paramMiles] = "-4"
	require.Error(t, record.(serviceports.ToolValidator).Validate(t.Context(), params))

	correct := newCorrectIFTAMileageEntryTool(entries)
	fix := executeParams(map[string]any{
		paramIFTAMileageEntryID: entries.entry.ID.String(),
		paramLoaded:             false,
	})
	preview = previewWithoutWrites(t, entries.guard, func() (*agent.ToolPreview, error) {
		return correct.(serviceports.ToolPreviewer).Preview(t.Context(), fix)
	})
	assert.Equal(t, false, fieldByPath(t, previewChange(t, preview, 0), "loaded").After)
	require.NoError(t, correct.Execute(t.Context(), fix))
	assert.False(t, entries.updatedMile.Loaded)
	assert.Equal(t, entries.entry.Miles, entries.updatedMile.Miles)
	assert.Equal(t, entries.entry.Version, entries.updatedMile.Version)

	remove := newDeleteIFTAMileageEntryTool(entries)
	gone := executeParams(map[string]any{paramIFTAMileageEntryID: entries.entry.ID.String()})
	preview = previewWithoutWrites(t, entries.guard, func() (*agent.ToolPreview, error) {
		return remove.(serviceports.ToolPreviewer).Preview(t.Context(), gone)
	})
	assert.Equal(t, "Would delete the 212.50 miles in Q3 2026 driven in TX (Texas).",
		preview.Summary)
	require.NoError(t, remove.Execute(t.Context(), gone))
	assert.Equal(t, entries.entry.Version, entries.deletedMile.Version)
	assert.Equal(t, agent.TierPropose, remove.Policy().MaxTier)
}

func TestRecalculateMoveJurisdictionMiles_AsksAgainOnlyOnExecute(t *testing.T) {
	t.Parallel()

	router := &fakeMileRouter{guard: &writeGuard{}}
	tool := newRecalculateMoveJurisdictionMilesTool(router)
	moveID := pulid.MustNew("smv_")
	params := executeParams(map[string]any{paramShipmentMoveID: moveID.String()})

	preview := previewWithoutWrites(t, router.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "across 2 stops and replace its 3 jurisdiction rows")

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, moveID, router.recalculated.ShipmentMoveID)
	assert.Equal(t, permission.ResourceShipmentMove, tool.Policy().Resource)
}

func TestBackfillJurisdictionMiles_PreviewsWithADryRunAndStartsOnExecute(t *testing.T) {
	t.Parallel()

	router := &fakeMileRouter{guard: &writeGuard{}, unattributed: 14}
	tool := newBackfillJurisdictionMilesTool(&fakeIFTA{}, router)
	params := executeParams(map[string]any{paramYear: 2026, paramQuarter: 3, paramMaxMoves: 50})

	preview := previewWithoutWrites(t, router.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would start routing 14 completed moves in Q3 2026")
	require.Len(t, router.backfills, 1)
	assert.True(t, router.backfills[0].DryRun)
	assert.Equal(t, 50, router.backfills[0].MaxMoves)
	assert.Equal(t, int64(1_782_864_000), router.backfills[0].Start)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.Len(t, router.backfills, 2)
	assert.False(t, router.backfills[1].DryRun)

	settled := newBackfillJurisdictionMilesTool(&fakeIFTA{}, &fakeMileRouter{})
	preview, err := settled.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "nothing would be routed")

	params.Params[paramMaxMoves] = maxBackfillMoves + 1
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), params))
	assert.Equal(t, permission.OpManage, tool.Policy().Operation)
}
