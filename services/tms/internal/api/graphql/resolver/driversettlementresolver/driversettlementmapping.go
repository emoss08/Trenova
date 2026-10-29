package driversettlementresolver

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/settlementcontrol"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payadvance"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paycode"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payescrow"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/recurringpay"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

func (r *MutationResolver) settlementAction(
	ctx context.Context,
	settlementID string,
	op permission.Operation,
) (*authctx.AuthContext, pulid.ID, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceDriverSettlement, op)
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
	page, err := gqlconnection.EntityCursorConnection(
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
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func payCodeConnectionToModel(
	result *pagination.CursorListResult[*paycode.PayCode],
) (*gqlmodel.PayCodeConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *paycode.PayCode, cursor string) *gqlmodel.PayCodeEdge {
			return &gqlmodel.PayCodeEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.PayCodeEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.PayCodeConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func payCodeFromCreateInput(
	input *gqlmodel.CreatePayCodeInput,
	tenantInfo pagination.TenantInfo,
) (*paycode.PayCode, error) {
	glAccountID, err := gqlid.OptionalID(input.GlAccountID)
	if err != nil {
		return nil, err
	}
	entity := &paycode.PayCode{
		OrganizationID:        tenantInfo.OrgID,
		BusinessUnitID:        tenantInfo.BuID,
		Status:                domaintypes.StatusActive,
		Direction:             input.Direction,
		Code:                  strings.ToUpper(strings.TrimSpace(input.Code)),
		Name:                  input.Name,
		Description:           gqlscalar.StringValue(input.Description),
		Taxable:               true,
		CountsTowardGuarantee: true,
		DefaultAmountMinor:    gqlscalar.Int64Ptr(input.DefaultAmountMinor),
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
) (*paycode.PayCode, error) {
	payCodeID, err := pulid.MustParse(input.ID)
	if err != nil {
		return nil, err
	}
	glAccountID, err := gqlid.OptionalID(input.GlAccountID)
	if err != nil {
		return nil, err
	}
	entity := &paycode.PayCode{
		ID:                    payCodeID,
		OrganizationID:        tenantInfo.OrgID,
		BusinessUnitID:        tenantInfo.BuID,
		Version:               int64(input.Version),
		Status:                input.Status,
		Code:                  strings.ToUpper(strings.TrimSpace(input.Code)),
		Name:                  input.Name,
		Description:           gqlscalar.StringValue(input.Description),
		Taxable:               input.Taxable,
		CountsTowardGuarantee: input.CountsTowardGuarantee,
		DefaultAmountMinor:    gqlscalar.Int64Ptr(input.DefaultAmountMinor),
	}
	if !glAccountID.IsNil() {
		entity.GLAccountID = &glAccountID
	}
	return entity, nil
}

func recurringDeductionConnectionToModel(
	result *pagination.CursorListResult[*recurringpay.RecurringDeduction],
) (*gqlmodel.RecurringDeductionConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *recurringpay.RecurringDeduction, cursor string) *gqlmodel.RecurringDeductionEdge {
			return &gqlmodel.RecurringDeductionEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.RecurringDeductionEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.RecurringDeductionConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func recurringEarningConnectionToModel(
	result *pagination.CursorListResult[*recurringpay.RecurringEarning],
) (*gqlmodel.RecurringEarningConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *recurringpay.RecurringEarning, cursor string) *gqlmodel.RecurringEarningEdge {
			return &gqlmodel.RecurringEarningEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.RecurringEarningEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.RecurringEarningConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func payAdvanceConnectionToModel(
	result *pagination.CursorListResult[*payadvance.PayAdvance],
) (*gqlmodel.PayAdvanceConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *payadvance.PayAdvance, cursor string) *gqlmodel.PayAdvanceEdge {
			return &gqlmodel.PayAdvanceEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.PayAdvanceEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.PayAdvanceConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func escrowAccountConnectionToModel(
	result *pagination.CursorListResult[*payescrow.EscrowAccount],
) (*gqlmodel.EscrowAccountConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *payescrow.EscrowAccount, cursor string) *gqlmodel.EscrowAccountEdge {
			return &gqlmodel.EscrowAccountEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.EscrowAccountEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.EscrowAccountConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func driverSettlementConnectionToModel(
	result *pagination.CursorListResult[*driversettlement.Settlement],
) (*gqlmodel.DriverSettlementConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
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
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func settlementBatchConnectionToModel(
	result *pagination.CursorListResult[*driversettlement.SettlementBatch],
) (*gqlmodel.SettlementBatchConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
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
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func driverPayEventConnectionToModel(
	result *pagination.CursorListResult[*driversettlement.PayEvent],
) (*gqlmodel.DriverPayEventConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
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
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func optionalDecimalFromString(value *string, field string) (decimal.Decimal, error) {
	if value == nil || *value == "" {
		return decimal.Zero, nil
	}
	return gqldecimal.DecimalFromString(*value, field)
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
		rate, err := gqldecimal.DecimalFromString(input.Rate, "bands.rate")
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
		rate, err := gqldecimal.DecimalFromString(input.Rate, "components.rate")
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
			Description:     gqlscalar.StringValue(input.Description),
			Rate:            rate,
			Bands:           bands,
			FreeTimeMinutes: gqlscalar.IntValue(input.FreeTimeMinutes),
			MinAmountMinor:  gqlscalar.Int64Ptr(input.MinAmountMinor),
			MaxAmountMinor:  gqlscalar.Int64Ptr(input.MaxAmountMinor),
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
		Description:                  gqlscalar.StringValue(input.Description),
		Classification:               input.Classification,
		CurrencyCode:                 money.DefaultCurrencyCode,
		GuaranteedPeriodMinimumMinor: gqlscalar.Int64Value(input.GuaranteedPeriodMinimumMinor),
		PerDiemRatePerMile:           perDiemRate,
		PerDiemDailyCapMinor:         gqlscalar.Int64Value(input.PerDiemDailyCapMinor),
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
		Description:                  gqlscalar.StringValue(input.Description),
		Classification:               input.Classification,
		CurrencyCode:                 money.DefaultCurrencyCode,
		GuaranteedPeriodMinimumMinor: gqlscalar.Int64Value(input.GuaranteedPeriodMinimumMinor),
		PerDiemRatePerMile:           perDiemRate,
		PerDiemDailyCapMinor:         gqlscalar.Int64Value(input.PerDiemDailyCapMinor),
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
) (*payassignment.WorkerPayAssignment, error) {
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
		splitPercent, err = gqldecimal.DecimalFromString(*input.SplitPercent, "splitPercent")
		if err != nil {
			return nil, err
		}
	}
	overrides := make([]payassignment.RateOverride, 0, len(input.RateOverrides))
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
		rate, rateErr := gqldecimal.DecimalFromString(override.Rate, "rateOverrides.rate")
		if rateErr != nil {
			return nil, rateErr
		}
		overrides = append(overrides, payassignment.RateOverride{
			ComponentID: componentID,
			Rate:        rate,
		})
	}
	return &payassignment.WorkerPayAssignment{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		WorkerID:       workerID,
		PayProfileID:   profileID,
		EffectiveFrom:  int64(input.EffectiveFrom),
		EffectiveTo:    gqlscalar.Int64Ptr(input.EffectiveTo),
		SplitPercent:   splitPercent,
		RateOverrides:  overrides,
		Notes:          gqlscalar.StringValue(input.Notes),
	}, nil
}

func recurringDeductionFromCreateInput(
	input *gqlmodel.CreateRecurringDeductionInput,
	tenantInfo pagination.TenantInfo,
) (*recurringpay.RecurringDeduction, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, err
	}
	payCodeID, err := pulid.MustParse(input.PayCodeID)
	if err != nil {
		return nil, err
	}
	escrowAccountID, err := gqlid.OptionalID(input.EscrowAccountID)
	if err != nil {
		return nil, err
	}
	entity := &recurringpay.RecurringDeduction{
		OrganizationID:      tenantInfo.OrgID,
		BusinessUnitID:      tenantInfo.BuID,
		WorkerID:            workerID,
		PayCodeID:           payCodeID,
		Status:              recurringpay.DeductionStatusActive,
		Frequency:           recurringpay.DeductionFrequencyEverySettlement,
		Description:         input.Description,
		AmountMinor:         int64(input.AmountMinor),
		TotalCapMinor:       gqlscalar.Int64Ptr(input.TotalCapMinor),
		StartDate:           int64(input.StartDate),
		EndDate:             gqlscalar.Int64Ptr(input.EndDate),
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
) (*recurringpay.RecurringDeduction, error) {
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
	escrowAccountID, err := gqlid.OptionalID(input.EscrowAccountID)
	if err != nil {
		return nil, err
	}
	entity := &recurringpay.RecurringDeduction{
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
		TotalCapMinor:  gqlscalar.Int64Ptr(input.TotalCapMinor),
		StartDate:      int64(input.StartDate),
		EndDate:        gqlscalar.Int64Ptr(input.EndDate),
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
) (*recurringpay.RecurringEarning, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, err
	}
	payCodeID, err := pulid.MustParse(input.PayCodeID)
	if err != nil {
		return nil, err
	}
	entity := &recurringpay.RecurringEarning{
		OrganizationID:  tenantInfo.OrgID,
		BusinessUnitID:  tenantInfo.BuID,
		WorkerID:        workerID,
		PayCodeID:       payCodeID,
		Status:          recurringpay.EarningStatusActive,
		Frequency:       recurringpay.EarningFrequencyEverySettlement,
		Description:     input.Description,
		AmountMinor:     int64(input.AmountMinor),
		TotalCapMinor:   gqlscalar.Int64Ptr(input.TotalCapMinor),
		StartDate:       int64(input.StartDate),
		EndDate:         gqlscalar.Int64Ptr(input.EndDate),
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
) (*recurringpay.RecurringEarning, error) {
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
	entity := &recurringpay.RecurringEarning{
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
		TotalCapMinor:  gqlscalar.Int64Ptr(input.TotalCapMinor),
		StartDate:      int64(input.StartDate),
		EndDate:        gqlscalar.Int64Ptr(input.EndDate),
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
) (*payadvance.PayAdvance, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, err
	}
	entity := &payadvance.PayAdvance{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		WorkerID:       workerID,
		Source:         input.Source,
		Reference:      gqlscalar.StringValue(input.Reference),
		IssuedDate:     int64(input.IssuedDate),
		AmountMinor:    int64(input.AmountMinor),
		Notes:          gqlscalar.StringValue(input.Notes),
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
) (*payescrow.EscrowAccount, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, err
	}
	interestRate, err := optionalDecimalFromString(input.AnnualInterestRate, "annualInterestRate")
	if err != nil {
		return nil, err
	}
	entity := &payescrow.EscrowAccount{
		OrganizationID:     tenantInfo.OrgID,
		BusinessUnitID:     tenantInfo.BuID,
		WorkerID:           workerID,
		TargetAmountMinor:  int64(input.TargetAmountMinor),
		AnnualInterestRate: interestRate,
		OpenedDate:         gqlscalar.Int64Value(input.OpenedDate),
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
) (*payescrow.EscrowAccount, error) {
	accountID, err := pulid.MustParse(input.ID)
	if err != nil {
		return nil, err
	}
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, err
	}
	interestRate, err := gqldecimal.DecimalFromString(
		input.AnnualInterestRate,
		"annualInterestRate",
	)
	if err != nil {
		return nil, err
	}
	return &payescrow.EscrowAccount{
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
) (*settlementcontrol.SettlementControl, error) {
	varianceThreshold, err := gqldecimal.DecimalFromString(
		input.VarianceThresholdPct,
		"varianceThresholdPct",
	)
	if err != nil {
		return nil, err
	}
	escrowInterestRate, err := gqldecimal.DecimalFromString(
		input.DefaultEscrowInterestRate,
		"defaultEscrowInterestRate",
	)
	if err != nil {
		return nil, err
	}
	return &settlementcontrol.SettlementControl{
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
