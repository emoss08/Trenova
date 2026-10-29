package iftatools

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentdistanceports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/iftaservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramIFTAReturnID = "iftaReturnId"

const paramIFTAMileageEntryID = "iftaMileageEntryId"

const paramYear = "year"

const paramQuarter = "quarter"

const paramTraveledAt = "traveledAt"

const paramMiles = "miles"

const paramLoaded = "loaded"

const paramMaxMoves = "maxMoves"

const minIFTAYear = 2000

const maxIFTAYear = 2100

const quartersInYear = 4

const maxAmendReason = 1000

const maxMileageNotes = 2000

const maxBackfillMoves = 5000

const iftaReturnSupplier = "The IFTA return, from fuel_list_ifta_returns. Never guess one."

const iftaMileageSupplier = "The mileage entry, from fuel_list_ifta_mileage_entries. Never guess one."

const iftaMileageTaintHold = "Miles read from a log or a message someone outside sent are proposed, since the IFTA return is computed from them."

var iftaMileageDates = map[string]assistantartifact.DisplayType{
	paramTraveledAt: assistantartifact.DisplayDateTime,
}

type IftaReturnKeeper interface {
	Get(ctx context.Context, tenantInfo pagination.TenantInfo, id pulid.ID) (*ifta.Return, error)
	PlanGenerate(
		ctx context.Context,
		req *iftaservice.GenerateReturnRequest,
	) (*ifta.Return, error)
	Generate(ctx context.Context, req *iftaservice.GenerateReturnRequest) (*ifta.Return, error)
	PlanRecompute(
		ctx context.Context,
		req *iftaservice.ReturnActionRequest,
	) (*iftaservice.ReturnChange, error)
	Recompute(ctx context.Context, req *iftaservice.ReturnActionRequest) (*ifta.Return, error)
	PlanAmend(
		ctx context.Context,
		req *iftaservice.AmendReturnRequest,
	) (*iftaservice.ReturnAmendment, error)
	Amend(ctx context.Context, req *iftaservice.AmendReturnRequest) (*ifta.Return, error)
	PlanDelete(ctx context.Context, req *iftaservice.ReturnActionRequest) (*ifta.Return, error)
	Delete(ctx context.Context, req *iftaservice.ReturnActionRequest) error
	PeriodInfo(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		year, quarter int,
	) (iftaservice.PeriodInfo, error)
}

type IftaMileageKeeper interface {
	GetMileageEntry(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*ifta.JurisdictionMileageEntry, error)
	GetJurisdiction(ctx context.Context, id pulid.ID) (*ifta.Jurisdiction, error)
	PlanCreateMileageEntry(
		ctx context.Context,
		entity *ifta.JurisdictionMileageEntry,
		userID pulid.ID,
	) (*ifta.JurisdictionMileageEntry, error)
	CreateMileageEntry(
		ctx context.Context,
		entity *ifta.JurisdictionMileageEntry,
		userID pulid.ID,
	) (*ifta.JurisdictionMileageEntry, error)
	PlanUpdateMileageEntry(
		ctx context.Context,
		entity *ifta.JurisdictionMileageEntry,
		userID pulid.ID,
	) (*iftaservice.MileageEntryChange, error)
	UpdateMileageEntry(
		ctx context.Context,
		entity *ifta.JurisdictionMileageEntry,
		userID pulid.ID,
	) (*ifta.JurisdictionMileageEntry, error)
	PlanDeleteMileageEntry(
		ctx context.Context,
		req *iftaservice.DeleteMileageEntryRequest,
	) (*ifta.JurisdictionMileageEntry, error)
	DeleteMileageEntry(ctx context.Context, req *iftaservice.DeleteMileageEntryRequest) error
}

type JurisdictionMileRouter interface {
	PlanMoveJurisdictionMiles(
		ctx context.Context,
		req *shipmentdistanceports.RecalculateMoveJurisdictionMilesRequest,
	) (*shipmentdistanceports.MoveJurisdictionMilesPlan, error)
	RecalculateMoveJurisdictionMiles(
		ctx context.Context,
		req shipmentdistanceports.RecalculateMoveJurisdictionMilesRequest,
	) ([]*shipment.ShipmentMoveJurisdictionMile, error)
	BackfillJurisdictionMiles(
		ctx context.Context,
		req *shipmentdistanceports.BackfillJurisdictionMilesRequest,
	) (*shipmentdistanceports.BackfillJurisdictionMilesResult, error)
}

type iftaReturnView struct {
	Status              string `json:"status"`
	AmendmentNumber     int    `json:"amendmentNumber,omitempty"`
	TotalMiles          string `json:"totalMiles"`
	TotalTaxableMiles   string `json:"totalTaxableMiles"`
	TotalGallons        string `json:"totalGallons"`
	TotalTaxPaidGallons string `json:"totalTaxPaidGallons"`
	TaxDue              string `json:"taxDue"`
	NetDue              string `json:"netDue"`
	Lines               int    `json:"jurisdictionLines"`
	Problems            int    `json:"problems"`
	BlockingProblems    int    `json:"blockingProblems"`
}

func iftaReturnViewOf(ret *ifta.Return) *iftaReturnView {
	return &iftaReturnView{
		Status:              string(ret.Status),
		AmendmentNumber:     ret.AmendmentNumber,
		TotalMiles:          ret.TotalMiles.StringFixed(2),
		TotalTaxableMiles:   ret.TotalTaxableMiles.StringFixed(2),
		TotalGallons:        ret.TotalGallons.StringFixed(3),
		TotalTaxPaidGallons: ret.TotalTaxPaidGallons.StringFixed(3),
		TaxDue:              ret.TaxDue().StringFixed(2),
		NetDue:              ret.NetDue().StringFixed(2),
		Lines:               len(ret.Lines),
		Problems:            len(ret.Problems),
		BlockingProblems:    len(ret.BlockingProblems()),
	}
}

func iftaReturnRecord(ret *ifta.Return) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceIFTAReturn,
		ID:       ret.ID,
		Label:    ret.Period().Label() + " IFTA return",
		Version:  changepreview.PinnedVersion(ret.Version),
	}
}

func iftaFigures(ret *ifta.Return) string {
	figures := fmt.Sprintf("%s miles, %s gallons, net due %s %s",
		ret.TotalMiles.StringFixed(2),
		ret.TotalGallons.StringFixed(3),
		ret.NetDue().StringFixed(2),
		ret.CurrencyCode,
	)
	if blocking := len(ret.BlockingProblems()); blocking > 0 {
		figures += fmt.Sprintf(
			"; %s block finalizing it",
			changepreview.CountOf(blocking, "problem"),
		)
	}

	return figures
}

func iftaInternalSpec(spec *toolspec.ReceivableSpec) *toolspec.ReceivableSpec {
	return toolspec.FuelInternalSpec(spec)
}

func targetIFTAReturn(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramIFTAReturnID, permission.ResourceIFTAReturn)
}

func iftaPeriodFrom(params *agenttoolports.ToolExecuteParams) (ifta.Period, error) {
	year, err := toolparams.RequireIntInRange(params.Params, paramYear, minIFTAYear, maxIFTAYear)
	if err != nil {
		return ifta.Period{}, err
	}
	quarter, err := toolparams.RequireIntInRange(params.Params, paramQuarter, 1, quartersInYear)
	if err != nil {
		return ifta.Period{}, err
	}

	return ifta.NewPeriod(year, quarter), nil
}

func iftaPeriodProperties() map[string]any {
	return map[string]any{
		paramYear: toolparams.IntegerProperty(
			"The year of the quarter.",
			minIFTAYear,
			maxIFTAYear,
		),
		paramQuarter: toolparams.IntegerProperty("The quarter, 1 to 4.", 1, quartersInYear),
	}
}

func NewGenerateIFTAReturnTool(returns IftaReturnKeeper) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(iftaInternalSpec(&toolspec.ReceivableSpec{
		Name: "fuel_generate_ifta_return",
		Description: "Generate a quarter's IFTA fuel tax return as a draft, computed from " +
			"the jurisdiction miles and fuel purchases on file. It files nothing: a person " +
			"finalizes and files it. Refused while a draft or finalized return exists for the " +
			"quarter, and once one is filed; use fuel_recalculate_ifta_return or fuel_amend_ifta_return.",
		Resource:   permission.ResourceIFTAReturn,
		Operation:  permission.OpCreate,
		Reversible: true,
		Rationale: "Computes a draft return inside Trenova; nothing is filed or sent, and " +
			"fuel_delete_ifta_return removes the draft.",
		Properties:  iftaPeriodProperties(),
		Required:    []string{paramYear, paramQuarter},
		SearchTerms: []string{toolparams.SearchTermIFTA, "fuel tax", "quarterly return", "quarter"},
	}), toolspec.ReceivablePlan[*iftaservice.GenerateReturnRequest, *ifta.Return]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*iftaservice.GenerateReturnRequest, error) {
			period, err := iftaPeriodFrom(params)
			if err != nil {
				return nil, err
			}

			return &iftaservice.GenerateReturnRequest{
				TenantInfo: toolparams.TenantFrom(params),
				Period:     period,
				UserID:     params.Actor.UserID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *iftaservice.GenerateReturnRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*ifta.Return, error) {
			return returns.PlanGenerate(ctx, req)
		},
		Refused: func(req *iftaservice.GenerateReturnRequest) string {
			return fmt.Sprintf("Would generate the %s IFTA return.", req.Period.Label())
		},
		Render: func(
			req *iftaservice.GenerateReturnRequest,
			planned *ifta.Return,
		) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Create(toolpreview.Record{
				Resource: permission.ResourceIFTAReturn,
				Label:    req.Period.Label() + " IFTA return",
			}, iftaReturnViewOf(planned))
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would generate the %s IFTA return as a draft: %s.",
				req.Period.Label(), iftaFigures(planned),
			), change), nil
		},
		Run: func(
			ctx context.Context,
			req *iftaservice.GenerateReturnRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := returns.Generate(ctx, req)

			return nil, err
		},
	})
}

func returnActionRequest(
	ctx context.Context,
	returns IftaReturnKeeper,
	params *agenttoolports.ToolExecuteParams,
	id pulid.ID,
) (*iftaservice.ReturnActionRequest, error) {
	stored, err := returns.Get(ctx, toolparams.TenantFrom(params), id)
	if err != nil {
		return nil, err
	}

	return &iftaservice.ReturnActionRequest{
		TenantInfo: toolparams.TenantFrom(params),
		ID:         stored.ID,
		Version:    stored.Version,
		UserID:     params.Actor.UserID,
	}, nil
}

func iftaReturnIDFrom(params *agenttoolports.ToolExecuteParams) (pulid.ID, error) {
	return toolparams.RequirePulid(params.Params, paramIFTAReturnID)
}

func NewRecomputeIFTAReturnTool(returns IftaReturnKeeper) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(iftaInternalSpec(&toolspec.ReceivableSpec{
		Name:        "fuel_recalculate_ifta_return",
		Recipe:      []string{"fuel_list_ifta_returns", "fuel_recalculate_ifta_return"},
		SearchTerms: []string{"refresh draft return", "recompute after corrections"},
		Description: "Recompute a draft IFTA return from the miles, purchases and tax rates " +
			"on file now, after purchases or mileage were corrected. Only a draft can be " +
			"recomputed; a finalized return must be reopened by a person first.",
		Resource:    permission.ResourceIFTAReturn,
		Operation:   permission.OpUpdate,
		DefaultTier: agent.TierActWithApproval,
		MaxTier:     agent.TierAutoExecute,
		Reversible:  true,
		Rationale: "Re-derives a draft return's figures inside Trenova from what is on file; " +
			"nothing is filed, and running it again gives the same figures.",
		Properties: map[string]any{
			paramIFTAReturnID: agenttoolschema.KindID(
				iftaReturnSupplier,
				recordkind.IFTAReturn,
			),
		},
		Required: []string{paramIFTAReturnID},
		Target:   targetIFTAReturn,
	}), toolspec.ReceivablePlan[pulid.ID, *iftaservice.ReturnChange]{
		Request: iftaReturnIDFrom,
		Plan: func(
			ctx context.Context,
			id pulid.ID,
			params *agenttoolports.ToolExecuteParams,
		) (*iftaservice.ReturnChange, error) {
			req, err := returnActionRequest(ctx, returns, params, id)
			if err != nil {
				return nil, err
			}

			return returns.PlanRecompute(ctx, req)
		},
		Refused: func(pulid.ID) string {
			return "Would recompute an IFTA return."
		},
		Render: func(_ pulid.ID, plan *iftaservice.ReturnChange) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Changed(
				iftaReturnRecord(plan.Before),
				iftaReturnViewOf(plan.Before),
				iftaReturnViewOf(plan.After),
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would recompute the %s: %s.",
				iftaReturnRecord(plan.Before).Label, iftaFigures(plan.After),
			), change), nil
		},
		Run: func(
			ctx context.Context,
			id pulid.ID,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			req, err := returnActionRequest(ctx, returns, params, id)
			if err != nil {
				return nil, err
			}
			_, err = returns.Recompute(ctx, req)

			return nil, err
		},
	})
}

func NewAmendIFTAReturnTool(returns IftaReturnKeeper) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(iftaInternalSpec(&toolspec.ReceivableSpec{
		Name: "fuel_amend_ifta_return",
		Description: "Propose opening an amendment of a filed IFTA return, with the reason: " +
			"a new draft for the same quarter, computed from what is on file now. The filed " +
			"return is left as it was, and a person finalizes and files the amendment.",
		Resource:  permission.ResourceIFTAReturn,
		Operation: permission.OpCreate,
		MaxTier:   agent.TierPropose,
		Rationale: "Starts a correction of a return already filed with the jurisdictions; " +
			"it files nothing, but a person decides to reopen a filing.",
		Properties: map[string]any{
			paramIFTAReturnID: agenttoolschema.KindID(iftaReturnSupplier+" It must be Filed.",
				recordkind.IFTAReturn),
			toolparams.ParamReason: toolparams.StringProperty(
				"What was wrong with the filed return, at least ten "+
					"characters, as the auditor will read it.",
				maxAmendReason,
			),
		},
		Required: []string{paramIFTAReturnID, toolparams.ParamReason},
		Target:   targetIFTAReturn,
	}), toolspec.ReceivablePlan[*iftaservice.AmendReturnRequest, *iftaservice.ReturnAmendment]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*iftaservice.AmendReturnRequest, error) {
			id, err := toolparams.RequirePulid(params.Params, paramIFTAReturnID)
			if err != nil {
				return nil, err
			}
			reason, err := toolparams.RequireBoundedText(
				params.Params,
				toolparams.ParamReason,
				maxAmendReason,
			)
			if err != nil {
				return nil, err
			}

			return &iftaservice.AmendReturnRequest{
				TenantInfo: toolparams.TenantFrom(params),
				ID:         id,
				Reason:     reason,
				UserID:     params.Actor.UserID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *iftaservice.AmendReturnRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*iftaservice.ReturnAmendment, error) {
			return returns.PlanAmend(ctx, req)
		},
		Refused: func(*iftaservice.AmendReturnRequest) string {
			return "Would open an amendment of an IFTA return."
		},
		Render: func(
			_ *iftaservice.AmendReturnRequest,
			plan *iftaservice.ReturnAmendment,
		) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Create(toolpreview.Record{
				Resource: permission.ResourceIFTAReturn,
				Label: fmt.Sprintf("Amendment %d of the %s IFTA return",
					plan.Draft.AmendmentNumber, plan.Filed.Period().Label()),
			}, iftaReturnViewOf(plan.Draft))
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would open amendment %d of the filed %s IFTA return as a draft: %s. The "+
					"filed return is not changed.",
				plan.Draft.AmendmentNumber, plan.Filed.Period().Label(), iftaFigures(plan.Draft),
			), change), nil
		},
		Run: func(
			ctx context.Context,
			req *iftaservice.AmendReturnRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := returns.Amend(ctx, req)

			return nil, err
		},
	})
}

func NewDeleteIFTAReturnTool(returns IftaReturnKeeper) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(iftaInternalSpec(&toolspec.ReceivableSpec{
		Name: "fuel_delete_ifta_return",
		Description: "Delete a draft IFTA return, such as one generated for the wrong " +
			"quarter. Only a draft can be deleted; fuel_generate_ifta_return makes it again.",
		Resource:   permission.ResourceIFTAReturn,
		Operation:  permission.OpDelete,
		Reversible: true,
		Rationale: "Removes a draft return inside Trenova that nothing was filed from; " +
			"generating it again recomputes the same figures.",
		Properties: map[string]any{
			paramIFTAReturnID: agenttoolschema.KindID(iftaReturnSupplier+" It must be a Draft.",
				recordkind.IFTAReturn),
		},
		Required: []string{paramIFTAReturnID},
		Target:   targetIFTAReturn,
	}), toolspec.ReceivablePlan[pulid.ID, *ifta.Return]{
		Request: iftaReturnIDFrom,
		Plan: func(
			ctx context.Context,
			id pulid.ID,
			params *agenttoolports.ToolExecuteParams,
		) (*ifta.Return, error) {
			req, err := returnActionRequest(ctx, returns, params, id)
			if err != nil {
				return nil, err
			}

			return returns.PlanDelete(ctx, req)
		},
		Refused: func(pulid.ID) string {
			return "Would delete a draft IFTA return."
		},
		Render: func(_ pulid.ID, planned *ifta.Return) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Delete(iftaReturnRecord(planned), iftaReturnViewOf(planned))
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would delete the draft %s.", iftaReturnRecord(planned).Label,
			), change), nil
		},
		Run: func(
			ctx context.Context,
			id pulid.ID,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			req, err := returnActionRequest(ctx, returns, params, id)
			if err != nil {
				return nil, err
			}

			return nil, returns.Delete(ctx, req)
		},
	})
}

func iftaMileageProperties(forCorrection bool) map[string]any {
	keep := ""
	if forCorrection {
		keep = toolparams.KeepWhenLeftOut
	}

	properties := map[string]any{
		toolparams.ParamTractorID: agenttoolschema.RecordIDText(permission.ResourceTractor,
			"The tractor that drove the miles, from equipment_list_tractors."+keep),
		toolparams.ParamJurisdictionID: agenttoolschema.KindID(
			"The state or province the miles were "+
				"driven in, from fuel_list_ifta_jurisdictions."+keep,
			recordkind.IFTAJurisdiction,
		),
		paramTraveledAt: agenttoolschema.DateTime("When the miles were driven; it decides the " +
			"quarter they count in." + keep),
		paramMiles: toolparams.StringProperty("The miles as a decimal such as 212.5."+keep, 0),
		paramLoaded: toolparams.BooleanProperty(
			"Whether the tractor was loaded. Defaults to true." +
				keep,
		),
		toolparams.ParamShipmentMoveID: agenttoolschema.RecordIDText(
			permission.ResourceShipmentMove,
			"The move these miles correct, from shipment_get; the entry then replaces the "+
				"move's routed miles on the return."+keep,
		),
		toolparams.ParamNote: toolparams.StringProperty(
			"Where the miles came from, such as a trip sheet."+keep,
			maxMileageNotes,
		),
	}
	if forCorrection {
		properties[paramIFTAMileageEntryID] = agenttoolschema.KindID(iftaMileageSupplier,
			recordkind.IFTAMileageEntry)
	}

	return properties
}

func applyMileageParams(entry *ifta.JurisdictionMileageEntry, values map[string]any) error {
	for key, set := range map[string]func(pulid.ID){
		toolparams.ParamTractorID:      func(id pulid.ID) { entry.TractorID = id },
		toolparams.ParamJurisdictionID: func(id pulid.ID) { entry.JurisdictionID = id },
		toolparams.ParamShipmentMoveID: func(id pulid.ID) { entry.ShipmentMoveID = &id },
	} {
		if _, given := values[key]; !given {
			continue
		}
		id, err := toolparams.RequirePulid(values, key)
		if err != nil {
			return err
		}
		set(id)
	}

	if _, given := values[paramTraveledAt]; given {
		traveledAt, err := toolparams.RequireDateTime(values, paramTraveledAt)
		if err != nil {
			return err
		}
		entry.TraveledAt = traveledAt
	}
	if _, given := values[paramMiles]; given {
		miles, present, err := toolparams.OptionalDecimal(values, paramMiles)
		if err != nil {
			return err
		}
		if !present || !miles.IsPositive() {
			return fmt.Errorf("parameter %q must be greater than zero", paramMiles)
		}
		entry.Miles = miles
	}
	loaded, err := toolparams.OptionalBoolPointer(values, paramLoaded)
	if err != nil {
		return err
	}
	if loaded != nil {
		entry.Loaded = *loaded
	}
	notes, err := toolparams.OptionalBoundedText(values, toolparams.ParamNote, maxMileageNotes)
	if err != nil {
		return err
	}
	if notes != nil {
		entry.Notes = *notes
	}

	return nil
}

type iftaMileageView struct {
	TractorID      string `json:"tractorId"`
	Jurisdiction   string `json:"jurisdiction"`
	TraveledAt     int64  `json:"traveledAt"`
	Quarter        string `json:"quarter"`
	Miles          string `json:"miles"`
	Loaded         bool   `json:"loaded"`
	Source         string `json:"source"`
	ShipmentMoveID string `json:"shipmentMoveId,omitempty"`
	Notes          string `json:"notes,omitempty"`
}

func iftaMileageViewOf(entry *ifta.JurisdictionMileageEntry, jurisdiction string) *iftaMileageView {
	view := &iftaMileageView{
		TractorID:    entry.TractorID.String(),
		Jurisdiction: jurisdiction,
		TraveledAt:   entry.TraveledAt,
		Quarter:      entry.Period().Label(),
		Miles:        entry.Miles.StringFixed(2),
		Loaded:       entry.Loaded,
		Source:       string(entry.Source),
		Notes:        entry.Notes,
	}
	if entry.ShipmentMoveID != nil {
		view.ShipmentMoveID = entry.ShipmentMoveID.String()
	}

	return view
}

func iftaMileageOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.WithRefs(map[string]permission.Resource{
			toolparams.ParamTractorID: permission.ResourceTractor,
		}),
		toolpreview.Types(iftaMileageDates),
	}
}

func iftaMileageRecord(entry *ifta.JurisdictionMileageEntry) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceIFTAJurisdictionMileage,
		ID:       entry.ID,
		Label: fmt.Sprintf("%s miles in %s", entry.Miles.StringFixed(2),
			entry.Period().Label()),
		Version: changepreview.PinnedVersion(entry.Version),
	}
}

type labelledMileage struct {
	entry        *ifta.JurisdictionMileageEntry
	jurisdiction string
}

func NewRecordIFTAMileageEntryTool(entries IftaMileageKeeper) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(iftaInternalSpec(&toolspec.ReceivableSpec{
		Name: "fuel_record_ifta_mileage_entry",
		Description: "Record miles a tractor drove in one state or province that no routed " +
			"move carries, or that correct a move's routed miles. They come from a trip " +
			"sheet or a telematics report, and the quarter's IFTA return counts them.",
		Resource:   permission.ResourceIFTAJurisdictionMileage,
		Operation:  permission.OpCreate,
		Reversible: true,
		TaintHold:  iftaMileageTaintHold,
		Rationale: "Adds jurisdiction miles inside Trenova that the next IFTA return counts; " +
			"nothing is sent, and fuel_delete_ifta_mileage_entry removes them.",
		Properties: iftaMileageProperties(false),
		Required: []string{
			toolparams.ParamTractorID,
			toolparams.ParamJurisdictionID,
			paramTraveledAt,
			paramMiles,
		},
		SearchTerms: []string{
			toolparams.SearchTermIFTA,
			"miles",
			"trip sheet",
			"state miles",
			"jurisdiction",
		},
	}), toolspec.ReceivablePlan[*ifta.JurisdictionMileageEntry, *labelledMileage]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*ifta.JurisdictionMileageEntry, error) {
			entry := &ifta.JurisdictionMileageEntry{
				OrganizationID: params.OrganizationID,
				BusinessUnitID: params.BusinessUnitID,
				Loaded:         true,
				Source:         ifta.MileageSourceManual,
			}
			if err := applyMileageParams(entry, params.Params); err != nil {
				return nil, err
			}

			return entry, nil
		},
		Plan: func(
			ctx context.Context,
			entry *ifta.JurisdictionMileageEntry,
			params *agenttoolports.ToolExecuteParams,
		) (*labelledMileage, error) {
			planned, err := entries.PlanCreateMileageEntry(ctx, entry, params.Actor.UserID)
			if err != nil {
				return nil, err
			}

			return &labelledMileage{
				entry:        planned,
				jurisdiction: changepreview.JurisdictionLabel(ctx, entries, planned.JurisdictionID),
			}, nil
		},
		Refused: func(*ifta.JurisdictionMileageEntry) string {
			return "Would record jurisdiction miles."
		},
		Render: func(
			_ *ifta.JurisdictionMileageEntry,
			plan *labelledMileage,
		) (*agentpreview.ToolPreview, error) {
			record := iftaMileageRecord(plan.entry)
			record.ID = pulid.Nil
			record.Version = nil
			change, err := toolpreview.Create(
				record,
				iftaMileageViewOf(plan.entry, plan.jurisdiction),
				iftaMileageOptions()...,
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would record %s miles in %s, counted in %s.",
				plan.entry.Miles.StringFixed(2), plan.jurisdiction, plan.entry.Period().Label(),
			), change), nil
		},
		Run: func(
			ctx context.Context,
			entry *ifta.JurisdictionMileageEntry,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := entries.CreateMileageEntry(ctx, entry, params.Actor.UserID)

			return nil, err
		},
	})
}

func targetIFTAMileage(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(
		params,
		paramIFTAMileageEntryID,
		permission.ResourceIFTAJurisdictionMileage,
	)
}

type mileageCorrection struct {
	id     pulid.ID
	values map[string]any
}

func (c *mileageCorrection) entry(
	ctx context.Context,
	entries IftaMileageKeeper,
	params *agenttoolports.ToolExecuteParams,
) (*ifta.JurisdictionMileageEntry, error) {
	stored, err := entries.GetMileageEntry(ctx, toolparams.TenantFrom(params), c.id)
	if err != nil {
		return nil, err
	}

	corrected := *stored
	corrected.Tractor = nil
	corrected.Jurisdiction = nil
	corrected.CreatedBy = nil
	if err = applyMileageParams(&corrected, c.values); err != nil {
		return nil, err
	}

	return &corrected, nil
}

type labelledMileageChange struct {
	change *iftaservice.MileageEntryChange
	before string
	after  string
}

func NewCorrectIFTAMileageEntryTool(entries IftaMileageKeeper) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(iftaInternalSpec(&toolspec.ReceivableSpec{
		Name:        "fuel_correct_ifta_mileage_entry",
		SearchTerms: []string{"fix mileage entry", "wrong jurisdiction miles", "correct miles"},
		Description: "Correct a jurisdiction mileage entry that was keyed wrong: its tractor, " +
			"state or province, day, miles or notes. Fields left out keep their value. A " +
			"finalized return keeps its figures until it is reopened or amended.",
		Resource:   permission.ResourceIFTAJurisdictionMileage,
		Operation:  permission.OpUpdate,
		Reversible: true,
		TaintHold:  iftaMileageTaintHold,
		Rationale: "Changes jurisdiction miles inside Trenova; nothing is sent, and a later " +
			"correction changes them back.",
		Properties: iftaMileageProperties(true),
		Required:   []string{paramIFTAMileageEntryID},
		Target:     targetIFTAMileage,
	}), toolspec.ReceivablePlan[*mileageCorrection, *labelledMileageChange]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*mileageCorrection, error) {
			id, err := toolparams.RequirePulid(params.Params, paramIFTAMileageEntryID)
			if err != nil {
				return nil, err
			}
			values := make(map[string]any, len(params.Params))
			for key, value := range params.Params {
				if key != paramIFTAMileageEntryID {
					values[key] = value
				}
			}
			if len(values) == 0 {
				return nil, errors.New("name at least one field of the entry to correct")
			}

			return &mileageCorrection{id: id, values: values}, nil
		},
		Plan: func(
			ctx context.Context,
			correction *mileageCorrection,
			params *agenttoolports.ToolExecuteParams,
		) (*labelledMileageChange, error) {
			entry, err := correction.entry(ctx, entries, params)
			if err != nil {
				return nil, err
			}
			change, err := entries.PlanUpdateMileageEntry(ctx, entry, params.Actor.UserID)
			if err != nil {
				return nil, err
			}

			return &labelledMileageChange{
				change: change,
				before: changepreview.JurisdictionLabel(ctx, entries, change.Before.JurisdictionID),
				after:  changepreview.JurisdictionLabel(ctx, entries, change.After.JurisdictionID),
			}, nil
		},
		Refused: func(*mileageCorrection) string {
			return "Would correct a jurisdiction mileage entry."
		},
		Render: func(
			_ *mileageCorrection,
			plan *labelledMileageChange,
		) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Changed(
				iftaMileageRecord(plan.change.Before),
				iftaMileageViewOf(plan.change.Before, plan.before),
				iftaMileageViewOf(plan.change.After, plan.after),
				iftaMileageOptions()...,
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would correct the %s; a finalized return keeps its figures until it is "+
					"reopened or amended.",
				iftaMileageRecord(plan.change.Before).Label,
			), change), nil
		},
		Run: func(
			ctx context.Context,
			correction *mileageCorrection,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			entry, err := correction.entry(ctx, entries, params)
			if err != nil {
				return nil, err
			}
			_, err = entries.UpdateMileageEntry(ctx, entry, params.Actor.UserID)

			return nil, err
		},
	})
}

func deleteMileageRequest(
	ctx context.Context,
	entries IftaMileageKeeper,
	params *agenttoolports.ToolExecuteParams,
	id pulid.ID,
) (*iftaservice.DeleteMileageEntryRequest, error) {
	stored, err := entries.GetMileageEntry(ctx, toolparams.TenantFrom(params), id)
	if err != nil {
		return nil, err
	}

	return &iftaservice.DeleteMileageEntryRequest{
		TenantInfo: toolparams.TenantFrom(params),
		ID:         stored.ID,
		Version:    stored.Version,
		UserID:     params.Actor.UserID,
	}, nil
}

func NewDeleteIFTAMileageEntryTool(entries IftaMileageKeeper) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(iftaInternalSpec(&toolspec.ReceivableSpec{
		Name:        "fuel_delete_ifta_mileage_entry",
		SearchTerms: []string{"remove mileage entry", "duplicate mileage entry"},
		Description: "Propose deleting a jurisdiction mileage entry recorded in error, such " +
			"as a trip sheet entered twice. The next IFTA return no longer counts its miles. " +
			"To fix a wrong field use fuel_correct_ifta_mileage_entry instead.",
		Resource:  permission.ResourceIFTAJurisdictionMileage,
		Operation: permission.OpDelete,
		MaxTier:   agent.TierPropose,
		Rationale: "Removes miles the IFTA return is computed from, and nothing brings them " +
			"back but entering them again, so a person always decides.",
		Properties: map[string]any{
			paramIFTAMileageEntryID: agenttoolschema.KindID(iftaMileageSupplier,
				recordkind.IFTAMileageEntry),
		},
		Required: []string{paramIFTAMileageEntryID},
		Target:   targetIFTAMileage,
	}), toolspec.ReceivablePlan[pulid.ID, *labelledMileage]{
		Request: func(params *agenttoolports.ToolExecuteParams) (pulid.ID, error) {
			return toolparams.RequirePulid(params.Params, paramIFTAMileageEntryID)
		},
		Plan: func(
			ctx context.Context,
			id pulid.ID,
			params *agenttoolports.ToolExecuteParams,
		) (*labelledMileage, error) {
			req, err := deleteMileageRequest(ctx, entries, params, id)
			if err != nil {
				return nil, err
			}
			planned, err := entries.PlanDeleteMileageEntry(ctx, req)
			if err != nil {
				return nil, err
			}

			return &labelledMileage{
				entry:        planned,
				jurisdiction: changepreview.JurisdictionLabel(ctx, entries, planned.JurisdictionID),
			}, nil
		},
		Refused: func(pulid.ID) string {
			return "Would delete a jurisdiction mileage entry."
		},
		Render: func(_ pulid.ID, plan *labelledMileage) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Delete(
				iftaMileageRecord(plan.entry),
				iftaMileageViewOf(plan.entry, plan.jurisdiction),
				iftaMileageOptions()...,
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would delete the %s driven in %s.",
				iftaMileageRecord(plan.entry).Label, plan.jurisdiction,
			), change), nil
		},
		Run: func(
			ctx context.Context,
			id pulid.ID,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			req, err := deleteMileageRequest(ctx, entries, params, id)
			if err != nil {
				return nil, err
			}

			return nil, entries.DeleteMileageEntry(ctx, req)
		},
	})
}

func moveJurisdictionRequest(
	params *agenttoolports.ToolExecuteParams,
) (shipmentdistanceports.RecalculateMoveJurisdictionMilesRequest, error) {
	moveID, err := toolparams.RequirePulid(params.Params, toolparams.ParamShipmentMoveID)
	if err != nil {
		return shipmentdistanceports.RecalculateMoveJurisdictionMilesRequest{}, err
	}

	return shipmentdistanceports.RecalculateMoveJurisdictionMilesRequest{
		TenantInfo:     toolparams.TenantFrom(params),
		ShipmentMoveID: moveID,
		UserID:         params.Actor.UserID,
	}, nil
}

func NewRecalculateMoveJurisdictionMilesTool(
	router JurisdictionMileRouter,
) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(iftaInternalSpec(&toolspec.ReceivableSpec{
		Name:        "fuel_recalculate_jurisdiction_miles",
		SearchTerms: []string{"move jurisdiction miles"},
		Description: "Ask the routing provider again for one move's state-by-state miles and " +
			"replace its jurisdiction rows. Use it when the move's stops changed or the IFTA " +
			"return shows it unattributed. The move's distance is left as it was. Each run is one " +
			"billable distance request.",
		Resource:   permission.ResourceShipmentMove,
		Operation:  permission.OpUpdate,
		Reversible: true,
		Rationale: "Re-derives one move's jurisdiction miles inside Trenova from its stops; " +
			"nothing is sent to a customer or carrier, and running it again gives the same " +
			"miles.",
		Properties: map[string]any{
			toolparams.ParamShipmentMoveID: agenttoolschema.RecordIDText(
				permission.ResourceShipmentMove,
				"The move, from shipment_get. Never guess one.",
			),
		},
		Required: []string{toolparams.ParamShipmentMoveID},
		Target: func(params map[string]any) (agenttoolports.ToolTarget, bool) {
			return toolparams.TargetOf(
				params,
				toolparams.ParamShipmentMoveID,
				permission.ResourceShipmentMove,
			)
		},
	}), toolspec.ReceivablePlan[shipmentdistanceports.RecalculateMoveJurisdictionMilesRequest, *shipmentdistanceports.MoveJurisdictionMilesPlan]{
		Request: moveJurisdictionRequest,
		Plan: func(
			ctx context.Context,
			req shipmentdistanceports.RecalculateMoveJurisdictionMilesRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*shipmentdistanceports.MoveJurisdictionMilesPlan, error) {
			return router.PlanMoveJurisdictionMiles(ctx, &req)
		},
		Refused: func(shipmentdistanceports.RecalculateMoveJurisdictionMilesRequest) string {
			return "Would ask for a move's jurisdiction miles again."
		},
		Render: func(
			_ shipmentdistanceports.RecalculateMoveJurisdictionMilesRequest,
			plan *shipmentdistanceports.MoveJurisdictionMilesPlan,
		) (*agentpreview.ToolPreview, error) {
			return toolpreview.Build(fmt.Sprintf(
				"Would ask the routing provider for the state-by-state miles of the move "+
					"across %s and replace its %s; its distance stays as it is.",
				changepreview.CountOf(len(plan.Move.Stops), "stop"),
				changepreview.CountOf(len(plan.Move.JurisdictionMiles), "jurisdiction row"),
			)), nil
		},
		Run: func(
			ctx context.Context,
			req shipmentdistanceports.RecalculateMoveJurisdictionMilesRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := router.RecalculateMoveJurisdictionMiles(ctx, req)

			return nil, err
		},
	})
}

type backfillRequest struct {
	period ifta.Period
	moves  int
}

func (b *backfillRequest) service(
	ctx context.Context,
	returns IftaReturnKeeper,
	params *agenttoolports.ToolExecuteParams,
	dryRun bool,
) (shipmentdistanceports.BackfillJurisdictionMilesRequest, error) {
	info, err := returns.PeriodInfo(
		ctx,
		toolparams.TenantFrom(params),
		b.period.Year,
		b.period.Quarter,
	)
	if err != nil {
		return shipmentdistanceports.BackfillJurisdictionMilesRequest{}, err
	}

	return shipmentdistanceports.BackfillJurisdictionMilesRequest{
		TenantInfo: toolparams.TenantFrom(params),
		Start:      info.Start,
		End:        info.End,
		MaxMoves:   b.moves,
		DryRun:     dryRun,
		UserID:     params.Actor.UserID,
	}, nil
}

func NewBackfillJurisdictionMilesTool(
	returns IftaReturnKeeper,
	router JurisdictionMileRouter,
) agenttoolports.AgentTool {
	properties := iftaPeriodProperties()
	properties[paramMaxMoves] = toolparams.IntegerProperty("The most moves to route in this run. "+
		"Defaults to 2000; each is a billable distance request.", 1, maxBackfillMoves)

	return toolspec.NewReceivableTool(iftaInternalSpec(&toolspec.ReceivableSpec{
		Name: "fuel_backfill_jurisdiction_miles",
		Description: "Start a background job that routes every completed move in a quarter " +
			"that has no state-by-state miles. The IFTA return then stops showing them " +
			"unattributed. The preview counts the moves and " +
			"miles first; each move is a billable distance request.",
		Resource:  permission.ResourceIFTAReturn,
		Operation: permission.OpManage,
		Rationale: "Routes a quarter's unattributed moves inside Trenova in the background; " +
			"nothing is sent to a customer or carrier, but each move is a billable request.",
		Properties: properties,
		Required:   []string{paramYear, paramQuarter},
	}), toolspec.ReceivablePlan[*backfillRequest, *shipmentdistanceports.BackfillJurisdictionMilesResult]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*backfillRequest, error) {
			period, err := iftaPeriodFrom(params)
			if err != nil {
				return nil, err
			}
			moves := 0
			if _, given := params.Params[paramMaxMoves]; given {
				if moves, err = toolparams.RequireIntInRange(
					params.Params,
					paramMaxMoves,
					1,
					maxBackfillMoves,
				); err != nil {
					return nil, err
				}
			}

			return &backfillRequest{period: period, moves: moves}, nil
		},
		Plan: func(
			ctx context.Context,
			req *backfillRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*shipmentdistanceports.BackfillJurisdictionMilesResult, error) {
			dry, err := req.service(ctx, returns, params, true)
			if err != nil {
				return nil, err
			}

			return router.BackfillJurisdictionMiles(ctx, &dry)
		},
		Refused: func(req *backfillRequest) string {
			return fmt.Sprintf("Would backfill jurisdiction miles for %s.", req.period.Label())
		},
		Render: func(
			req *backfillRequest,
			result *shipmentdistanceports.BackfillJurisdictionMilesResult,
		) (*agentpreview.ToolPreview, error) {
			if result.UnattributedMoves == 0 {
				return toolpreview.Build(fmt.Sprintf(
					"Every completed move in %s already has jurisdiction miles; nothing would "+
						"be routed.", req.period.Label(),
				)), nil
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would start routing %s in %s with no jurisdiction miles, %s miles in all; "+
					"each is a billable distance request.",
				changepreview.CountOf(result.UnattributedMoves, "completed move"),
				req.period.Label(),
				result.UnattributedMiles.StringFixed(1),
			)), nil
		},
		Run: func(
			ctx context.Context,
			req *backfillRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			started, err := req.service(ctx, returns, params, false)
			if err != nil {
				return nil, err
			}
			_, err = router.BackfillJurisdictionMiles(ctx, &started)

			return nil, err
		},
	})
}
