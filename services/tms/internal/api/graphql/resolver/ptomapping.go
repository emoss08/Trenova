package resolver

import (
	"strconv"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/ptoledgerservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func decimalString(value decimal.Decimal) string {
	return value.StringFixed(2)
}

func parseDecimalField(field string, raw string, required bool) (decimal.Decimal, error) {
	if raw == "" {
		if required {
			return decimal.Zero, errortypes.NewValidationError(
				field,
				errortypes.ErrRequired,
				"Value is required",
			)
		}
		return decimal.Zero, nil
	}
	value, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero, errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"Value must be a number",
		)
	}
	return value, nil
}

func parseNullDecimalField(field string, raw *string) (decimal.NullDecimal, error) {
	if raw == nil || *raw == "" {
		return decimal.NullDecimal{}, nil
	}
	value, err := parseDecimalField(field, *raw, true)
	if err != nil {
		return decimal.NullDecimal{}, err
	}
	return decimal.NewNullDecimal(value), nil
}

func ptoPolicyFromInput(
	input *gqlmodel.PTOPolicyInput,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*worker.PTOPolicy, error) {
	floor, err := parseNullDecimalField("negativeFloorDays", input.NegativeFloorDays)
	if err != nil {
		return nil, err
	}

	entity := &worker.PTOPolicy{
		ID:                id,
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		Name:              input.Name,
		Code:              input.Code,
		Description:       stringValue(input.Description),
		Status:            input.Status,
		IsDefault:         input.IsDefault,
		YearBasis:         input.YearBasis,
		CountWeekends:     input.CountWeekends,
		WaitingPeriodDays: int32(input.WaitingPeriodDays), //nolint:gosec // bounded by validation
		RequiresApproval:  input.RequiresApproval,
		EnforceBalance:    input.EnforceBalance,
		AllowNegative:     input.AllowNegative,
		NegativeFloorDays: floor.Decimal,
		Version:           int64(intValue(input.Version)),
		Rules:             make([]*worker.PTOPolicyRule, 0, len(input.Rules)),
	}

	for i, ruleInput := range input.Rules {
		if ruleInput == nil {
			continue
		}
		prefix := "rules[" + strconv.Itoa(i) + "]."
		amount, aErr := parseDecimalField(
			prefix+"accrualAmountDays",
			ruleInput.AccrualAmountDays,
			false,
		)
		if aErr != nil {
			return nil, aErr
		}
		maxBalance, mErr := parseNullDecimalField(prefix+"maxBalanceDays", ruleInput.MaxBalanceDays)
		if mErr != nil {
			return nil, mErr
		}
		carryCap, cErr := parseNullDecimalField(
			prefix+"carryoverCapDays",
			ruleInput.CarryoverCapDays,
		)
		if cErr != nil {
			return nil, cErr
		}
		tiers, tErr := ptoTiersFromInput(prefix, ruleInput.Tiers)
		if tErr != nil {
			return nil, tErr
		}
		onTermination := worker.PTOTerminationForfeit
		if ruleInput.OnTermination != nil {
			onTermination = *ruleInput.OnTermination
		}
		entity.Rules = append(entity.Rules, &worker.PTOPolicyRule{
			OrganizationID:    tenantInfo.OrgID,
			BusinessUnitID:    tenantInfo.BuID,
			PTOPolicyID:       id,
			PTOType:           ruleInput.PTOType,
			AccrualMethod:     ruleInput.AccrualMethod,
			AccrualAmountDays: amount,
			MaxBalanceDays:    maxBalance,
			CarryoverCapDays:  carryCap,
			CarryoverExpiryDays: int32(
				intValue(ruleInput.CarryoverExpiryDays),
			), //nolint:gosec // bounded by validation
			Tiers:         tiers,
			OnTermination: onTermination,
			SortOrder: int32(
				i,
			), //nolint:gosec // rule counts are tiny
		})
	}

	return entity, nil
}

func ptoTiersFromInput(
	prefix string,
	inputs []*gqlmodel.PTOAccrualTierInput,
) ([]worker.PTOAccrualTier, error) {
	tiers := make([]worker.PTOAccrualTier, 0, len(inputs))
	for i, tierInput := range inputs {
		if tierInput == nil {
			continue
		}
		tierPrefix := prefix + "tiers[" + strconv.Itoa(i) + "]."
		amount, err := parseDecimalField(tierPrefix+"accrualAmountDays", tierInput.AccrualAmountDays, true)
		if err != nil {
			return nil, err
		}
		maxBalance, err := parseNullDecimalField(tierPrefix+"maxBalanceDays", tierInput.MaxBalanceDays)
		if err != nil {
			return nil, err
		}
		tiers = append(tiers, worker.PTOAccrualTier{
			MinMonths:         int32(tierInput.MinMonths), //nolint:gosec // bounded by validation
			AccrualAmountDays: amount,
			MaxBalanceDays:    maxBalance,
		})
	}
	return tiers, nil
}

func ptoPolicyCursorConnectionToModel(
	result *pagination.CursorListResult[*worker.PTOPolicy],
) (*gqlmodel.PTOPolicyConnection, error) {
	edges, err := entityCursorEdges(
		result.Items,
		result.CursorSort,
		result,
		func(node *worker.PTOPolicy, cursor string) *gqlmodel.PTOPolicyEdge {
			return &gqlmodel.PTOPolicyEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.PTOPolicyConnection{
		Edges: edges,
		PageInfo: pageInfo(
			result.HasNextPage,
			lastEdgeCursor(edges, func(edge *gqlmodel.PTOPolicyEdge) string { return edge.Cursor }),
		),
		TotalCount: result.TotalCount,
	}, nil
}

func ptoLedgerCursorConnectionToModel(
	result *pagination.CursorListResult[*worker.WorkerPTOLedgerEntry],
) (*gqlmodel.WorkerPTOLedgerConnection, error) {
	edges, err := entityCursorEdges(
		result.Items,
		result.CursorSort,
		result,
		func(node *worker.WorkerPTOLedgerEntry, cursor string) *gqlmodel.WorkerPTOLedgerEdge {
			return &gqlmodel.WorkerPTOLedgerEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.WorkerPTOLedgerConnection{
		Edges: edges,
		PageInfo: pageInfo(
			result.HasNextPage,
			lastEdgeCursor(
				edges,
				func(edge *gqlmodel.WorkerPTOLedgerEdge) string { return edge.Cursor },
			),
		),
		TotalCount: result.TotalCount,
	}, nil
}

func plannedEntriesToPointers(
	entries []ptoledgerservice.PlannedEntry,
) []*ptoledgerservice.PlannedEntry {
	out := make([]*ptoledgerservice.PlannedEntry, 0, len(entries))
	for i := range entries {
		out = append(out, &entries[i])
	}
	return out
}

func ptoPolicyStatusString(status *worker.PTOPolicyStatus) string {
	if status == nil {
		return ""
	}
	return string(*status)
}

func ptoLedgerEntryTypeString(entryType *worker.PTOLedgerEntryType) string {
	if entryType == nil {
		return ""
	}
	return string(*entryType)
}

func openingBalancesFromInput(
	inputs []*gqlmodel.OpeningPTOBalanceInput,
) ([]ptoledgerservice.OpeningBalance, error) {
	balances := make([]ptoledgerservice.OpeningBalance, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		days, err := parseDecimalField("openingBalances", input.Days, true)
		if err != nil {
			return nil, err
		}
		balances = append(
			balances,
			ptoledgerservice.OpeningBalance{PTOType: input.PTOType, Days: days},
		)
	}
	return balances, nil
}
