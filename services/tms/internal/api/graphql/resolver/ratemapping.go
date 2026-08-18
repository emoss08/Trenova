package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/ratezone"
	"github.com/emoss08/trenova/pkg/pagination"
)

func rateAgreementToModel(entity *rateagreement.RateAgreement) *gqlmodel.RateAgreement {
	if entity == nil {
		return nil
	}

	return &gqlmodel.RateAgreement{
		ID:                   entity.ID.String(),
		BusinessUnitID:       entity.BusinessUnitID.String(),
		OrganizationID:       entity.OrganizationID.String(),
		PartyType:            gqlmodel.RateAgreementPartyType(entity.PartyType),
		CustomerID:           idPtrFromPulidPtr(entity.CustomerID),
		CarrierID:            idPtrFromPulidPtr(entity.CarrierID),
		Code:                 entity.Code,
		Name:                 entity.Name,
		Description:          entity.Description,
		AgreementType:        gqlmodel.RateAgreementType(entity.AgreementType),
		Status:               gqlmodel.RateAgreementStatus(entity.Status),
		ContractRef:          entity.ContractRef,
		Priority:             int(entity.Priority),
		EffectiveFrom:        int(entity.EffectiveFrom),
		EffectiveTo:          int64PtrToIntPtr(entity.EffectiveTo),
		AutoRenew:            entity.AutoRenew,
		RenewalNoticeDays:    int(entity.RenewalNoticeDays),
		Currency:             entity.Currency,
		DefaultMinCharge:     nullDecimalToStringPtr(entity.DefaultMinCharge),
		DefaultMaxCharge:     nullDecimalToStringPtr(entity.DefaultMaxCharge),
		MarginFloorPercent:   nullDecimalToStringPtr(entity.MarginFloorPercent),
		MaxPayPercentOfSell:  nullDecimalToStringPtr(entity.MaxPayPercentOfSell),
		SubmittedByID:        idPtrFromPulidPtr(entity.SubmittedByID),
		SubmittedAt:          int64PtrToIntPtr(entity.SubmittedAt),
		ApprovedByID:         idPtrFromPulidPtr(entity.ApprovedByID),
		ApprovedAt:           int64PtrToIntPtr(entity.ApprovedAt),
		ReviewComment:        entity.ReviewComment,
		CurrentVersionNumber: int(entity.CurrentVersionNumber),
		Version:              int(entity.Version),
		CreatedAt:            int(entity.CreatedAt),
		UpdatedAt:            int(entity.UpdatedAt),
	}
}

func rateAgreementConnectionToModel(
	result *pagination.CursorListResult[*rateagreement.RateAgreement],
) (*gqlmodel.RateAgreementConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *rateagreement.RateAgreement, cursor string) *gqlmodel.RateAgreementEdge {
			return &gqlmodel.RateAgreementEdge{
				Node:   rateAgreementToModel(node),
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
	page, err := entityCursorConnection(
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

	return &gqlmodel.RateMatrix{
		ID:             entity.ID.String(),
		BusinessUnitID: entity.BusinessUnitID.String(),
		OrganizationID: entity.OrganizationID.String(),
		Code:           entity.Code,
		Name:           entity.Name,
		Description:    entity.Description,
		Status:         string(entity.Status),
		ValueKind:      string(entity.ValueKind),
		Currency:       entity.Currency,
		Version:        int(entity.Version),
		CreatedAt:      int(entity.CreatedAt),
		UpdatedAt:      int(entity.UpdatedAt),
	}
}

func rateMatrixConnectionToModel(
	result *pagination.CursorListResult[*ratematrix.RateMatrix],
) (*gqlmodel.RateMatrixConnection, error) {
	page, err := entityCursorConnection(
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
		ShipmentID:          idPtrFromPulidPtr(entity.ShipmentID),
		PartyType:           gqlmodel.RateAgreementPartyType(entity.PartyType),
		PartyID:             entity.PartyID.String(),
		Purpose:             gqlmodel.RateQuotePurpose(entity.Purpose),
		Outcome:             gqlmodel.RateQuoteOutcome(entity.Outcome),
		RateAgreementID:     idPtrFromPulidPtr(entity.RateAgreementID),
		RateAgreementRuleID: idPtrFromPulidPtr(entity.RateAgreementRuleID),
		FormulaTemplateID:   idPtrFromPulidPtr(entity.FormulaTemplateID),
		SpecificityScore:    int(entity.SpecificityScore),
		Currency:            entity.Currency,
		BillingCurrency:     entity.BillingCurrency,
		LinehaulAmount:      entity.LinehaulAmount.String(),
		TotalAmount:         entity.TotalAmount.String(),
		BillingAmount:       entity.BillingAmount.String(),
		ForegoneAmount:      nullDecimalToStringPtr(entity.ForegoneAmount),
		OverrideReason:      entity.OverrideReason,
		AsOf:                int(entity.AsOf),
		RatedAt:             int(entity.RatedAt),
		RatedByID:           idPtrFromPulidPtr(entity.RatedByID),
		EngineVersion:       entity.EngineVersion,
		CreatedAt:           int(entity.CreatedAt),
	}
}

func rateQuoteConnectionToModel(
	result *pagination.CursorListResult[*ratequote.RateQuote],
) (*gqlmodel.RateQuoteConnection, error) {
	page, err := entityCursorConnection(
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
