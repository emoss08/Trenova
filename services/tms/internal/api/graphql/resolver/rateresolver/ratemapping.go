package rateresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratezone"
	"github.com/emoss08/trenova/pkg/pagination"
)

func rateAgreementToModel(
	entity *rateagreement.RateAgreement,
	inputs map[string]any,
) *gqlmodel.RateAgreement {
	if entity == nil {
		return nil
	}

	return &gqlmodel.RateAgreement{
		ID:                   entity.ID.String(),
		BusinessUnitID:       entity.BusinessUnitID.String(),
		OrganizationID:       entity.OrganizationID.String(),
		PartyType:            gqlmodel.RateAgreementPartyType(entity.PartyType),
		CustomerID:           base.IDPtrFromPulidPtr(entity.CustomerID),
		CarrierID:            base.IDPtrFromPulidPtr(entity.CarrierID),
		Code:                 entity.Code,
		Name:                 entity.Name,
		Description:          entity.Description,
		AgreementType:        gqlmodel.RateAgreementType(entity.AgreementType),
		Status:               gqlmodel.RateAgreementStatus(entity.Status),
		ContractRef:          entity.ContractRef,
		Priority:             int(entity.Priority),
		EffectiveFrom:        int(entity.EffectiveFrom),
		EffectiveTo:          base.Int64PtrToIntPtr(entity.EffectiveTo),
		AutoRenew:            entity.AutoRenew,
		RenewalNoticeDays:    int(entity.RenewalNoticeDays),
		Currency:             entity.Currency,
		Inputs:               inputs,
		MarginFloorPercent:   base.NullDecimalToStringPtr(entity.MarginFloorPercent),
		MaxPayPercentOfSell:  base.NullDecimalToStringPtr(entity.MaxPayPercentOfSell),
		SubmittedByID:        base.IDPtrFromPulidPtr(entity.SubmittedByID),
		SubmittedAt:          base.Int64PtrToIntPtr(entity.SubmittedAt),
		ApprovedByID:         base.IDPtrFromPulidPtr(entity.ApprovedByID),
		ApprovedAt:           base.Int64PtrToIntPtr(entity.ApprovedAt),
		ReviewComment:        entity.ReviewComment,
		CurrentVersionNumber: int(entity.CurrentVersionNumber),
		BillToCustomerID:     base.IDPtrFromPulidPtr(entity.BillToCustomerID),
		Version:              int(entity.Version),
		CreatedAt:            int(entity.CreatedAt),
		UpdatedAt:            int(entity.UpdatedAt),
	}
}

func rateAgreementConnectionToModel(
	result *pagination.CursorListResult[*rateagreement.RateAgreement],
) (*gqlmodel.RateAgreementConnection, error) {
	inputs := make(map[*rateagreement.RateAgreement]map[string]any, len(result.Items))
	for _, item := range result.Items {
		if item == nil {
			continue
		}
		values, err := base.TariffInputsToModel(item.Inputs)
		if err != nil {
			return nil, err
		}
		inputs[item] = values
	}
	page, err := base.EntityCursorConnection(
		result,
		func(node *rateagreement.RateAgreement, cursor string) *gqlmodel.RateAgreementEdge {
			return &gqlmodel.RateAgreementEdge{
				Node:   rateAgreementToModel(node, inputs[node]),
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.RateAgreementEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.RateAgreementConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func rateZoneToModel(entity *ratezone.RateZone) *gqlmodel.RateZone {
	if entity == nil {
		return nil
	}

	return &gqlmodel.RateZone{
		ID:             entity.ID.String(),
		BusinessUnitID: entity.BusinessUnitID.String(),
		OrganizationID: entity.OrganizationID.String(),
		Code:           entity.Code,
		Name:           entity.Name,
		Description:    entity.Description,
		Status:         string(entity.Status),
		Version:        int(entity.Version),
		CreatedAt:      int(entity.CreatedAt),
		UpdatedAt:      int(entity.UpdatedAt),
	}
}

func rateZoneConnectionToModel(
	result *pagination.CursorListResult[*ratezone.RateZone],
) (*gqlmodel.RateZoneConnection, error) {
	page, err := base.EntityCursorConnection(
		result,
		func(node *ratezone.RateZone, cursor string) *gqlmodel.RateZoneEdge {
			return &gqlmodel.RateZoneEdge{Node: rateZoneToModel(node), Cursor: cursor}
		},
		func(edge *gqlmodel.RateZoneEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.RateZoneConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func rateMatrixToModel(entity *ratematrix.RateMatrix) *gqlmodel.RateMatrix {
	if entity == nil {
		return nil
	}

	model := &gqlmodel.RateMatrix{
		ID:                     entity.ID.String(),
		BusinessUnitID:         entity.BusinessUnitID.String(),
		OrganizationID:         entity.OrganizationID.String(),
		Code:                   entity.Code,
		Name:                   entity.Name,
		Description:            entity.Description,
		Status:                 entity.Status,
		ValueType:              entity.ValueType,
		CellCount:              int(entity.DraftCellCount),
		PublishedVersionNumber: int(entity.PublishedVersionNumber),
		PublishedAt:            base.Int64PtrToIntPtr(entity.PublishedAt),
		HasDraftChanges:        entity.HasDraftChanges,
		SubmittedByID:          base.IDPtrFromPtr(entity.SubmittedByID),
		SubmittedAt:            base.Int64PtrToIntPtr(entity.SubmittedAt),
		ApprovedByID:           base.IDPtrFromPtr(entity.ApprovedByID),
		ApprovedAt:             base.Int64PtrToIntPtr(entity.ApprovedAt),
		ReviewComment:          entity.ReviewComment,
		Version:                int(entity.Version),
		CreatedAt:              int(entity.CreatedAt),
		UpdatedAt:              int(entity.UpdatedAt),
	}

	return model
}

func rateMatrixDimensionsToModel(
	dimensions []*ratematrix.RateMatrixDimension,
) []*gqlmodel.RateMatrixDimension {
	out := make([]*gqlmodel.RateMatrixDimension, 0, len(dimensions))
	for _, dimension := range dimensions {
		if dimension == nil {
			continue
		}
		out = append(out, &gqlmodel.RateMatrixDimension{
			ID:               dimension.ID.String(),
			Position:         int(dimension.Position),
			Kind:             string(dimension.Kind),
			MatchMode:        string(dimension.MatchMode),
			Label:            dimension.Label,
			KeyNormalization: string(dimension.KeyNormalization),
			RangeOverflow:    string(dimension.RangeOverflow),
		})
	}
	return out
}

func rateMatrixConnectionToModel(
	result *pagination.CursorListResult[*ratematrix.RateMatrix],
) (*gqlmodel.RateMatrixConnection, error) {
	page, err := base.EntityCursorConnection(
		result,
		func(node *ratematrix.RateMatrix, cursor string) *gqlmodel.RateMatrixEdge {
			return &gqlmodel.RateMatrixEdge{Node: rateMatrixToModel(node), Cursor: cursor}
		},
		func(edge *gqlmodel.RateMatrixEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.RateMatrixConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func rateQuoteToModel(entity *ratequote.RateQuote) *gqlmodel.RateQuote {
	if entity == nil {
		return nil
	}

	return &gqlmodel.RateQuote{
		ID:                  entity.ID.String(),
		BusinessUnitID:      entity.BusinessUnitID.String(),
		OrganizationID:      entity.OrganizationID.String(),
		ShipmentID:          base.IDPtrFromPulidPtr(entity.ShipmentID),
		PartyType:           gqlmodel.RateAgreementPartyType(entity.PartyType),
		PartyID:             entity.PartyID.String(),
		Purpose:             gqlmodel.RateQuotePurpose(entity.Purpose),
		Outcome:             gqlmodel.RateQuoteOutcome(entity.Outcome),
		RateAgreementID:     base.IDPtrFromPulidPtr(entity.RateAgreementID),
		RateAgreementRuleID: base.IDPtrFromPulidPtr(entity.RateAgreementRuleID),
		FormulaTemplateID:   base.IDPtrFromPulidPtr(entity.FormulaTemplateID),
		SpecificityScore:    int(entity.SpecificityScore),
		Currency:            entity.Currency,
		BillingCurrency:     entity.BillingCurrency,
		LinehaulAmount:      entity.LinehaulAmount.String(),
		TotalAmount:         entity.TotalAmount.String(),
		BillingAmount:       entity.BillingAmount.String(),
		ForegoneAmount:      base.NullDecimalToStringPtr(entity.ForegoneAmount),
		OverrideReason:      entity.OverrideReason,
		AsOf:                int(entity.AsOf),
		RatedAt:             int(entity.RatedAt),
		RatedByID:           base.IDPtrFromPulidPtr(entity.RatedByID),
		EngineVersion:       entity.EngineVersion,
		CreatedAt:           int(entity.CreatedAt),
	}
}

func rateQuoteConnectionToModel(
	result *pagination.CursorListResult[*ratequote.RateQuote],
) (*gqlmodel.RateQuoteConnection, error) {
	page, err := base.EntityCursorConnection(
		result,
		func(node *ratequote.RateQuote, cursor string) *gqlmodel.RateQuoteEdge {
			return &gqlmodel.RateQuoteEdge{Node: rateQuoteToModel(node), Cursor: cursor}
		},
		func(edge *gqlmodel.RateQuoteEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.RateQuoteConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

// rateAgreementCardToModel is one agreement as rateAgreement reads it: the
// header with its accessorial schedule, its fuel binding and the version of
// its terms in force. Its lanes are read a page at a time by their own field.
func rateAgreementCardToModel(
	entity *rateagreement.RateAgreement,
	current *rateagreement.RateAgreementVersion,
) (*gqlmodel.RateAgreement, error) {
	inputs, err := base.TariffInputsToModel(entity.Inputs)
	if err != nil {
		return nil, err
	}
	model := rateAgreementToModel(entity, inputs)

	model.Accessorials = make([]*gqlmodel.RateAgreementAccessorial, 0, len(entity.Accessorials))
	for _, accessorial := range entity.Accessorials {
		if accessorial != nil {
			model.Accessorials = append(
				model.Accessorials,
				rateAgreementAccessorialToModel(accessorial),
			)
		}
	}
	if binding := entity.FuelBinding; binding != nil {
		model.FuelBinding = &gqlmodel.RateAgreementFuelBinding{
			ID:                     binding.ID.String(),
			FuelSurchargeProgramID: binding.FuelSurchargeProgramID.String(),
			Waived:                 binding.Waived,
		}
	}
	if current != nil {
		model.CurrentVersion = &gqlmodel.RateAgreementVersion{
			ID:            current.ID.String(),
			VersionNumber: int(current.VersionNumber),
			EffectiveFrom: int(current.EffectiveFrom),
			EffectiveTo:   base.Int64PtrToIntPtr(current.EffectiveTo),
			ChangeMessage: current.ChangeMessage,
		}
	}

	return model, nil
}

func rateAgreementAccessorialToModel(
	entity *rateagreement.RateAgreementAccessorial,
) *gqlmodel.RateAgreementAccessorial {
	model := &gqlmodel.RateAgreementAccessorial{
		ID:                  entity.ID.String(),
		AccessorialChargeID: entity.AccessorialChargeID.String(),
		Method:              string(entity.Method),
		RateUnit:            string(entity.RateUnit),
		Waived:              entity.Waived,
		AutoApply:           entity.AutoApply,
		EffectiveFrom:       base.Int64PtrToIntPtr(entity.EffectiveFrom),
		EffectiveTo:         base.Int64PtrToIntPtr(entity.EffectiveTo),
		Amount:              entity.Amount.String(),
		MaxAmount:           base.NullDecimalToStringPtr(entity.MaxAmount),
		Condition:           entity.Condition,
	}
	if entity.FreeUnits != nil {
		units := int(*entity.FreeUnits)
		model.FreeUnits = &units
	}

	return model
}

func rateAgreementLaneToModel(
	entity *rateagreement.RateAgreementRule,
) (*gqlmodel.RateAgreementLane, error) {
	inputs, err := base.TariffInputsToModel(entity.Inputs)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.RateAgreementLane{
		ID:                    entity.ID.String(),
		Label:                 entity.Label,
		OriginScopeType:       string(entity.OriginScopeType),
		OriginScopeValue:      entity.OriginScopeValue,
		OriginCity:            entity.OriginCity,
		DestinationScopeType:  string(entity.DestinationScopeType),
		DestinationScopeValue: entity.DestinationScopeValue,
		DestinationCity:       entity.DestinationCity,
		Direction:             string(entity.Direction),
		Priority:              int(entity.Priority),
		EffectiveFrom:         int(entity.EffectiveFrom),
		EffectiveTo:           base.Int64PtrToIntPtr(entity.EffectiveTo),
		HazmatOnly:            entity.HazmatOnly,
		TempControlOnly:       entity.TempControlOnly,
		FormulaTemplateID:     entity.FormulaTemplateID.String(),
		Currency:              entity.Currency,
		Inputs:                inputs,
	}, nil
}

// rateMatrixCardToModel is one matrix as rateMatrix reads it, with its axes.
func rateMatrixCardToModel(entity *ratematrix.RateMatrix) *gqlmodel.RateMatrix {
	model := rateMatrixToModel(entity)
	if model == nil {
		return nil
	}

	model.Dimensions = rateMatrixDimensionsToModel(entity.Dimensions)

	return model
}

func rateMatrixCellToModel(entity *ratematrix.RateMatrixCell) *gqlmodel.RateMatrixCell {
	return &gqlmodel.RateMatrixCell{
		ID:              entity.ID.String(),
		D0Key:           entity.D0Key,
		D1Key:           entity.D1Key,
		D2Key:           entity.D2Key,
		D3Key:           entity.D3Key,
		D0KeyTo:         entity.D0KeyTo,
		D1KeyTo:         entity.D1KeyTo,
		D2KeyTo:         entity.D2KeyTo,
		D3KeyTo:         entity.D3KeyTo,
		D0Min:           base.NullDecimalToStringPtr(entity.D0Min),
		D0Max:           base.NullDecimalToStringPtr(entity.D0Max),
		D1Min:           base.NullDecimalToStringPtr(entity.D1Min),
		D1Max:           base.NullDecimalToStringPtr(entity.D1Max),
		D2Min:           base.NullDecimalToStringPtr(entity.D2Min),
		D2Max:           base.NullDecimalToStringPtr(entity.D2Max),
		D3Min:           base.NullDecimalToStringPtr(entity.D3Min),
		D3Max:           base.NullDecimalToStringPtr(entity.D3Max),
		DeficitEligible: entity.DeficitEligible,
		Value:           entity.Value.String(),
		MinCharge:       base.NullDecimalToStringPtr(entity.MinCharge),
	}
}
