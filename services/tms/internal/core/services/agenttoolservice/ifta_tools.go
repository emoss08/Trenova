package agenttoolservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/iftaservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramIFTAReturnID       = "iftaReturnId"
	paramIFTAMileageEntryID = "iftaMileageEntryId"
	paramYear               = "year"
	paramQuarter            = "quarter"
	paramTraveledAt         = "traveledAt"
	paramMiles              = "miles"
	paramLoaded             = "loaded"
	paramShipmentMoveID     = "shipmentMoveId"
	paramMaxMoves           = "maxMoves"

	minIFTAYear           = 2000
	maxIFTAYear           = 2100
	quartersInYear        = 4
	maxAmendReason        = 1000
	maxMileageNotes       = 2000
	maxBackfillMoves      = 5000
	iftaReturnSupplier    = "The IFTA return, from list_ifta_returns. Never guess one."
	iftaMileageSupplier   = "The mileage entry, from list_ifta_mileage_entries. Never guess one."
	iftaMileageTaintHold  = "Miles read from a log or a message someone outside sent are proposed, since the IFTA return is computed from them."
	iftaReturnRecordKind  = "IFTA return"
	iftaMileageRecordKind = "mileage entry"
)

var iftaMileageDates = map[string]assistantartifact.DisplayType{
	paramTraveledAt: assistantartifact.DisplayDateTime,
}

type iftaReturnKeeper interface {
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

type iftaMileageKeeper interface {
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

type jurisdictionMileRouter interface {
	PlanMoveJurisdictionMiles(
		ctx context.Context,
		req *serviceports.RecalculateMoveJurisdictionMilesRequest,
	) (*serviceports.MoveJurisdictionMilesPlan, error)
	RecalculateMoveJurisdictionMiles(
		ctx context.Context,
		req serviceports.RecalculateMoveJurisdictionMilesRequest,
	) ([]*shipment.ShipmentMoveJurisdictionMile, error)
	BackfillJurisdictionMiles(
		ctx context.Context,
		req serviceports.BackfillJurisdictionMilesRequest,
	) (*serviceports.BackfillJurisdictionMilesResult, error)
}

var (
	_ iftaReturnKeeper       = (*iftaservice.Service)(nil)
	_ iftaMileageKeeper      = (*iftaservice.Service)(nil)
	_ jurisdictionMileRouter = serviceports.DistanceCalculationService(nil)
)

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
		Version:  pinnedVersion(ret.Version),
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
		figures += fmt.Sprintf("; %s block finalizing it", countOf(blocking, "problem"))
	}

	return figures
}

func iftaInternalSpec(spec *receivableSpec) *receivableSpec {
	return fuelInternalSpec(spec)
}

func targetIFTAReturn(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramIFTAReturnID, permission.ResourceIFTAReturn)
}

func iftaPeriodFrom(params *serviceports.ToolExecuteParams) (ifta.Period, error) {
	year, err := requireIntInRange(params.Params, paramYear, minIFTAYear, maxIFTAYear)
	if err != nil {
		return ifta.Period{}, err
	}
	quarter, err := requireIntInRange(params.Params, paramQuarter, 1, quartersInYear)
	if err != nil {
		return ifta.Period{}, err
	}

	return ifta.NewPeriod(year, quarter), nil
}

func iftaPeriodProperties() map[string]any {
	return map[string]any{
		paramYear:    integerProperty("The year of the quarter.", minIFTAYear, maxIFTAYear),
		paramQuarter: integerProperty("The quarter, 1 to 4.", 1, quartersInYear),
	}
}

func newGenerateIFTAReturnTool(returns iftaReturnKeeper) serviceports.AgentTool {
	return newReceivableTool(iftaInternalSpec(&receivableSpec{
		name: "generate_ifta_return",
		description: "Generate a quarter's IFTA fuel tax return as a draft, computed from " +
			"the jurisdiction miles and fuel purchases on file. It files nothing: a person " +
			"finalizes and files it. Refused while a draft or finalized return exists for the " +
			"quarter, and once one is filed; use recompute_ifta_return or amend_ifta_return.",
		resource:   permission.ResourceIFTAReturn,
		operation:  permission.OpCreate,
		reversible: true,
		rationale: "Computes a draft return inside Trenova; nothing is filed or sent, and " +
			"delete_ifta_return removes the draft.",
		properties:  iftaPeriodProperties(),
		required:    []string{paramYear, paramQuarter},
		searchTerms: []string{searchTermIFTA, "fuel tax", "quarterly return", "quarter"},
	}), receivablePlan[*iftaservice.GenerateReturnRequest, *ifta.Return]{
		request: func(params *serviceports.ToolExecuteParams) (*iftaservice.GenerateReturnRequest, error) {
			period, err := iftaPeriodFrom(params)
			if err != nil {
				return nil, err
			}

			return &iftaservice.GenerateReturnRequest{
				TenantInfo: tenantFrom(*params),
				Period:     period,
				UserID:     params.Actor.UserID,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *iftaservice.GenerateReturnRequest,
			_ *serviceports.ToolExecuteParams,
		) (*ifta.Return, error) {
			return returns.PlanGenerate(ctx, req)
		},
		refused: func(req *iftaservice.GenerateReturnRequest) string {
			return fmt.Sprintf("Would generate the %s IFTA return.", req.Period.Label())
		},
		render: func(
			req *iftaservice.GenerateReturnRequest,
			planned *ifta.Return,
		) (*agent.ToolPreview, error) {
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
		run: func(
			ctx context.Context,
			req *iftaservice.GenerateReturnRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := returns.Generate(ctx, req)

			return nil, err
		},
	})
}

func returnActionRequest(
	ctx context.Context,
	returns iftaReturnKeeper,
	params *serviceports.ToolExecuteParams,
	id pulid.ID,
) (*iftaservice.ReturnActionRequest, error) {
	stored, err := returns.Get(ctx, tenantFrom(*params), id)
	if err != nil {
		return nil, err
	}

	return &iftaservice.ReturnActionRequest{
		TenantInfo: tenantFrom(*params),
		ID:         stored.ID,
		Version:    stored.Version,
		UserID:     params.Actor.UserID,
	}, nil
}

func iftaReturnIDFrom(params *serviceports.ToolExecuteParams) (pulid.ID, error) {
	return requirePulid(params.Params, paramIFTAReturnID)
}

func newRecomputeIFTAReturnTool(returns iftaReturnKeeper) serviceports.AgentTool {
	return newReceivableTool(iftaInternalSpec(&receivableSpec{
		name: "recompute_ifta_return",
		description: "Recompute a draft IFTA return from the miles, purchases and tax rates " +
			"on file now, after purchases or mileage were corrected. Only a draft can be " +
			"recomputed; a finalized return must be reopened by a person first.",
		resource:    permission.ResourceIFTAReturn,
		operation:   permission.OpUpdate,
		defaultTier: agent.TierActWithApproval,
		maxTier:     agent.TierAutoExecute,
		reversible:  true,
		rationale: "Re-derives a draft return's figures inside Trenova from what is on file; " +
			"nothing is filed, and running it again gives the same figures.",
		properties: map[string]any{
			paramIFTAReturnID: stringProperty(iftaReturnSupplier, 0),
		},
		required: []string{paramIFTAReturnID},
		target:   targetIFTAReturn,
	}), receivablePlan[pulid.ID, *iftaservice.ReturnChange]{
		request: iftaReturnIDFrom,
		plan: func(
			ctx context.Context,
			id pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*iftaservice.ReturnChange, error) {
			req, err := returnActionRequest(ctx, returns, params, id)
			if err != nil {
				return nil, err
			}

			return returns.PlanRecompute(ctx, req)
		},
		refused: func(pulid.ID) string {
			return "Would recompute an IFTA return."
		},
		render: func(_ pulid.ID, plan *iftaservice.ReturnChange) (*agent.ToolPreview, error) {
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
		run: func(
			ctx context.Context,
			id pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			req, err := returnActionRequest(ctx, returns, params, id)
			if err != nil {
				return nil, err
			}
			_, err = returns.Recompute(ctx, req)

			return nil, err
		},
	})
}

func newAmendIFTAReturnTool(returns iftaReturnKeeper) serviceports.AgentTool {
	return newReceivableTool(iftaInternalSpec(&receivableSpec{
		name: "amend_ifta_return",
		description: "Propose opening an amendment of a filed IFTA return, with the reason: " +
			"a new draft for the same quarter, computed from what is on file now. The filed " +
			"return is left as it was, and a person finalizes and files the amendment.",
		resource:  permission.ResourceIFTAReturn,
		operation: permission.OpCreate,
		maxTier:   agent.TierPropose,
		rationale: "Starts a correction of a return already filed with the jurisdictions; " +
			"it files nothing, but a person decides to reopen a filing.",
		properties: map[string]any{
			paramIFTAReturnID: stringProperty(iftaReturnSupplier+" It must be Filed.", 0),
			paramReason: stringProperty("What was wrong with the filed return, at least ten "+
				"characters, as the auditor will read it.", maxAmendReason),
		},
		required: []string{paramIFTAReturnID, paramReason},
		target:   targetIFTAReturn,
	}), receivablePlan[*iftaservice.AmendReturnRequest, *iftaservice.ReturnAmendment]{
		request: func(params *serviceports.ToolExecuteParams) (*iftaservice.AmendReturnRequest, error) {
			id, err := requirePulid(params.Params, paramIFTAReturnID)
			if err != nil {
				return nil, err
			}
			reason, err := requireBoundedText(params.Params, paramReason, maxAmendReason)
			if err != nil {
				return nil, err
			}

			return &iftaservice.AmendReturnRequest{
				TenantInfo: tenantFrom(*params),
				ID:         id,
				Reason:     reason,
				UserID:     params.Actor.UserID,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *iftaservice.AmendReturnRequest,
			_ *serviceports.ToolExecuteParams,
		) (*iftaservice.ReturnAmendment, error) {
			return returns.PlanAmend(ctx, req)
		},
		refused: func(*iftaservice.AmendReturnRequest) string {
			return "Would open an amendment of an IFTA return."
		},
		render: func(
			_ *iftaservice.AmendReturnRequest,
			plan *iftaservice.ReturnAmendment,
		) (*agent.ToolPreview, error) {
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
		run: func(
			ctx context.Context,
			req *iftaservice.AmendReturnRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := returns.Amend(ctx, req)

			return nil, err
		},
	})
}

func newDeleteIFTAReturnTool(returns iftaReturnKeeper) serviceports.AgentTool {
	return newReceivableTool(iftaInternalSpec(&receivableSpec{
		name: "delete_ifta_return",
		description: "Delete a draft IFTA return, such as one generated for the wrong " +
			"quarter. Only a draft can be deleted; generate_ifta_return makes it again.",
		resource:   permission.ResourceIFTAReturn,
		operation:  permission.OpDelete,
		reversible: true,
		rationale: "Removes a draft return inside Trenova that nothing was filed from; " +
			"generating it again recomputes the same figures.",
		properties: map[string]any{
			paramIFTAReturnID: stringProperty(iftaReturnSupplier+" It must be a Draft.", 0),
		},
		required: []string{paramIFTAReturnID},
		target:   targetIFTAReturn,
	}), receivablePlan[pulid.ID, *ifta.Return]{
		request: iftaReturnIDFrom,
		plan: func(
			ctx context.Context,
			id pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*ifta.Return, error) {
			req, err := returnActionRequest(ctx, returns, params, id)
			if err != nil {
				return nil, err
			}

			return returns.PlanDelete(ctx, req)
		},
		refused: func(pulid.ID) string {
			return "Would delete a draft IFTA return."
		},
		render: func(_ pulid.ID, planned *ifta.Return) (*agent.ToolPreview, error) {
			change, err := toolpreview.Delete(iftaReturnRecord(planned), iftaReturnViewOf(planned))
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would delete the draft %s.", iftaReturnRecord(planned).Label,
			), change), nil
		},
		run: func(
			ctx context.Context,
			id pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
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
		keep = keepWhenLeftOut
	}

	properties := map[string]any{
		paramTractorID: stringProperty("The tractor that drove the miles, from "+
			"list_tractors."+keep, 0),
		paramJurisdictionID: stringProperty("The state or province the miles were driven in, "+
			"from list_ifta_jurisdictions."+keep, 0),
		paramTraveledAt: dateTimeProperty("When the miles were driven; it decides the " +
			"quarter they count in." + keep),
		paramMiles: stringProperty("The miles as a decimal such as 212.5."+keep, 0),
		paramLoaded: booleanProperty("Whether the tractor was loaded. Defaults to true." +
			keep),
		paramShipmentMoveID: stringProperty("The move these miles correct, from "+
			"get_shipment; the entry then replaces the move's routed miles on the return."+
			keep, 0),
		paramNotes: stringProperty("Where the miles came from, such as a trip sheet."+keep,
			maxMileageNotes),
	}
	if forCorrection {
		properties[paramIFTAMileageEntryID] = stringProperty(iftaMileageSupplier, 0)
	}

	return properties
}

func applyMileageParams(entry *ifta.JurisdictionMileageEntry, values map[string]any) error {
	for key, set := range map[string]func(pulid.ID){
		paramTractorID:      func(id pulid.ID) { entry.TractorID = id },
		paramJurisdictionID: func(id pulid.ID) { entry.JurisdictionID = id },
		paramShipmentMoveID: func(id pulid.ID) { entry.ShipmentMoveID = &id },
	} {
		if _, given := values[key]; !given {
			continue
		}
		id, err := requirePulid(values, key)
		if err != nil {
			return err
		}
		set(id)
	}

	if _, given := values[paramTraveledAt]; given {
		traveledAt, err := requireDateTime(values, paramTraveledAt)
		if err != nil {
			return err
		}
		entry.TraveledAt = traveledAt
	}
	if _, given := values[paramMiles]; given {
		miles, present, err := optionalDecimal(values, paramMiles)
		if err != nil {
			return err
		}
		if !present || !miles.IsPositive() {
			return fmt.Errorf("parameter %q must be greater than zero", paramMiles)
		}
		entry.Miles = miles
	}
	loaded, err := optionalBoolPointer(values, paramLoaded)
	if err != nil {
		return err
	}
	if loaded != nil {
		entry.Loaded = *loaded
	}
	notes, err := optionalBoundedText(values, paramNotes, maxMileageNotes)
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
			paramTractorID: permission.ResourceTractor,
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
		Version: pinnedVersion(entry.Version),
	}
}

type labelledMileage struct {
	entry        *ifta.JurisdictionMileageEntry
	jurisdiction string
}

func newRecordIFTAMileageEntryTool(entries iftaMileageKeeper) serviceports.AgentTool {
	return newReceivableTool(iftaInternalSpec(&receivableSpec{
		name: "record_ifta_mileage_entry",
		description: "Record miles a tractor drove in one state or province that no routed " +
			"move carries, or that correct a move's routed miles. They come from a trip " +
			"sheet or a telematics report, and the quarter's IFTA return counts them.",
		resource:   permission.ResourceIFTAJurisdictionMileage,
		operation:  permission.OpCreate,
		reversible: true,
		taintHold:  iftaMileageTaintHold,
		rationale: "Adds jurisdiction miles inside Trenova that the next IFTA return counts; " +
			"nothing is sent, and delete_ifta_mileage_entry removes them.",
		properties:  iftaMileageProperties(false),
		required:    []string{paramTractorID, paramJurisdictionID, paramTraveledAt, paramMiles},
		searchTerms: []string{searchTermIFTA, "miles", "trip sheet", "state miles", "jurisdiction"},
	}), receivablePlan[*ifta.JurisdictionMileageEntry, *labelledMileage]{
		request: func(params *serviceports.ToolExecuteParams) (*ifta.JurisdictionMileageEntry, error) {
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
		plan: func(
			ctx context.Context,
			entry *ifta.JurisdictionMileageEntry,
			params *serviceports.ToolExecuteParams,
		) (*labelledMileage, error) {
			planned, err := entries.PlanCreateMileageEntry(ctx, entry, params.Actor.UserID)
			if err != nil {
				return nil, err
			}

			return &labelledMileage{
				entry:        planned,
				jurisdiction: jurisdictionLabel(ctx, entries, planned.JurisdictionID),
			}, nil
		},
		refused: func(*ifta.JurisdictionMileageEntry) string {
			return "Would record jurisdiction miles."
		},
		render: func(
			_ *ifta.JurisdictionMileageEntry,
			plan *labelledMileage,
		) (*agent.ToolPreview, error) {
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
		run: func(
			ctx context.Context,
			entry *ifta.JurisdictionMileageEntry,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := entries.CreateMileageEntry(ctx, entry, params.Actor.UserID)

			return nil, err
		},
	})
}

func targetIFTAMileage(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramIFTAMileageEntryID, permission.ResourceIFTAJurisdictionMileage)
}

type mileageCorrection struct {
	id     pulid.ID
	values map[string]any
}

func (c *mileageCorrection) entry(
	ctx context.Context,
	entries iftaMileageKeeper,
	params *serviceports.ToolExecuteParams,
) (*ifta.JurisdictionMileageEntry, error) {
	stored, err := entries.GetMileageEntry(ctx, tenantFrom(*params), c.id)
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

func newCorrectIFTAMileageEntryTool(entries iftaMileageKeeper) serviceports.AgentTool {
	return newReceivableTool(iftaInternalSpec(&receivableSpec{
		name: "correct_ifta_mileage_entry",
		description: "Correct a jurisdiction mileage entry that was keyed wrong: its tractor, " +
			"state or province, day, miles or notes. Fields left out keep their value. A " +
			"finalized return keeps its figures until it is reopened or amended.",
		resource:   permission.ResourceIFTAJurisdictionMileage,
		operation:  permission.OpUpdate,
		reversible: true,
		taintHold:  iftaMileageTaintHold,
		rationale: "Changes jurisdiction miles inside Trenova; nothing is sent, and a later " +
			"correction changes them back.",
		properties: iftaMileageProperties(true),
		required:   []string{paramIFTAMileageEntryID},
		target:     targetIFTAMileage,
	}), receivablePlan[*mileageCorrection, *labelledMileageChange]{
		request: func(params *serviceports.ToolExecuteParams) (*mileageCorrection, error) {
			id, err := requirePulid(params.Params, paramIFTAMileageEntryID)
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
		plan: func(
			ctx context.Context,
			correction *mileageCorrection,
			params *serviceports.ToolExecuteParams,
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
				before: jurisdictionLabel(ctx, entries, change.Before.JurisdictionID),
				after:  jurisdictionLabel(ctx, entries, change.After.JurisdictionID),
			}, nil
		},
		refused: func(*mileageCorrection) string {
			return "Would correct a jurisdiction mileage entry."
		},
		render: func(
			_ *mileageCorrection,
			plan *labelledMileageChange,
		) (*agent.ToolPreview, error) {
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
		run: func(
			ctx context.Context,
			correction *mileageCorrection,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
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
	entries iftaMileageKeeper,
	params *serviceports.ToolExecuteParams,
	id pulid.ID,
) (*iftaservice.DeleteMileageEntryRequest, error) {
	stored, err := entries.GetMileageEntry(ctx, tenantFrom(*params), id)
	if err != nil {
		return nil, err
	}

	return &iftaservice.DeleteMileageEntryRequest{
		TenantInfo: tenantFrom(*params),
		ID:         stored.ID,
		Version:    stored.Version,
		UserID:     params.Actor.UserID,
	}, nil
}

func newDeleteIFTAMileageEntryTool(entries iftaMileageKeeper) serviceports.AgentTool {
	return newReceivableTool(iftaInternalSpec(&receivableSpec{
		name: "delete_ifta_mileage_entry",
		description: "Propose deleting a jurisdiction mileage entry recorded in error, such " +
			"as a trip sheet entered twice. The next IFTA return no longer counts its miles. " +
			"To fix a wrong field use correct_ifta_mileage_entry instead.",
		resource:  permission.ResourceIFTAJurisdictionMileage,
		operation: permission.OpDelete,
		maxTier:   agent.TierPropose,
		rationale: "Removes miles the IFTA return is computed from, and nothing brings them " +
			"back but entering them again, so a person always decides.",
		properties: map[string]any{
			paramIFTAMileageEntryID: stringProperty(iftaMileageSupplier, 0),
		},
		required: []string{paramIFTAMileageEntryID},
		target:   targetIFTAMileage,
	}), receivablePlan[pulid.ID, *labelledMileage]{
		request: func(params *serviceports.ToolExecuteParams) (pulid.ID, error) {
			return requirePulid(params.Params, paramIFTAMileageEntryID)
		},
		plan: func(
			ctx context.Context,
			id pulid.ID,
			params *serviceports.ToolExecuteParams,
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
				jurisdiction: jurisdictionLabel(ctx, entries, planned.JurisdictionID),
			}, nil
		},
		refused: func(pulid.ID) string {
			return "Would delete a jurisdiction mileage entry."
		},
		render: func(_ pulid.ID, plan *labelledMileage) (*agent.ToolPreview, error) {
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
		run: func(
			ctx context.Context,
			id pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			req, err := deleteMileageRequest(ctx, entries, params, id)
			if err != nil {
				return nil, err
			}

			return nil, entries.DeleteMileageEntry(ctx, req)
		},
	})
}

func moveJurisdictionRequest(
	params *serviceports.ToolExecuteParams,
) (serviceports.RecalculateMoveJurisdictionMilesRequest, error) {
	moveID, err := requirePulid(params.Params, paramShipmentMoveID)
	if err != nil {
		return serviceports.RecalculateMoveJurisdictionMilesRequest{}, err
	}

	return serviceports.RecalculateMoveJurisdictionMilesRequest{
		TenantInfo:     tenantFrom(*params),
		ShipmentMoveID: moveID,
		UserID:         params.Actor.UserID,
	}, nil
}

func newRecalculateMoveJurisdictionMilesTool(router jurisdictionMileRouter) serviceports.AgentTool {
	return newReceivableTool(iftaInternalSpec(&receivableSpec{
		name: "recalculate_move_jurisdiction_miles",
		description: "Ask the routing provider again for one move's state-by-state miles and " +
			"replace its jurisdiction rows. Use it when the move's stops changed or the IFTA " +
			"return shows it unattributed. The move's distance is left as it was. Each run is one " +
			"billable distance request.",
		resource:   permission.ResourceShipmentMove,
		operation:  permission.OpUpdate,
		reversible: true,
		rationale: "Re-derives one move's jurisdiction miles inside Trenova from its stops; " +
			"nothing is sent to a customer or carrier, and running it again gives the same " +
			"miles.",
		properties: map[string]any{
			paramShipmentMoveID: stringProperty("The move, from get_shipment. Never guess "+
				"one.", 0),
		},
		required: []string{paramShipmentMoveID},
		target: func(params map[string]any) (serviceports.ToolTarget, bool) {
			return targetOf(params, paramShipmentMoveID, permission.ResourceShipmentMove)
		},
	}), receivablePlan[serviceports.RecalculateMoveJurisdictionMilesRequest, *serviceports.MoveJurisdictionMilesPlan]{
		request: moveJurisdictionRequest,
		plan: func(
			ctx context.Context,
			req serviceports.RecalculateMoveJurisdictionMilesRequest,
			_ *serviceports.ToolExecuteParams,
		) (*serviceports.MoveJurisdictionMilesPlan, error) {
			return router.PlanMoveJurisdictionMiles(ctx, &req)
		},
		refused: func(serviceports.RecalculateMoveJurisdictionMilesRequest) string {
			return "Would ask for a move's jurisdiction miles again."
		},
		render: func(
			_ serviceports.RecalculateMoveJurisdictionMilesRequest,
			plan *serviceports.MoveJurisdictionMilesPlan,
		) (*agent.ToolPreview, error) {
			return toolpreview.Build(fmt.Sprintf(
				"Would ask the routing provider for the state-by-state miles of the move "+
					"across %s and replace its %s; its distance stays as it is.",
				countOf(len(plan.Move.Stops), "stop"),
				countOf(len(plan.Move.JurisdictionMiles), "jurisdiction row"),
			)), nil
		},
		run: func(
			ctx context.Context,
			req serviceports.RecalculateMoveJurisdictionMilesRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
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
	returns iftaReturnKeeper,
	params *serviceports.ToolExecuteParams,
	dryRun bool,
) (serviceports.BackfillJurisdictionMilesRequest, error) {
	info, err := returns.PeriodInfo(ctx, tenantFrom(*params), b.period.Year, b.period.Quarter)
	if err != nil {
		return serviceports.BackfillJurisdictionMilesRequest{}, err
	}

	return serviceports.BackfillJurisdictionMilesRequest{
		TenantInfo: tenantFrom(*params),
		Start:      info.Start,
		End:        info.End,
		MaxMoves:   b.moves,
		DryRun:     dryRun,
		UserID:     params.Actor.UserID,
	}, nil
}

func newBackfillJurisdictionMilesTool(
	returns iftaReturnKeeper,
	router jurisdictionMileRouter,
) serviceports.AgentTool {
	properties := iftaPeriodProperties()
	properties[paramMaxMoves] = integerProperty("The most moves to route in this run. "+
		"Defaults to 2000; each is a billable distance request.", 1, maxBackfillMoves)

	return newReceivableTool(iftaInternalSpec(&receivableSpec{
		name: "backfill_jurisdiction_miles",
		description: "Start a background job that routes every completed move in a quarter " +
			"that has no state-by-state miles. The IFTA return then stops showing them " +
			"unattributed. The preview counts the moves and " +
			"miles first; each move is a billable distance request.",
		resource:  permission.ResourceIFTAReturn,
		operation: permission.OpManage,
		rationale: "Routes a quarter's unattributed moves inside Trenova in the background; " +
			"nothing is sent to a customer or carrier, but each move is a billable request.",
		properties: properties,
		required:   []string{paramYear, paramQuarter},
	}), receivablePlan[*backfillRequest, *serviceports.BackfillJurisdictionMilesResult]{
		request: func(params *serviceports.ToolExecuteParams) (*backfillRequest, error) {
			period, err := iftaPeriodFrom(params)
			if err != nil {
				return nil, err
			}
			moves := 0
			if _, given := params.Params[paramMaxMoves]; given {
				if moves, err = requireIntInRange(
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
		plan: func(
			ctx context.Context,
			req *backfillRequest,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.BackfillJurisdictionMilesResult, error) {
			dry, err := req.service(ctx, returns, params, true)
			if err != nil {
				return nil, err
			}

			return router.BackfillJurisdictionMiles(ctx, dry)
		},
		refused: func(req *backfillRequest) string {
			return fmt.Sprintf("Would backfill jurisdiction miles for %s.", req.period.Label())
		},
		render: func(
			req *backfillRequest,
			result *serviceports.BackfillJurisdictionMilesResult,
		) (*agent.ToolPreview, error) {
			if result.UnattributedMoves == 0 {
				return toolpreview.Build(fmt.Sprintf(
					"Every completed move in %s already has jurisdiction miles; nothing would "+
						"be routed.", req.period.Label(),
				)), nil
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would start routing %s in %s with no jurisdiction miles, %s miles in all; "+
					"each is a billable distance request.",
				countOf(result.UnattributedMoves, "completed move"),
				req.period.Label(),
				result.UnattributedMiles.StringFixed(1),
			)), nil
		},
		run: func(
			ctx context.Context,
			req *backfillRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			started, err := req.service(ctx, returns, params, false)
			if err != nil {
				return nil, err
			}
			_, err = router.BackfillJurisdictionMiles(ctx, started)

			return nil, err
		},
	})
}
