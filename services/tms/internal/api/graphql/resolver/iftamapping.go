package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/iftaservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func iftaTaxRateCursorConnectionToModel(
	result *pagination.CursorListResult[*ifta.TaxRate],
) (*gqlmodel.IFTATaxRateConnection, error) {
	edges, err := entityCursorEdges(
		result.Items,
		result.CursorSort,
		result,
		func(node *ifta.TaxRate, cursor string) *gqlmodel.IFTATaxRateEdge {
			return &gqlmodel.IFTATaxRateEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.IFTATaxRateConnection{
		Edges: edges,
		PageInfo: pageInfo(
			result.HasNextPage,
			lastEdgeCursor(
				edges,
				func(edge *gqlmodel.IFTATaxRateEdge) string { return edge.Cursor },
			),
		),
		TotalCount: result.TotalCount,
	}, nil
}

func iftaMileageEntryCursorConnectionToModel(
	result *pagination.CursorListResult[*ifta.JurisdictionMileageEntry],
) (*gqlmodel.IFTAJurisdictionMileageEntryConnection, error) {
	edges, err := entityCursorEdges(
		result.Items,
		result.CursorSort,
		result,
		func(
			node *ifta.JurisdictionMileageEntry,
			cursor string,
		) *gqlmodel.IFTAJurisdictionMileageEntryEdge {
			return &gqlmodel.IFTAJurisdictionMileageEntryEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.IFTAJurisdictionMileageEntryConnection{
		Edges: edges,
		PageInfo: pageInfo(
			result.HasNextPage,
			lastEdgeCursor(
				edges,
				func(edge *gqlmodel.IFTAJurisdictionMileageEntryEdge) string { return edge.Cursor },
			),
		),
		TotalCount: result.TotalCount,
	}, nil
}

func iftaReturnCursorConnectionToModel(
	result *pagination.CursorListResult[*ifta.Return],
) (*gqlmodel.IFTAReturnConnection, error) {
	edges, err := entityCursorEdges(
		result.Items,
		result.CursorSort,
		result,
		func(node *ifta.Return, cursor string) *gqlmodel.IFTAReturnEdge {
			return &gqlmodel.IFTAReturnEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.IFTAReturnConnection{
		Edges: edges,
		PageInfo: pageInfo(
			result.HasNextPage,
			lastEdgeCursor(
				edges,
				func(edge *gqlmodel.IFTAReturnEdge) string { return edge.Cursor },
			),
		),
		TotalCount: result.TotalCount,
	}, nil
}

func iftaPeriodFromInput(input gqlmodel.IFTAPeriodInput) (ifta.Period, error) {
	period := ifta.NewPeriod(input.Year, input.Quarter)
	if err := period.Validate(); err != nil {
		return ifta.Period{}, errortypes.NewValidationError(
			"period",
			errortypes.ErrInvalid,
			"Year must be between 2000 and 2100 and quarter between 1 and 4",
		)
	}

	return period, nil
}

func iftaTaxRatesRequestFromGraphQL(
	connection gqlDataTableConnection,
	input gqlmodel.IFTATaxRatesInput,
) (*repositories.ListTaxRatesRequest, error) {
	jurisdictionID, err := optionalScopedID("jurisdictionId", input.JurisdictionID)
	if err != nil {
		return nil, err
	}

	req := &repositories.ListTaxRatesRequest{
		Filter:              connection.Filter,
		Cursor:              connection.Cursor,
		JurisdictionID:      jurisdictionID,
		IncludeJurisdiction: true,
	}
	if input.Period != nil {
		req.Year = input.Period.Year
		req.Quarter = input.Period.Quarter
	}
	if input.FuelType != nil {
		req.FuelType = *input.FuelType
	}

	return req, nil
}

func iftaMileageEntriesRequestFromGraphQL(
	connection gqlDataTableConnection,
	input gqlmodel.IFTAMileageEntriesInput,
) (*repositories.ListMileageEntriesRequest, error) {
	tractorID, err := optionalScopedID("tractorId", input.TractorID)
	if err != nil {
		return nil, err
	}

	jurisdictionID, err := optionalScopedID("jurisdictionId", input.JurisdictionID)
	if err != nil {
		return nil, err
	}

	req := &repositories.ListMileageEntriesRequest{
		Filter:              connection.Filter,
		Cursor:              connection.Cursor,
		TractorID:           tractorID,
		JurisdictionID:      jurisdictionID,
		Sources:             input.Sources,
		IncludeTractor:      true,
		IncludeJurisdiction: true,
	}
	if input.Period != nil {
		req.Year = input.Period.Year
		req.Quarter = input.Period.Quarter
	}

	return req, nil
}

func iftaReturnsRequestFromGraphQL(
	connection gqlDataTableConnection,
	input gqlmodel.IFTAReturnsInput,
) *repositories.ListReturnsRequest {
	return &repositories.ListReturnsRequest{
		Filter:   connection.Filter,
		Cursor:   connection.Cursor,
		Year:     intValue(input.Year),
		Statuses: input.Statuses,
	}
}

func iftaTaxRateFromInput(input *gqlmodel.IFTATaxRateInput) (*ifta.TaxRate, error) {
	if input == nil {
		return nil, errortypes.NewValidationError(
			"rates",
			errortypes.ErrRequired,
			"A rate is required",
		)
	}

	jurisdictionID, err := requiredID("jurisdictionId", input.JurisdictionID)
	if err != nil {
		return nil, err
	}

	rate, err := decimalFromString(input.RatePerGallon, "ratePerGallon")
	if err != nil {
		return nil, err
	}

	surcharge, err := nullDecimalFromStringPtr(
		input.SurchargeRatePerGallon,
		"surchargeRatePerGallon",
	)
	if err != nil {
		return nil, err
	}

	return &ifta.TaxRate{
		JurisdictionID:         jurisdictionID,
		Year:                   input.Year,
		Quarter:                input.Quarter,
		FuelType:               input.FuelType,
		RatePerGallon:          rate,
		SurchargeRatePerGallon: surcharge,
		SourceNote:             stringValue(input.SourceNote),
		SourceURL:              stringValue(input.SourceURL),
	}, nil
}

func iftaTaxRatesFromInput(inputs []*gqlmodel.IFTATaxRateInput) ([]*ifta.TaxRate, error) {
	rates := make([]*ifta.TaxRate, 0, len(inputs))
	for _, input := range inputs {
		rate, err := iftaTaxRateFromInput(input)
		if err != nil {
			return nil, err
		}
		rates = append(rates, rate)
	}

	return rates, nil
}

func iftaMileageEntryFromInput(
	input gqlmodel.IFTAMileageEntryInput,
	tenant pagination.TenantInfo,
) (*ifta.JurisdictionMileageEntry, error) {
	entry := &ifta.JurisdictionMileageEntry{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	}
	if err := applyIFTAMileageEntryInput(entry, input); err != nil {
		return nil, err
	}

	return entry, nil
}

func applyIFTAMileageEntryInput(
	entry *ifta.JurisdictionMileageEntry,
	input gqlmodel.IFTAMileageEntryInput,
) error {
	tractorID, err := requiredID("tractorId", input.TractorID)
	if err != nil {
		return err
	}

	jurisdictionID, err := requiredID("jurisdictionId", input.JurisdictionID)
	if err != nil {
		return err
	}

	moveID, err := optionalScopedID("shipmentMoveId", input.ShipmentMoveID)
	if err != nil {
		return err
	}

	miles, err := decimalFromString(input.Miles, "miles")
	if err != nil {
		return err
	}

	entry.TractorID = tractorID
	entry.JurisdictionID = jurisdictionID
	entry.TraveledAt = int64(input.TraveledAt)
	entry.Miles = miles
	entry.ShipmentMoveID = idValuePtr(moveID)
	entry.Notes = stringValue(input.Notes)
	entry.Loaded = true
	if input.Loaded != nil {
		entry.Loaded = *input.Loaded
	}
	if input.Source != nil {
		entry.Source = *input.Source
	}

	return nil
}

func backfillResultToModel(
	result *serviceports.BackfillJurisdictionMilesResult,
) *gqlmodel.JurisdictionMilesBackfillResult {
	if result == nil {
		return nil
	}

	return &gqlmodel.JurisdictionMilesBackfillResult{
		Started:           result.Started,
		DryRun:            result.DryRun,
		UnattributedMoves: result.UnattributedMoves,
		UnattributedMiles: result.UnattributedMiles.StringFixed(2),
		WorkflowID:        nonEmptyPtr(result.WorkflowID),
	}
}

func zeroIfEmpty(value string) string {
	if value == "" {
		return "0"
	}
	return value
}

func loadUser(ctx context.Context, id pulid.ID) (*tenant.User, error) {
	if id.IsNil() {
		return nil, nil
	}

	l, ok := loaders.FromContext(ctx)
	if !ok || l == nil {
		return nil, nil
	}

	return l.UserByID.Load(ctx, id.String())
}

func returnActionRequest(
	authCtx *authctx.AuthContext,
	id string,
	version int,
) (*iftaservice.ReturnActionRequest, error) {
	returnID, err := requiredID("id", id)
	if err != nil {
		return nil, err
	}

	return &iftaservice.ReturnActionRequest{
		TenantInfo: tenantInfo(authCtx),
		ID:         returnID,
		Version:    int64(version),
		UserID:     authCtx.UserID,
	}, nil
}

func (r *iFTAPeriodResolver) periodInfo(
	ctx context.Context,
	obj *ifta.Period,
) (iftaservice.PeriodInfo, error) {
	authCtx, err := r.requireAuth(ctx)
	if err != nil {
		return iftaservice.PeriodInfo{}, err
	}

	return r.iftaService.PeriodInfo(ctx, tenantInfo(authCtx), obj.Year, obj.Quarter)
}
