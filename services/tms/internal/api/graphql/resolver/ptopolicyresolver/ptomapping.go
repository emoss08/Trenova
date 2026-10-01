package ptopolicyresolver

import (
	"strconv"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/ptoledgerservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func ptoPolicyFromInput(
	input *gqlmodel.PTOPolicyInput,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*worker.PTOPolicy, error) {
	floor, err := base.ParseNullDecimalField("negativeFloorDays", input.NegativeFloorDays)
	if err != nil {
		return nil, err
	}

	entity := &worker.PTOPolicy{
		ID:                id,
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		Name:              input.Name,
		Code:              input.Code,
		Description:       base.StringValue(input.Description),
		Status:            input.Status,
		IsDefault:         input.IsDefault,
		YearBasis:         input.YearBasis,
		CountWeekends:     input.CountWeekends,
		WaitingPeriodDays: int32(input.WaitingPeriodDays), //nolint:gosec // bounded by validation
		RequiresApproval:  input.RequiresApproval,
		EnforceBalance:    input.EnforceBalance,
		AllowNegative:     input.AllowNegative,
		NegativeFloorDays: floor.Decimal,
		Version:           int64(base.IntValue(input.Version)),
		Rules:             make([]*worker.PTOPolicyRule, 0, len(input.Rules)),
	}

	for i, ruleInput := range input.Rules {
		if ruleInput == nil {
			continue
		}
		prefix := "rules[" + strconv.Itoa(i) + "]."
		amount, aErr := base.ParseDecimalField(
			prefix+"accrualAmountDays",
			ruleInput.AccrualAmountDays,
			false,
		)
		if aErr != nil {
			return nil, aErr
		}
		maxBalance, mErr := base.ParseNullDecimalField(
			prefix+"maxBalanceDays",
			ruleInput.MaxBalanceDays,
		)
		if mErr != nil {
			return nil, mErr
		}
		carryCap, cErr := base.ParseNullDecimalField(
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
				base.IntValue(ruleInput.CarryoverExpiryDays),
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
		amount, err := base.ParseDecimalField(
			tierPrefix+"accrualAmountDays",
			tierInput.AccrualAmountDays,
			true,
		)
		if err != nil {
			return nil, err
		}
		maxBalance, err := base.ParseNullDecimalField(
			tierPrefix+"maxBalanceDays",
			tierInput.MaxBalanceDays,
		)
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
	edges, err := base.EntityCursorEdges(
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
		PageInfo: base.PageInfo(
			result.HasNextPage,
			base.LastEdgeCursor(
				edges,
				func(edge *gqlmodel.PTOPolicyEdge) string { return edge.Cursor },
			),
		),
		TotalCount: result.TotalCount,
	}, nil
}

func ptoLedgerCursorConnectionToModel(
	result *pagination.CursorListResult[*worker.WorkerPTOLedgerEntry],
) (*gqlmodel.WorkerPTOLedgerConnection, error) {
	edges, err := base.EntityCursorEdges(
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
		PageInfo: base.PageInfo(
			result.HasNextPage,
			base.LastEdgeCursor(
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
		days, err := base.ParseDecimalField("openingBalances", input.Days, true)
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
