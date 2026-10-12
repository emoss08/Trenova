package shipment

import (
	"sort"

	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// percentPlaces matches the precision the percent column stores.
const percentPlaces = 6

// StaleAmountSplit is an amount split whose rows no longer add up to the charge
// they divide, which happens when the charge itself changes after it was split.
type StaleAmountSplit struct {
	Kind               ChargeAllocationKind
	AdditionalChargeID pulid.ID
	// ChargeIndex is the charge's position on the shipment, or -1 for freight.
	ChargeIndex    int
	AllocatedTotal decimal.Decimal
	ChargeTotal    decimal.Decimal
}

type chargeSplitGroup struct {
	kind        ChargeAllocationKind
	chargeID    pulid.ID
	chargeIndex int
	total       decimal.Decimal
	rows        []*ChargeAllocation
}

// chargeSplitGroups pairs each freight or accessorial charge on the shipment with
// the allocations that divide it. Charges nobody split are left out.
func chargeSplitGroups(shp *Shipment, allocations []*ChargeAllocation) []*chargeSplitGroup {
	if shp == nil {
		return nil
	}
	freight := shp.FreightChargeAmount.Decimal

	byCharge := make(map[pulid.ID][]*ChargeAllocation, len(allocations))
	freightRows := make([]*ChargeAllocation, 0, len(allocations))
	for _, allocation := range allocations {
		if allocation == nil {
			continue
		}
		switch allocation.ChargeKind {
		case ChargeAllocationKindFreight:
			freightRows = append(freightRows, allocation)
		case ChargeAllocationKindAccessorial:
			if target := allocation.TargetID(); target.IsNotNil() {
				byCharge[target] = append(byCharge[target], allocation)
			}
		case ChargeAllocationKindOrderCharge:
		}
	}

	groups := make([]*chargeSplitGroup, 0, len(byCharge)+1)
	if len(freightRows) > 0 {
		groups = append(groups, &chargeSplitGroup{
			kind:        ChargeAllocationKindFreight,
			chargeIndex: -1,
			total:       freight,
			rows:        freightRows,
		})
	}
	for i, charge := range shp.AdditionalCharges {
		if charge == nil {
			continue
		}
		rows := byCharge[charge.ID]
		if len(rows) == 0 {
			continue
		}
		groups = append(groups, &chargeSplitGroup{
			kind:        ChargeAllocationKindAccessorial,
			chargeID:    charge.ID,
			chargeIndex: i,
			total:       charge.Total(freight),
			rows:        rows,
		})
	}

	return groups
}

func (g *chargeSplitGroup) amountTotal() (decimal.Decimal, bool) {
	sum := decimal.Zero
	for _, row := range g.rows {
		if row.Method != ChargeAllocationMethodAmount {
			return decimal.Zero, false
		}
		sum = sum.Add(row.Amount.Decimal)
	}

	return sum, true
}

func (g *chargeSplitGroup) stale() (decimal.Decimal, bool) {
	sum, allAmount := g.amountTotal()
	if !allAmount {
		return decimal.Zero, false
	}

	return sum, !decimalutils.SumEquals([]decimal.Decimal{sum}, g.total, SharePlaces)
}

// FindStaleAmountSplits lists every charge split by amount whose rows no longer
// add up to the charge. Percent splits follow the charge and never go stale.
func FindStaleAmountSplits(shp *Shipment, allocations []*ChargeAllocation) []StaleAmountSplit {
	groups := chargeSplitGroups(shp, allocations)
	stale := make([]StaleAmountSplit, 0, len(groups))
	for _, group := range groups {
		sum, isStale := group.stale()
		if !isStale {
			continue
		}
		stale = append(stale, StaleAmountSplit{
			Kind:               group.kind,
			AdditionalChargeID: group.chargeID,
			ChargeIndex:        group.chargeIndex,
			AllocatedTotal:     sum,
			ChargeTotal:        group.total,
		})
	}

	return stale
}

// ConvertAmountSplitsToPercent turns every stale amount split into a percent
// split that keeps each payer's proportion of what they had agreed to, so a
// changed charge is divided the same way it was before. Rows are rewritten in
// place; the last row by sequence absorbs rounding so the split totals exactly
// 100. It returns how many charges were converted.
func ConvertAmountSplitsToPercent(shp *Shipment, allocations []*ChargeAllocation) int {
	converted := 0
	for _, group := range chargeSplitGroups(shp, allocations) {
		sum, isStale := group.stale()
		if !isStale || !sum.IsPositive() {
			continue
		}

		rows := make([]*ChargeAllocation, len(group.rows))
		copy(rows, group.rows)
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].Sequence != rows[j].Sequence {
				return rows[i].Sequence < rows[j].Sequence
			}
			return rows[i].ID < rows[j].ID
		})

		assigned := decimal.Zero
		for i, row := range rows {
			percent := decimalutils.Percent100.Sub(assigned)
			if i < len(rows)-1 {
				percent = row.Amount.Decimal.Mul(decimalutils.Percent100).
					Div(sum).
					Round(percentPlaces)
				assigned = assigned.Add(percent)
			}
			row.Method = ChargeAllocationMethodPercent
			row.Percent = decimal.NewNullDecimal(percent)
			row.Amount = decimal.NullDecimal{}
		}
		converted++
	}

	return converted
}
