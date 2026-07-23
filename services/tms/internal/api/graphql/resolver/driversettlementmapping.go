package resolver

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

func (r *mutationResolver) settlementAction(
	ctx context.Context,
	settlementID string,
	op permission.Operation,
) (*authctx.AuthContext, pulid.ID, error) {
	authCtx, err := r.requirePermission(ctx, permission.ResourceDriverSettlement, op)
	if err != nil {
		return nil, pulid.Nil, err
	}
	id, err := pulid.MustParse(settlementID)
	if err != nil {
		return nil, pulid.Nil, errortypes.NewValidationError(
			"settlementId",
			errortypes.ErrInvalid,
			"Invalid settlement",
		)
	}
	return authCtx, id, nil
}

func payProfileConnectionToModel(
	result *pagination.CursorListResult[*driverpay.PayProfile],
) (*gqlmodel.PayProfileConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *driverpay.PayProfile, cursor string) *gqlmodel.PayProfileEdge {
			return &gqlmodel.PayProfileEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.PayProfileEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.PayProfileConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func payCodeConnectionToModel(
	result *pagination.CursorListResult[*driverpay.PayCode],
) (*gqlmodel.PayCodeConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *driverpay.PayCode, cursor string) *gqlmodel.PayCodeEdge {
			return &gqlmodel.PayCodeEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.PayCodeEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.PayCodeConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func settlementDisputeConnectionToModel(
	result *pagination.CursorListResult[*driversettlement.Dispute],
) (*gqlmodel.SettlementDisputeConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *driversettlement.Dispute, cursor string) *gqlmodel.SettlementDisputeEdge {
			return &gqlmodel.SettlementDisputeEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.SettlementDisputeEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.SettlementDisputeConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func driverExpenseConnectionToModel(
	result *pagination.CursorListResult[*driverpay.Expense],
) (*gqlmodel.DriverExpenseConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *driverpay.Expense, cursor string) *gqlmodel.DriverExpenseEdge {
			return &gqlmodel.DriverExpenseEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.DriverExpenseEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.DriverExpenseConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func payCodeFromCreateInput(
	input *gqlmodel.CreatePayCodeInput,
	tenantInfo pagination.TenantInfo,
) (*driverpay.PayCode, error) {
	glAccountID, err := optionalID(input.GlAccountID)
	if err != nil {
		return nil, err
	}
	entity := &driverpay.PayCode{
		OrganizationID:        tenantInfo.OrgID,
		BusinessUnitID:        tenantInfo.BuID,
		Status:                domaintypes.StatusActive,
		Direction:             input.Direction,
		Code:                  strings.ToUpper(strings.TrimSpace(input.Code)),
		Name:                  input.Name,
		Description:           stringValue(input.Description),
		Taxable:               true,
		CountsTowardGuarantee: true,
		DefaultAmountMinor:    int64Ptr(input.DefaultAmountMinor),
	}
	if input.Taxable != nil {
		entity.Taxable = *input.Taxable
	}
	if input.CountsTowardGuarantee != nil {
		entity.CountsTowardGuarantee = *input.CountsTowardGuarantee
	}
	if !glAccountID.IsNil() {
		entity.GLAccountID = &glAccountID
	}
	return entity, nil
}

func payCodeFromUpdateInput(
	input *gqlmodel.UpdatePayCodeInput,
	tenantInfo pagination.TenantInfo,
) (*driverpay.PayCode, error) {
	payCodeID, err := pulid.MustParse(input.ID)
	if err != nil {
		return nil, err
	}
	glAccountID, err := optionalID(input.GlAccountID)
	if err != nil {
		return nil, err
	}
	entity := &driverpay.PayCode{
		ID:                    payCodeID,
		OrganizationID:        tenantInfo.OrgID,
		BusinessUnitID:        tenantInfo.BuID,
		Version:               int64(input.Version),
		Status:                input.Status,
		Code:                  strings.ToUpper(strings.TrimSpace(input.Code)),
		Name:                  input.Name,
		Description:           stringValue(input.Description),
		Taxable:               input.Taxable,
		CountsTowardGuarantee: input.CountsTowardGuarantee,
		DefaultAmountMinor:    int64Ptr(input.DefaultAmountMinor),
	}
	if !glAccountID.IsNil() {
		entity.GLAccountID = &glAccountID
	}
	return entity, nil
}

func recurringDeductionConnectionToModel(
	result *pagination.CursorListResult[*driverpay.RecurringDeduction],
) (*gqlmodel.RecurringDeductionConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *driverpay.RecurringDeduction, cursor string) *gqlmodel.RecurringDeductionEdge {
			return &gqlmodel.RecurringDeductionEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.RecurringDeductionEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.RecurringDeductionConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func recurringEarningConnectionToModel(
	result *pagination.CursorListResult[*driverpay.RecurringEarning],
) (*gqlmodel.RecurringEarningConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *driverpay.RecurringEarning, cursor string) *gqlmodel.RecurringEarningEdge {
			return &gqlmodel.RecurringEarningEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.RecurringEarningEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.RecurringEarningConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func payAdvanceConnectionToModel(
	result *pagination.CursorListResult[*driverpay.PayAdvance],
) (*gqlmodel.PayAdvanceConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *driverpay.PayAdvance, cursor string) *gqlmodel.PayAdvanceEdge {
			return &gqlmodel.PayAdvanceEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.PayAdvanceEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.PayAdvanceConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func escrowAccountConnectionToModel(
	result *pagination.CursorListResult[*driverpay.EscrowAccount],
) (*gqlmodel.EscrowAccountConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *driverpay.EscrowAccount, cursor string) *gqlmodel.EscrowAccountEdge {
			return &gqlmodel.EscrowAccountEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.EscrowAccountEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.EscrowAccountConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func driverSettlementConnectionToModel(
	result *pagination.CursorListResult[*driversettlement.Settlement],
) (*gqlmodel.DriverSettlementConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *driversettlement.Settlement, cursor string) *gqlmodel.DriverSettlementEdge {
			return &gqlmodel.DriverSettlementEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.DriverSettlementEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.DriverSettlementConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func settlementBatchConnectionToModel(
	result *pagination.CursorListResult[*driversettlement.SettlementBatch],
) (*gqlmodel.SettlementBatchConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *driversettlement.SettlementBatch, cursor string) *gqlmodel.SettlementBatchEdge {
			return &gqlmodel.SettlementBatchEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.SettlementBatchEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.SettlementBatchConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func driverPayEventConnectionToModel(
	result *pagination.CursorListResult[*driversettlement.PayEvent],
) (*gqlmodel.DriverPayEventConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *driversettlement.PayEvent, cursor string) *gqlmodel.DriverPayEventEdge {
			return &gqlmodel.DriverPayEventEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.DriverPayEventEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.DriverPayEventConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func optionalDecimalFromString(value *string, field string) (decimal.Decimal, error) {
	if value == nil || *value == "" {
		return decimal.Zero, nil
	}
	return decimalFromString(*value, field)
}

func mileageBandsFromInput(
	inputs []*gqlmodel.PayMileageBandInput,
) ([]driverpay.MileageBand, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	bands := make([]driverpay.MileageBand, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		rate, err := decimalFromString(input.Rate, "bands.rate")
		if err != nil {
			return nil, err
		}
		bands = append(bands, driverpay.MileageBand{
			MinMiles: input.MinMiles,
			MaxMiles: input.MaxMiles,
			Rate:     rate,
		})
	}
	return bands, nil
}

func payProfileComponentsFromInput(
	inputs []*gqlmodel.PayProfileComponentInput,
) ([]*driverpay.PayProfileComponent, error) {
	components := make([]*driverpay.PayProfileComponent, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		rate, err := decimalFromString(input.Rate, "components.rate")
		if err != nil {
			return nil, err
		}
		bands, err := mileageBandsFromInput(input.Bands)
		if err != nil {
			return nil, err
		}
		component := &driverpay.PayProfileComponent{
			Kind:            input.Kind,
			Method:          input.Method,
			Description:     stringValue(input.Description),
			Rate:            rate,
			Bands:           bands,
			FreeTimeMinutes: intValue(input.FreeTimeMinutes),
			MinAmountMinor:  int64Ptr(input.MinAmountMinor),
			MaxAmountMinor:  int64Ptr(input.MaxAmountMinor),
			IsActive:        true,
		}
		if input.RevenueBasis != nil {
			component.RevenueBasis = *input.RevenueBasis
		}
		if input.IsActive != nil {
			component.IsActive = *input.IsActive
		}
		components = append(components, component)
	}
	return components, nil
}

func payProfileFromCreateInput(
	input *gqlmodel.CreatePayProfileInput,
	tenantInfo pagination.TenantInfo,
) (*driverpay.PayProfile, error) {
	perDiemRate, err := optionalDecimalFromString(input.PerDiemRatePerMile, "perDiemRatePerMile")
	if err != nil {
		return nil, err
	}
	components, err := payProfileComponentsFromInput(input.Components)
	if err != nil {
		return nil, err
	}
	entity := &driverpay.PayProfile{
		OrganizationID:               tenantInfo.OrgID,
		BusinessUnitID:               tenantInfo.BuID,
		Name:                         input.Name,
		Description:                  stringValue(input.Description),
		Classification:               input.Classification,
		CurrencyCode:                 money.DefaultCurrencyCode,
		GuaranteedPeriodMinimumMinor: int64Value(input.GuaranteedPeriodMinimumMinor),
		PerDiemRatePerMile:           perDiemRate,
		PerDiemDailyCapMinor:         int64Value(input.PerDiemDailyCapMinor),
		Components:                   components,
	}
	if input.Status != nil {
		entity.Status = *input.Status
	} else {
		entity.Status = "Active"
	}
	if input.CurrencyCode != nil && *input.CurrencyCode != "" {
		entity.CurrencyCode = *input.CurrencyCode
	}
	return entity, nil
}

func payProfileFromUpdateInput(
	input *gqlmodel.UpdatePayProfileInput,
	tenantInfo pagination.TenantInfo,
) (*driverpay.PayProfile, error) {
	profileID, err := pulid.MustParse(input.ID)
	if err != nil {
		return nil, err
	}
	perDiemRate, err := optionalDecimalFromString(input.PerDiemRatePerMile, "perDiemRatePerMile")
	if err != nil {
		return nil, err
	}
	components, err := payProfileComponentsFromInput(input.Components)
	if err != nil {
		return nil, err
	}
	entity := &driverpay.PayProfile{
		ID:                           profileID,
		OrganizationID:               tenantInfo.OrgID,
		BusinessUnitID:               tenantInfo.BuID,
		Version:                      int64(input.Version),
		Name:                         input.Name,
		Description:                  stringValue(input.Description),
		Classification:               input.Classification,
		CurrencyCode:                 money.DefaultCurrencyCode,
		GuaranteedPeriodMinimumMinor: int64Value(input.GuaranteedPeriodMinimumMinor),
		PerDiemRatePerMile:           perDiemRate,
		PerDiemDailyCapMinor:         int64Value(input.PerDiemDailyCapMinor),
		Components:                   components,
	}
	if input.Status != nil {
		entity.Status = *input.Status
	} else {
		entity.Status = "Active"
	}
	if input.CurrencyCode != nil && *input.CurrencyCode != "" {
		entity.CurrencyCode = *input.CurrencyCode
	}
	return entity, nil
}

func workerPayAssignmentFromInput(
	input *gqlmodel.AssignPayProfileInput,
	tenantInfo pagination.TenantInfo,
) (*driverpay.WorkerPayAssignment, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, err
	}
	profileID, err := pulid.MustParse(input.PayProfileID)
	if err != nil {
		return nil, err
	}
	splitPercent := decimal.NewFromInt(100)
	if input.SplitPercent != nil && *input.SplitPercent != "" {
		splitPercent, err = decimalFromString(*input.SplitPercent, "splitPercent")
		if err != nil {
			return nil, err
		}
	}
	overrides := make([]driverpay.RateOverride, 0, len(input.RateOverrides))
	for _, override := range input.RateOverrides {
		if override == nil {
			continue
		}
		componentID, parseErr := pulid.MustParse(override.ComponentID)
		if parseErr != nil {
			return nil, errortypes.NewValidationError(
				"rateOverrides.componentId",
				errortypes.ErrInvalid,
				"Invalid override component",
			)
		}
		rate, rateErr := decimalFromString(override.Rate, "rateOverrides.rate")
		if rateErr != nil {
			return nil, rateErr
		}
		overrides = append(overrides, driverpay.RateOverride{
			ComponentID: componentID,
			Rate:        rate,
		})
	}
	return &driverpay.WorkerPayAssignment{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		WorkerID:       workerID,
		PayProfileID:   profileID,
		EffectiveFrom:  int64(input.EffectiveFrom),
		EffectiveTo:    int64Ptr(input.EffectiveTo),
		SplitPercent:   splitPercent,
		RateOverrides:  overrides,
		Notes:          stringValue(input.Notes),
	}, nil
}

func recurringDeductionFromCreateInput(
	input *gqlmodel.CreateRecurringDeductionInput,
	tenantInfo pagination.TenantInfo,
) (*driverpay.RecurringDeduction, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, err
	}
	payCodeID, err := pulid.MustParse(input.PayCodeID)
	if err != nil {
		return nil, err
	}
	escrowAccountID, err := optionalID(input.EscrowAccountID)
	if err != nil {
		return nil, err
	}
	entity := &driverpay.RecurringDeduction{
		OrganizationID:      tenantInfo.OrgID,
		BusinessUnitID:      tenantInfo.BuID,
		WorkerID:            workerID,
		PayCodeID:           payCodeID,
		Status:              driverpay.DeductionStatusActive,
		Frequency:           driverpay.DeductionFrequencyEverySettlement,
		Description:         input.Description,
		AmountMinor:         int64(input.AmountMinor),
		TotalCapMinor:       int64Ptr(input.TotalCapMinor),
		StartDate:           int64(input.StartDate),
		EndDate:             int64Ptr(input.EndDate),
		CurrencyCode:        money.DefaultCurrencyCode,
		DeductedToDateMinor: 0,
	}
	if !escrowAccountID.IsNil() {
		entity.EscrowAccountID = &escrowAccountID
	}
	if input.Frequency != nil {
		entity.Frequency = *input.Frequency
	}
	if input.CurrencyCode != nil && *input.CurrencyCode != "" {
		entity.CurrencyCode = *input.CurrencyCode
	}
	return entity, nil
}

func recurringDeductionFromUpdateInput(
	input *gqlmodel.UpdateRecurringDeductionInput,
	tenantInfo pagination.TenantInfo,
) (*driverpay.RecurringDeduction, error) {
	deductionID, err := pulid.MustParse(input.ID)
	if err != nil {
		return nil, err
	}
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, err
	}
	payCodeID, err := pulid.MustParse(input.PayCodeID)
	if err != nil {
		return nil, err
	}
	escrowAccountID, err := optionalID(input.EscrowAccountID)
	if err != nil {
		return nil, err
	}
	entity := &driverpay.RecurringDeduction{
		ID:             deductionID,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Version:        int64(input.Version),
		WorkerID:       workerID,
		PayCodeID:      payCodeID,
		Status:         input.Status,
		Frequency:      input.Frequency,
		Description:    input.Description,
		AmountMinor:    int64(input.AmountMinor),
		TotalCapMinor:  int64Ptr(input.TotalCapMinor),
		StartDate:      int64(input.StartDate),
		EndDate:        int64Ptr(input.EndDate),
		CurrencyCode:   money.DefaultCurrencyCode,
	}
	if !escrowAccountID.IsNil() {
		entity.EscrowAccountID = &escrowAccountID
	}
	if input.CurrencyCode != nil && *input.CurrencyCode != "" {
		entity.CurrencyCode = *input.CurrencyCode
	}
	return entity, nil
}

func recurringEarningFromCreateInput(
	input *gqlmodel.CreateRecurringEarningInput,
	tenantInfo pagination.TenantInfo,
) (*driverpay.RecurringEarning, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, err
	}
	payCodeID, err := pulid.MustParse(input.PayCodeID)
	if err != nil {
		return nil, err
	}
	entity := &driverpay.RecurringEarning{
		OrganizationID:  tenantInfo.OrgID,
		BusinessUnitID:  tenantInfo.BuID,
		WorkerID:        workerID,
		PayCodeID:       payCodeID,
		Status:          driverpay.EarningStatusActive,
		Frequency:       driverpay.EarningFrequencyEverySettlement,
		Description:     input.Description,
		AmountMinor:     int64(input.AmountMinor),
		TotalCapMinor:   int64Ptr(input.TotalCapMinor),
		StartDate:       int64(input.StartDate),
		EndDate:         int64Ptr(input.EndDate),
		CurrencyCode:    money.DefaultCurrencyCode,
		PaidToDateMinor: 0,
	}
	if input.Frequency != nil {
		entity.Frequency = *input.Frequency
	}
	if input.CurrencyCode != nil && *input.CurrencyCode != "" {
		entity.CurrencyCode = *input.CurrencyCode
	}
	return entity, nil
}

func recurringEarningFromUpdateInput(
	input *gqlmodel.UpdateRecurringEarningInput,
	tenantInfo pagination.TenantInfo,
) (*driverpay.RecurringEarning, error) {
	earningID, err := pulid.MustParse(input.ID)
	if err != nil {
		return nil, err
	}
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, err
	}
	payCodeID, err := pulid.MustParse(input.PayCodeID)
	if err != nil {
		return nil, err
	}
	entity := &driverpay.RecurringEarning{
		ID:             earningID,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Version:        int64(input.Version),
		WorkerID:       workerID,
		PayCodeID:      payCodeID,
		Status:         input.Status,
		Frequency:      input.Frequency,
		Description:    input.Description,
		AmountMinor:    int64(input.AmountMinor),
		TotalCapMinor:  int64Ptr(input.TotalCapMinor),
		StartDate:      int64(input.StartDate),
		EndDate:        int64Ptr(input.EndDate),
		CurrencyCode:   money.DefaultCurrencyCode,
	}
	if input.CurrencyCode != nil && *input.CurrencyCode != "" {
		entity.CurrencyCode = *input.CurrencyCode
	}
	return entity, nil
}

func payAdvanceFromIssueInput(
	input *gqlmodel.IssuePayAdvanceInput,
	tenantInfo pagination.TenantInfo,
) (*driverpay.PayAdvance, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, err
	}
	entity := &driverpay.PayAdvance{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		WorkerID:       workerID,
		Source:         input.Source,
		Reference:      stringValue(input.Reference),
		IssuedDate:     int64(input.IssuedDate),
		AmountMinor:    int64(input.AmountMinor),
		Notes:          stringValue(input.Notes),
		CurrencyCode:   money.DefaultCurrencyCode,
	}
	if input.CurrencyCode != nil && *input.CurrencyCode != "" {
		entity.CurrencyCode = *input.CurrencyCode
	}
	return entity, nil
}

func escrowAccountFromOpenInput(
	input *gqlmodel.OpenEscrowAccountInput,
	tenantInfo pagination.TenantInfo,
) (*driverpay.EscrowAccount, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, err
	}
	interestRate, err := optionalDecimalFromString(input.AnnualInterestRate, "annualInterestRate")
	if err != nil {
		return nil, err
	}
	entity := &driverpay.EscrowAccount{
		OrganizationID:     tenantInfo.OrgID,
		BusinessUnitID:     tenantInfo.BuID,
		WorkerID:           workerID,
		TargetAmountMinor:  int64(input.TargetAmountMinor),
		AnnualInterestRate: interestRate,
		OpenedDate:         int64Value(input.OpenedDate),
		CurrencyCode:       money.DefaultCurrencyCode,
	}
	if input.CurrencyCode != nil && *input.CurrencyCode != "" {
		entity.CurrencyCode = *input.CurrencyCode
	}
	return entity, nil
}

func escrowAccountFromUpdateInput(
	input *gqlmodel.UpdateEscrowAccountInput,
	tenantInfo pagination.TenantInfo,
) (*driverpay.EscrowAccount, error) {
	accountID, err := pulid.MustParse(input.ID)
	if err != nil {
		return nil, err
	}
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, err
	}
	interestRate, err := decimalFromString(input.AnnualInterestRate, "annualInterestRate")
	if err != nil {
		return nil, err
	}
	return &driverpay.EscrowAccount{
		ID:                 accountID,
		OrganizationID:     tenantInfo.OrgID,
		BusinessUnitID:     tenantInfo.BuID,
		Version:            int64(input.Version),
		WorkerID:           workerID,
		TargetAmountMinor:  int64(input.TargetAmountMinor),
		AnnualInterestRate: interestRate,
		OpenedDate:         timeutils.NowUnix(),
		CurrencyCode:       money.DefaultCurrencyCode,
	}, nil
}

func parsePayEventIDs(rawIDs []string) ([]pulid.ID, error) {
	eventIDs := make([]pulid.ID, 0, len(rawIDs))
	for _, rawID := range rawIDs {
		eventID, err := pulid.MustParse(rawID)
		if err != nil {
			return nil, errortypes.NewValidationError(
				"payEventIds",
				errortypes.ErrInvalid,
				"Invalid pay event",
			)
		}
		eventIDs = append(eventIDs, eventID)
	}
	return eventIDs, nil
}

func settlementControlFromUpdateInput(
	input *gqlmodel.UpdateSettlementControlInput,
	tenantInfo pagination.TenantInfo,
) (*tenant.SettlementControl, error) {
	varianceThreshold, err := decimalFromString(input.VarianceThresholdPct, "varianceThresholdPct")
	if err != nil {
		return nil, err
	}
	escrowInterestRate, err := decimalFromString(
		input.DefaultEscrowInterestRate,
		"defaultEscrowInterestRate",
	)
	if err != nil {
		return nil, err
	}
	return &tenant.SettlementControl{
		OrganizationID:                tenantInfo.OrgID,
		BusinessUnitID:                tenantInfo.BuID,
		Version:                       int64(input.Version),
		PayPeriodFrequency:            input.PayPeriodFrequency,
		PeriodEndDayOfWeek:            input.PeriodEndDayOfWeek,
		PayDelayDays:                  input.PayDelayDays,
		PayTrigger:                    input.PayTrigger,
		AutoGenerateBatches:           input.AutoGenerateBatches,
		AutoApproveClean:              input.AutoApproveClean,
		AutoAttachAccruals:            input.AutoAttachAccruals,
		AutoPostOnApprove:             input.AutoPostOnApprove,
		AllowNegativeNet:              input.AllowNegativeNet,
		VarianceThresholdPct:          varianceThreshold,
		VarianceLookbackWeeks:         input.VarianceLookbackWeeks,
		DefaultEscrowInterestRate:     escrowInterestRate,
		EscrowInterestFrequencyMonths: input.EscrowInterestFrequencyMonths,
	}, nil
}
