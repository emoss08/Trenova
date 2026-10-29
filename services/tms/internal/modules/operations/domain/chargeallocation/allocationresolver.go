package chargeallocation

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const SharePlaces = 2

var errSplitShape = errors.New("the rules module answered a split with the wrong shape")

// AllocatedCharge is one payer's slice of one charge.
type AllocatedCharge struct {
	Kind         shipmentvocab.ChargeAllocationKind
	AllocationID pulid.ID
	// Method is how the charge was split; empty when it was billed whole.
	Method        shipmentvocab.ChargeAllocationMethod
	Charge        *shipment.AdditionalCharge
	OrderChargeID pulid.ID
	Description   string
	ChargeTotal   decimal.Decimal
	Percent       decimal.NullDecimal
	Amount        decimal.Decimal
	Partial       bool
}

// PayerShare is everything one payer is billed for on a shipment.
type PayerShare struct {
	PayerID           pulid.ID
	Charges           []AllocatedCharge
	FreightAmount     decimal.Decimal
	AccessorialAmount decimal.Decimal
	TotalAmount       decimal.Decimal
}

// ShareResolution is a shipment's charges divided among its payers. The default
// payer always leads, so a shipment nobody split still resolves to one share.
type ShareResolution struct {
	DefaultPayerID pulid.ID
	Shares         []*PayerShare
	IsSplit        bool
}

func (r *ShareResolution) ShareFor(payerID pulid.ID) *PayerShare {
	if r == nil {
		return nil
	}
	for _, share := range r.Shares {
		if share.PayerID == payerID {
			return share
		}
	}

	return nil
}

func (r *ShareResolution) PayerIDs() []pulid.ID {
	if r == nil {
		return nil
	}
	ids := make([]pulid.ID, 0, len(r.Shares))
	for _, share := range r.Shares {
		ids = append(ids, share.PayerID)
	}

	return ids
}

func (r *ShareResolution) Primary() *PayerShare {
	if r == nil || len(r.Shares) == 0 {
		return nil
	}

	return r.Shares[0]
}

type chargeGroup struct {
	base   AllocatedCharge
	field  string
	charge rules.AllocationCharge
}

func allocationRows(allocations []*shipment.ChargeAllocation) []rules.AllocationRow {
	rows := make([]rules.AllocationRow, 0, len(allocations))
	for _, allocation := range allocations {
		rows = append(rows, rules.AllocationRow{
			ID:               allocation.ID.String(),
			BillToCustomerID: allocation.BillToCustomerID.String(),
			Method:           string(allocation.Method),
			Percent:          allocation.Percent,
			Amount:           allocation.Amount,
			Sequence:         int64(allocation.Sequence),
		})
	}

	return rows
}

func accessorialCharge(
	charge *shipment.AdditionalCharge,
	allocations []*shipment.ChargeAllocation,
) rules.AllocationCharge {
	if charge == nil {
		return rules.AllocationCharge{Kind: rules.AllocationKindAccessorial}
	}

	return rules.AllocationCharge{
		Kind:        rules.AllocationKindAccessorial,
		Method:      string(charge.Method),
		Amount:      decimal.NewNullDecimal(charge.Amount),
		Unit:        int64(charge.Unit),
		Allocations: allocationRows(allocations),
	}
}

func totalCharge(
	kind string,
	total decimal.Decimal,
	allocations []*shipment.ChargeAllocation,
) rules.AllocationCharge {
	return rules.AllocationCharge{
		Kind:        kind,
		Total:       decimal.NewNullDecimal(total),
		Allocations: allocationRows(allocations),
	}
}

func groupAllocations(
	allocations []*shipment.ChargeAllocation,
) (freight []*shipment.ChargeAllocation, byCharge map[pulid.ID][]*shipment.ChargeAllocation) {
	freight = make([]*shipment.ChargeAllocation, 0, len(allocations))
	byCharge = make(map[pulid.ID][]*shipment.ChargeAllocation, len(allocations))
	for _, allocation := range allocations {
		if allocation == nil {
			continue
		}
		switch allocation.ChargeKind {
		case shipmentvocab.ChargeAllocationKindFreight:
			freight = append(freight, allocation)
		case shipmentvocab.ChargeAllocationKindAccessorial:
			if target := allocation.TargetID(); target.IsNotNil() {
				byCharge[target] = append(byCharge[target], allocation)
			}
		case shipmentvocab.ChargeAllocationKindOrderCharge:
		}
	}

	return freight, byCharge
}

func resolveGroups(
	ctx context.Context,
	defaultPayer pulid.ID,
	base decimal.Decimal,
	groups []chargeGroup,
) (*ShareResolution, error) {
	req := &rules.AllocationSplitRequest{
		DefaultPayerID: defaultPayer.String(),
		Base:           base,
		Charges:        make([]rules.AllocationCharge, 0, len(groups)),
	}
	for i := range groups {
		req.Charges = append(req.Charges, groups[i].charge)
	}

	split, err := rules.SplitCharges(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("split charges between payers: %w", err)
	}
	if len(split.Charges) != len(groups) {
		return nil, fmt.Errorf(
			"%w: %d charges came back for %d sent",
			errSplitShape,
			len(split.Charges),
			len(groups),
		)
	}
	if index, message := split.FirstError(); message != nil {
		return nil, message.ValidationError(groups[index].field, errortypes.ErrInvalid)
	}

	resolution := &ShareResolution{
		DefaultPayerID: defaultPayer,
		Shares:         make([]*PayerShare, 0, len(split.Payers)),
		IsSplit:        split.IsSplit,
	}
	byPayer := make(map[pulid.ID]*PayerShare, len(split.Payers))
	for _, payer := range split.Payers {
		share := &PayerShare{
			PayerID:           pulid.ID(payer.PayerID),
			FreightAmount:     payer.FreightAmount,
			AccessorialAmount: payer.AccessorialAmount,
			TotalAmount:       payer.TotalAmount,
		}
		byPayer[share.PayerID] = share
		resolution.Shares = append(resolution.Shares, share)
	}

	for i, charge := range split.Charges {
		for _, share := range charge.Shares {
			payer, ok := byPayer[pulid.ID(share.BillToCustomerID)]
			if !ok {
				return nil, fmt.Errorf(
					"%w: a share went to %s, who is not among the payers",
					errSplitShape,
					share.BillToCustomerID,
				)
			}
			allocated := groups[i].base
			allocated.ChargeTotal = charge.Total
			allocated.AllocationID = pulid.ID(share.AllocationID)
			allocated.Method = shipmentvocab.ChargeAllocationMethod(share.Method)
			allocated.Percent = share.Percent
			allocated.Amount = share.Amount
			allocated.Partial = share.Partial
			payer.Charges = append(payer.Charges, allocated)
		}
	}

	return resolution, nil
}

func ResolveShares(
	ctx context.Context,
	shp *shipment.Shipment,
	allocations []*shipment.ChargeAllocation,
) (*ShareResolution, error) {
	if shp == nil {
		return nil, errortypes.NewValidationError(
			"shipment",
			errortypes.ErrRequired,
			"Shipment is required",
		)
	}

	freight := shp.FreightChargeAmount.Decimal
	freightAllocations, byCharge := groupAllocations(allocations)

	groups := make([]chargeGroup, 0, len(shp.AdditionalCharges)+1)
	groups = append(groups, chargeGroup{
		base:   AllocatedCharge{Kind: shipmentvocab.ChargeAllocationKindFreight},
		field:  "freightAllocations",
		charge: totalCharge(rules.AllocationKindFreight, freight, freightAllocations),
	})
	for i, charge := range shp.AdditionalCharges {
		if charge == nil {
			continue
		}
		groups = append(groups, chargeGroup{
			base: AllocatedCharge{
				Kind:   shipmentvocab.ChargeAllocationKindAccessorial,
				Charge: charge,
			},
			field:  "additionalCharges[" + strconv.Itoa(i) + "].allocations",
			charge: accessorialCharge(charge, byCharge[charge.ID]),
		})
	}

	return resolveGroups(ctx, shp.PayerID(), freight, groups)
}

func BillsWholeChargeToPayer(
	ctx context.Context,
	shp *shipment.Shipment,
	charge *shipment.AdditionalCharge,
	rows []*shipment.ChargeAllocation,
) (bool, error) {
	if shp == nil {
		return false, nil
	}

	freight := shp.FreightChargeAmount.Decimal
	request := totalCharge(rules.AllocationKindFreight, freight, rows)
	if charge != nil {
		request = accessorialCharge(charge, rows)
	}

	split, err := rules.SplitCharges(ctx, &rules.AllocationSplitRequest{
		DefaultPayerID: shp.PayerID().String(),
		Base:           freight,
		Charges:        []rules.AllocationCharge{request},
	})
	if err != nil {
		return false, fmt.Errorf("check whether a split bills the whole charge: %w", err)
	}
	if len(split.Charges) != 1 || split.Charges[0].Error != nil {
		return false, nil
	}

	shares := split.Charges[0].Shares
	return len(shares) == 1 && !shares[0].Partial &&
		pulid.ID(shares[0].BillToCustomerID) == shp.PayerID(), nil
}

type OrderChargeRef struct {
	ID          pulid.ID
	Description string
	Amount      decimal.Decimal
}

func ResolveOrderChargeShares(
	ctx context.Context,
	charges []OrderChargeRef,
	allocations []*shipment.ChargeAllocation,
	defaultPayerID pulid.ID,
) (*ShareResolution, error) {
	byCharge := make(map[pulid.ID][]*shipment.ChargeAllocation, len(allocations))
	for _, allocation := range allocations {
		if allocation == nil ||
			allocation.ChargeKind != shipmentvocab.ChargeAllocationKindOrderCharge {
			continue
		}
		if target := allocation.TargetID(); target.IsNotNil() {
			byCharge[target] = append(byCharge[target], allocation)
		}
	}

	groups := make([]chargeGroup, 0, len(charges))
	for i, charge := range charges {
		groups = append(groups, chargeGroup{
			base: AllocatedCharge{
				Kind:          shipmentvocab.ChargeAllocationKindOrderCharge,
				OrderChargeID: charge.ID,
				Description:   charge.Description,
			},
			field: "charges[" + strconv.Itoa(i) + "].allocations",
			charge: totalCharge(
				rules.AllocationKindOrderCharge,
				charge.Amount,
				byCharge[charge.ID],
			),
		})
	}

	return resolveGroups(ctx, defaultPayerID, decimal.Zero, groups)
}

// ValidateAllocationsParams carries what a validation pass needs to judge a set
// of allocations without touching storage.
type ValidateAllocationsParams struct {
	TenantOrgID pulid.ID
	TenantBuID  pulid.ID
	Allocations []*shipment.ChargeAllocation
	Customers   map[pulid.ID]*customer.Customer
	Locked      map[pulid.ID]struct{}
	Field       string
}

// ValidateAllocations checks each row on its own and the payer it names: a
// payer must exist, be active and belong to the tenant, and a row already
// carried on a posted invoice cannot change hands.
func ValidateAllocations(p *ValidateAllocationsParams) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if p == nil {
		return multiErr
	}
	prefix := p.Field
	if prefix == "" {
		prefix = "chargeAllocations"
	}

	for i, allocation := range p.Allocations {
		if allocation == nil {
			continue
		}
		field := prefix + "[" + strconv.Itoa(i) + "]"

		rowErr := errortypes.NewMultiError()
		allocation.Validate(rowErr)
		for _, e := range rowErr.Errors {
			multiErr.Add(field+"."+e.Field, e.Code, e.Message)
		}

		if _, locked := p.Locked[allocation.ID]; locked {
			multiErr.Add(
				field,
				errortypes.ErrInvalidOperation,
				"This allocation is already carried on a posted invoice and cannot change",
			)
		}

		if allocation.BillToCustomerID.IsNil() {
			continue
		}
		payer, ok := p.Customers[allocation.BillToCustomerID]
		switch {
		case !ok || payer == nil:
			multiErr.Add(
				field+".billToCustomerId",
				errortypes.ErrInvalid,
				"Bill-to customer was not found",
			)
		case payer.OrganizationID != p.TenantOrgID || payer.BusinessUnitID != p.TenantBuID:
			multiErr.Add(
				field+".billToCustomerId",
				errortypes.ErrInvalid,
				"Bill-to customer belongs to another organization",
			)
		case payer.Status != domaintypes.StatusActive:
			multiErr.Add(
				field+".billToCustomerId",
				errortypes.ErrInvalid,
				"Bill-to customer is not active",
			)
		}
	}

	return multiErr
}
