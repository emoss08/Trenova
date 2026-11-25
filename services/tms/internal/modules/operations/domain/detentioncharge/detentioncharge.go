package detentioncharge

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func Reconcile(entity *shipment.Shipment, occurrences []*detention.DetentionOccurrence) {
	groups := groupChargeable(occurrences)

	rest := entity.AdditionalCharges
	generated := make([]*shipment.AdditionalCharge, 0, len(groups))

	for i := range groups {
		var existing *shipment.AdditionalCharge
		existing, rest = Take(rest, groups[i].accessorialID)
		generated = append(generated, groupCharge(entity, &groups[i], existing))
	}

	kept := make([]*shipment.AdditionalCharge, 0, len(rest)+len(generated))
	for _, charge := range rest {
		if charge == nil || charge.Owner() == shipmentvocab.SystemOwnerDetention {
			continue
		}
		kept = append(kept, charge)
	}

	entity.AdditionalCharges = append(kept, generated...)
}

type Change struct {
	AccessorialChargeID pulid.ID
	Before              decimal.Decimal
	After               decimal.Decimal
}

func (c Change) Removed() bool { return c.After.IsZero() }

func Settle(entity *shipment.Shipment, occurrences []*detention.DetentionOccurrence) []Change {
	if entity == nil {
		return nil
	}

	governed := governedAccessorials(occurrences)
	groups := groupChargeable(occurrences)
	byAccessorial := make(map[pulid.ID]*chargeGroup, len(groups))
	for i := range groups {
		byAccessorial[groups[i].accessorialID] = &groups[i]
	}

	changes := make([]Change, 0, 1)
	kept := make([]*shipment.AdditionalCharge, 0, len(entity.AdditionalCharges))
	for _, charge := range entity.AdditionalCharges {
		if charge == nil || charge.Owner() != shipmentvocab.SystemOwnerDetention {
			kept = append(kept, charge)
			continue
		}
		if _, ok := governed[charge.AccessorialChargeID]; !ok {
			kept = append(kept, charge)
			continue
		}

		group := byAccessorial[charge.AccessorialChargeID]
		amount := decimal.Zero
		var occurrenceIDs []pulid.ID
		if group != nil {
			amount = group.amount
			occurrenceIDs = group.occurrenceIDs
			delete(byAccessorial, charge.AccessorialChargeID)
		}

		before := chargeAmount(charge)
		if !before.Equal(amount) {
			changes = append(changes, Change{
				AccessorialChargeID: charge.AccessorialChargeID,
				Before:              before,
				After:               amount,
			})
		}
		if amount.IsZero() {
			continue
		}

		charge.Method = accessorialcharge.MethodFlat
		charge.Amount = amount
		charge.Unit = 1
		charge.DetentionOccurrenceIDs = occurrenceIDs
		kept = append(kept, charge)
	}

	entity.AdditionalCharges = kept

	return changes
}

func Take(
	charges []*shipment.AdditionalCharge,
	accessorialID pulid.ID,
) (*shipment.AdditionalCharge, []*shipment.AdditionalCharge) {
	var found *shipment.AdditionalCharge
	rest := make([]*shipment.AdditionalCharge, 0, len(charges))

	for _, charge := range charges {
		if charge == nil {
			continue
		}

		if !isChargeFor(charge, accessorialID) {
			rest = append(rest, charge)
			continue
		}

		if found == nil || prefer(charge, found) {
			found = charge
		}
	}

	return found, rest
}

func chargeAmount(charge *shipment.AdditionalCharge) decimal.Decimal {
	if charge.Method == accessorialcharge.MethodFlat || charge.Unit <= 1 {
		return charge.Amount
	}

	return charge.Amount.Mul(decimal.NewFromInt(int64(charge.Unit)))
}

func isChargeFor(charge *shipment.AdditionalCharge, accessorialID pulid.ID) bool {
	if charge == nil || !charge.IsSystemGenerated || charge.AccessorialChargeID != accessorialID {
		return false
	}

	owner := charge.Owner()

	return owner == shipmentvocab.SystemOwnerDetention || owner == shipmentvocab.SystemOwnerNone
}

func prefer(candidate, current *shipment.AdditionalCharge) bool {
	switch {
	case candidate.ID.IsNil() != current.ID.IsNil():
		return current.ID.IsNil()
	case candidate.CreatedAt != current.CreatedAt:
		return candidate.CreatedAt < current.CreatedAt
	default:
		return candidate.ID.String() < current.ID.String()
	}
}

func governedAccessorials(occurrences []*detention.DetentionOccurrence) map[pulid.ID]struct{} {
	governed := make(map[pulid.ID]struct{}, 1)
	for _, occurrence := range occurrences {
		if occurrence == nil || occurrence.PolicySnapshot == nil {
			continue
		}
		governed[occurrence.PolicySnapshot.AccessorialChargeID] = struct{}{}
	}

	return governed
}

type chargeGroup struct {
	accessorialID pulid.ID
	amount        decimal.Decimal
	occurrenceIDs []pulid.ID
}

func groupChargeable(occurrences []*detention.DetentionOccurrence) []chargeGroup {
	groups := make([]chargeGroup, 0, 1)
	index := make(map[pulid.ID]int, 1)

	for _, occurrence := range occurrences {
		if !occurrence.Chargeable() {
			continue
		}

		accessorialID := occurrence.PolicySnapshot.AccessorialChargeID

		i, ok := index[accessorialID]
		if !ok {
			i = len(groups)
			index[accessorialID] = i
			groups = append(groups, chargeGroup{
				accessorialID: accessorialID,
				amount:        decimal.Zero,
				occurrenceIDs: make([]pulid.ID, 0, 2),
			})
		}

		groups[i].amount = groups[i].amount.Add(occurrence.BillableAmount)
		groups[i].occurrenceIDs = append(groups[i].occurrenceIDs, occurrence.ID)
	}

	return groups
}

func groupCharge(
	entity *shipment.Shipment,
	group *chargeGroup,
	existing *shipment.AdditionalCharge,
) *shipment.AdditionalCharge {
	charge := existing
	if charge == nil {
		charge = &shipment.AdditionalCharge{}
	}

	charge.OrganizationID = entity.OrganizationID
	charge.BusinessUnitID = entity.BusinessUnitID
	charge.ShipmentID = entity.ID
	charge.IsSystemGenerated = true
	charge.IsDetention = true
	charge.AccessorialChargeID = group.accessorialID
	charge.Method = accessorialcharge.MethodFlat
	charge.Amount = group.amount
	charge.Unit = 1
	charge.DetentionOccurrenceIDs = group.occurrenceIDs

	return charge
}
