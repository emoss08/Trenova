package ptopolicyresolver

import (
	"strconv"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func ptoPolicyFromInput(
	input *gqlmodel.PTOPolicyInput,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*pto.PTOPolicy, error) {
	floor, err := gqldecimal.ParseNullDecimalField("negativeFloorDays", input.NegativeFloorDays)
	if err != nil {
		return nil, err
	}

	entity := &pto.PTOPolicy{
		ID:                id,
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		Name:              input.Name,
		Code:              input.Code,
		Description:       gqlscalar.StringValue(input.Description),
		Status:            input.Status,
		IsDefault:         input.IsDefault,
		YearBasis:         input.YearBasis,
		CountWeekends:     input.CountWeekends,
		WaitingPeriodDays: int32(input.WaitingPeriodDays), //nolint:gosec // bounded by validation
		RequiresApproval:  input.RequiresApproval,
		EnforceBalance:    input.EnforceBalance,
		AllowNegative:     input.AllowNegative,
		NegativeFloorDays: floor.Decimal,
		Version:           int64(gqlscalar.IntValue(input.Version)),
		Rules:             make([]*pto.PTOPolicyRule, 0, len(input.Rules)),
	}

	for i, ruleInput := range input.Rules {
		if ruleInput == nil {
			continue
		}
		prefix := "rules[" + strconv.Itoa(i) + "]."
		amount, aErr := gqldecimal.ParseDecimalField(
			prefix+"accrualAmountDays",
			ruleInput.AccrualAmountDays,
			false,
		)
		if aErr != nil {
			return nil, aErr
		}
		maxBalance, mErr := gqldecimal.ParseNullDecimalField(
			prefix+"maxBalanceDays",
			ruleInput.MaxBalanceDays,
		)
		if mErr != nil {
			return nil, mErr
		}
		carryCap, cErr := gqldecimal.ParseNullDecimalField(
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
		onTermination := pto.PTOTerminationForfeit
		if ruleInput.OnTermination != nil {
			onTermination = *ruleInput.OnTermination
		}
		entity.Rules = append(entity.Rules, &pto.PTOPolicyRule{
			OrganizationID:    tenantInfo.OrgID,
			BusinessUnitID:    tenantInfo.BuID,
			PTOPolicyID:       id,
			PTOType:           ruleInput.PTOType,
			AccrualMethod:     ruleInput.AccrualMethod,
			AccrualAmountDays: amount,
			MaxBalanceDays:    maxBalance,
			CarryoverCapDays:  carryCap,
			CarryoverExpiryDays: int32(
				gqlscalar.IntValue(ruleInput.CarryoverExpiryDays),
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
) ([]pto.PTOAccrualTier, error) {
	tiers := make([]pto.PTOAccrualTier, 0, len(inputs))
	for i, tierInput := range inputs {
		if tierInput == nil {
			continue
		}
		tierPrefix := prefix + "tiers[" + strconv.Itoa(i) + "]."
		amount, err := gqldecimal.ParseDecimalField(
			tierPrefix+"accrualAmountDays",
			tierInput.AccrualAmountDays,
			true,
		)
		if err != nil {
			return nil, err
		}
		maxBalance, err := gqldecimal.ParseNullDecimalField(
			tierPrefix+"maxBalanceDays",
			tierInput.MaxBalanceDays,
		)
		if err != nil {
			return nil, err
		}
		tiers = append(tiers, pto.PTOAccrualTier{
			MinMonths:         int32(tierInput.MinMonths), //nolint:gosec // bounded by validation
			AccrualAmountDays: amount,
			MaxBalanceDays:    maxBalance,
		})
	}
	return tiers, nil
}

func ptoPolicyCursorConnectionToModel(
	result *pagination.CursorListResult[*pto.PTOPolicy],
) (*gqlmodel.PTOPolicyConnection, error) {
	edges, err := gqlconnection.EntityCursorEdges(
		result.Items,
		result,
		func(node *pto.PTOPolicy, cursor string) *gqlmodel.PTOPolicyEdge {
			return &gqlmodel.PTOPolicyEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.PTOPolicyConnection{
		Edges: edges,
		PageInfo: gqlconnection.PageInfo(
			result.HasNextPage,
			gqlconnection.LastEdgeCursor(
				edges,
				func(edge *gqlmodel.PTOPolicyEdge) string { return edge.Cursor },
			),
		),
		TotalCount:      result.TotalCount,
		TotalCountExact: result.TotalCountExact,
	}, nil
}

func ptoLedgerCursorConnectionToModel(
	result *pagination.CursorListResult[*pto.WorkerPTOLedgerEntry],
) (*gqlmodel.WorkerPTOLedgerConnection, error) {
	edges, err := gqlconnection.EntityCursorEdges(
		result.Items,
		result,
		func(node *pto.WorkerPTOLedgerEntry, cursor string) *gqlmodel.WorkerPTOLedgerEdge {
			return &gqlmodel.WorkerPTOLedgerEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.WorkerPTOLedgerConnection{
		Edges: edges,
		PageInfo: gqlconnection.PageInfo(
			result.HasNextPage,
			gqlconnection.LastEdgeCursor(
				edges,
				func(edge *gqlmodel.WorkerPTOLedgerEdge) string { return edge.Cursor },
			),
		),
		TotalCount:      result.TotalCount,
		TotalCountExact: result.TotalCountExact,
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

func ptoPolicyStatusString(status *pto.PTOPolicyStatus) string {
	if status == nil {
		return ""
	}
	return string(*status)
}

func ptoLedgerEntryTypeString(entryType *pto.PTOLedgerEntryType) string {
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
		days, err := gqldecimal.ParseDecimalField("openingBalances", input.Days, true)
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
