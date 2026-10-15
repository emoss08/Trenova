package driversettlement

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/settlementload"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
	"github.com/shopspring/decimal"
)

type LineGroup struct {
	Category             LineCategory
	ComponentKind        driverpay.ComponentKind
	Method               driverpay.CalcMethod
	PayCodeID            *pulid.ID
	Description          string
	Rate                 decimal.NullDecimal
	Quantity             decimal.Decimal
	AmountMinor          int64
	LineCount            int
	ShipmentCount        int
	AdvanceID            *pulid.ID
	EscrowAccountID      *pulid.ID
	RecurringDeductionID *pulid.ID
	RecurringEarningID   *pulid.ID
	Manual               bool
}

type lineGroupKey struct {
	category             LineCategory
	componentKind        driverpay.ComponentKind
	method               driverpay.CalcMethod
	payCodeID            pulid.ID
	advanceID            pulid.ID
	escrowAccountID      pulid.ID
	recurringDeductionID pulid.ID
	recurringEarningID   pulid.ID
	manual               bool
	description          string
}

type lineGroupTally struct {
	group       LineGroup
	first       int
	shipments   map[pulid.ID]struct{}
	description string
	sameText    bool
	rate        decimal.Decimal
	sameRate    bool
}

var lineCategoryRank = map[LineCategory]int{
	LineCategoryEarning:            0,
	LineCategoryGuaranteeTopUp:     1,
	LineCategoryReimbursement:      2,
	LineCategoryCarryForward:       3,
	LineCategoryAdjustment:         4,
	LineCategoryDeduction:          5,
	LineCategoryAdvanceRecovery:    6,
	LineCategoryEscrowContribution: 7,
}

func (l *SettlementLine) IsManual() bool {
	return l.Category == LineCategoryAdjustment &&
		typeutils.DerefID(l.PayEventID).IsNil() &&
		typeutils.DerefID(l.RecurringEarningID).IsNil() &&
		typeutils.DerefID(l.RecurringDeductionID).IsNil()
}

func LineGroups(lines []*SettlementLine) []LineGroup {
	tallies := make([]*lineGroupTally, 0, len(lines))
	at := make(map[lineGroupKey]*lineGroupTally, len(lines))
	for idx, line := range lines {
		if line == nil {
			continue
		}
		key := groupKeyOf(line)
		tally, seen := at[key]
		if !seen {
			tally = &lineGroupTally{
				group: LineGroup{
					Category:             line.Category,
					ComponentKind:        line.ComponentKind,
					Method:               line.Method,
					PayCodeID:            line.PayCodeID,
					AdvanceID:            line.AdvanceID,
					EscrowAccountID:      line.EscrowAccountID,
					RecurringDeductionID: line.RecurringDeductionID,
					RecurringEarningID:   line.RecurringEarningID,
					Manual:               key.manual,
				},
				first:       idx,
				shipments:   make(map[pulid.ID]struct{}),
				description: line.Description,
				sameText:    true,
				rate:        line.Rate,
				sameRate:    true,
			}
			at[key] = tally
			tallies = append(tallies, tally)
		}
		tally.add(line)
	}

	slices.SortStableFunc(tallies, func(a, b *lineGroupTally) int {
		return cmp.Or(
			cmp.Compare(lineCategoryRank[a.group.Category], lineCategoryRank[b.group.Category]),
			cmp.Compare(a.first, b.first),
		)
	})

	groups := make([]LineGroup, 0, len(tallies))
	for _, tally := range tallies {
		groups = append(groups, tally.finish())
	}
	return groups
}

func LoadLines(lines []*SettlementLine) []settlementload.Line {
	out := make([]settlementload.Line, 0, len(lines))
	for _, line := range lines {
		if line == nil {
			continue
		}
		out = append(out, settlementload.Line{
			ShipmentID:  line.ShipmentID,
			MoveID:      line.MoveID,
			ProNumber:   line.ProNumber,
			AmountMinor: line.AmountMinor,
		})
	}
	return out
}

func groupKeyOf(line *SettlementLine) lineGroupKey {
	key := lineGroupKey{
		category:             line.Category,
		componentKind:        line.ComponentKind,
		method:               line.Method,
		payCodeID:            typeutils.DerefID(line.PayCodeID),
		advanceID:            typeutils.DerefID(line.AdvanceID),
		escrowAccountID:      typeutils.DerefID(line.EscrowAccountID),
		recurringDeductionID: typeutils.DerefID(line.RecurringDeductionID),
		recurringEarningID:   typeutils.DerefID(line.RecurringEarningID),
		manual:               line.IsManual(),
	}
	if key.manual || describedOnly(&key) {
		key.description = line.Description
	}
	return key
}

func describedOnly(key *lineGroupKey) bool {
	return key.componentKind == "" &&
		key.payCodeID.IsNil() &&
		key.advanceID.IsNil() &&
		key.escrowAccountID.IsNil() &&
		key.recurringDeductionID.IsNil() &&
		key.recurringEarningID.IsNil()
}

func (t *lineGroupTally) add(line *SettlementLine) {
	t.group.AmountMinor += line.AmountMinor
	t.group.Quantity = t.group.Quantity.Add(line.Quantity)
	t.group.LineCount++
	if line.Description != t.description {
		t.sameText = false
	}
	if !line.Rate.Equal(t.rate) {
		t.sameRate = false
	}
	if id := typeutils.DerefID(line.ShipmentID); !id.IsNil() {
		t.shipments[id] = struct{}{}
	}
}

func (t *lineGroupTally) finish() LineGroup {
	group := t.group
	group.ShipmentCount = len(t.shipments)
	if t.sameText {
		group.Description = t.description
	}
	if t.sameRate && !t.rate.IsZero() {
		group.Rate = decimal.NewNullDecimal(t.rate)
	}
	return group
}
