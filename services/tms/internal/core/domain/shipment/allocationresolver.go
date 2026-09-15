package shipment

import (
	"sort"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const SharePlaces = 2

// AllocatedCharge is one payer's slice of one charge.
type AllocatedCharge struct {
	Kind          ChargeAllocationKind
	AllocationID  pulid.ID
	Charge        *AdditionalCharge
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

func (p *PayerShare) add(charge AllocatedCharge) {
	p.Charges = append(p.Charges, charge)
	switch charge.Kind {
	case ChargeAllocationKindFreight:
		p.FreightAmount = p.FreightAmount.Add(charge.Amount)
	case ChargeAllocationKindAccessorial, ChargeAllocationKindOrderCharge:
		p.AccessorialAmount = p.AccessorialAmount.Add(charge.Amount)
	}
	p.TotalAmount = p.TotalAmount.Add(charge.Amount)
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

type shareBuilder struct {
	defaultPayer pulid.ID
	byPayer      map[pulid.ID]*PayerShare
	order        []pulid.ID
	split        bool
}

func newShareBuilder(defaultPayer pulid.ID) *shareBuilder {
	b := &shareBuilder{
		defaultPayer: defaultPayer,
		byPayer:      make(map[pulid.ID]*PayerShare, 2),
	}
	b.share(defaultPayer)

	return b
}

func (b *shareBuilder) share(payerID pulid.ID) *PayerShare {
	if existing, ok := b.byPayer[payerID]; ok {
		return existing
	}
	share := &PayerShare{PayerID: payerID}
	b.byPayer[payerID] = share
	b.order = append(b.order, payerID)

	return share
}

func (b *shareBuilder) resolution() *ShareResolution {
	shares := make([]*PayerShare, 0, len(b.order))
	shares = append(shares, b.byPayer[b.defaultPayer])
	others := make([]pulid.ID, 0, len(b.order))
	for _, id := range b.order {
		if id != b.defaultPayer {
			others = append(others, id)
		}
	}
	sort.Slice(others, func(i, j int) bool { return others[i] < others[j] })
	for _, id := range others {
		shares = append(shares, b.byPayer[id])
	}

	return &ShareResolution{
		DefaultPayerID: b.defaultPayer,
		Shares:         shares,
		IsSplit:        b.split || len(shares) > 1,
	}
}

// AllocateCharge divides one charge among the allocations that name it, or
// gives it whole to the default payer when nothing does. Percent rows are
// rounded to cents with the remainder on the last row by sequence; amount rows
// must add up to the charge exactly.
func (b *shareBuilder) allocate(
	base AllocatedCharge,
	allocations []*ChargeAllocation,
	field string,
) error {
	if len(allocations) == 0 {
		base.Percent = decimal.NewNullDecimal(decimalutils.Percent100)
		base.Amount = base.ChargeTotal.RoundBank(SharePlaces)
		base.Partial = false
		b.share(b.defaultPayer).add(base)
		return nil
	}

	sorted := make([]*ChargeAllocation, len(allocations))
	copy(sorted, allocations)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Sequence != sorted[j].Sequence {
			return sorted[i].Sequence < sorted[j].Sequence
		}
		return sorted[i].ID < sorted[j].ID
	})

	method := sorted[0].Method
	amounts := make([]decimal.Decimal, 0, len(sorted))
	for _, allocation := range sorted {
		if allocation.Method != method {
			return errortypes.NewValidationError(
				field,
				errortypes.ErrInvalid,
				"A charge must be split either by percent or by amount, not both",
			)
		}
	}

	switch method {
	case ChargeAllocationMethodPercent:
		percents := make([]decimal.Decimal, 0, len(sorted))
		for _, allocation := range sorted {
			percents = append(percents, allocation.Percent.Decimal)
		}
		shares, err := decimalutils.AllocatePercent(base.ChargeTotal, percents, SharePlaces)
		if err != nil {
			return errortypes.NewValidationError(
				field,
				errortypes.ErrInvalid,
				"Percent allocations for this charge must total 100",
			)
		}
		amounts = shares
	case ChargeAllocationMethodAmount:
		for _, allocation := range sorted {
			amounts = append(amounts, allocation.Amount.Decimal)
		}
		if !decimalutils.SumEquals(amounts, base.ChargeTotal, SharePlaces) {
			return errortypes.NewValidationError(
				field,
				errortypes.ErrInvalid,
				"Amount allocations for this charge must add up to {0}",
				base.ChargeTotal.StringFixed(SharePlaces),
			)
		}
	default:
		return errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"Allocation method must be Percent or Amount",
		)
	}

	for i, allocation := range sorted {
		share := base
		share.AllocationID = allocation.ID
		share.Amount = amounts[i].RoundBank(SharePlaces)
		share.Partial = !share.Amount.Equal(base.ChargeTotal.RoundBank(SharePlaces))
		switch method {
		case ChargeAllocationMethodPercent:
			share.Percent = allocation.Percent
		case ChargeAllocationMethodAmount:
			share.Percent = decimal.NullDecimal{}
			if !base.ChargeTotal.IsZero() {
				share.Percent = decimal.NewNullDecimal(
					share.Amount.Mul(decimalutils.Percent100).Div(base.ChargeTotal).Round(6),
				)
			}
		}
		if share.Partial || allocation.BillToCustomerID != b.defaultPayer {
			b.split = true
		}
		b.share(allocation.BillToCustomerID).add(share)
	}

	return nil
}

// ResolveShares divides a shipment's freight and accessorials among its payers.
// Percentage accessorials are computed on the full freight charge and only then
// split, so a payer with 40% of the linehaul does not also see a smaller fuel
// surcharge than the contract produced.
func ResolveShares(shp *Shipment, allocations []*ChargeAllocation) (*ShareResolution, error) {
	if shp == nil {
		return nil, errortypes.NewValidationError(
			"shipment",
			errortypes.ErrRequired,
			"Shipment is required",
		)
	}

	builder := newShareBuilder(shp.PayerID())
	freight := shp.FreightChargeAmount.Decimal

	freightAllocations := make([]*ChargeAllocation, 0, len(allocations))
	byCharge := make(map[pulid.ID][]*ChargeAllocation, len(allocations))
	for _, allocation := range allocations {
		if allocation == nil {
			continue
		}
		switch allocation.ChargeKind {
		case ChargeAllocationKindFreight:
			freightAllocations = append(freightAllocations, allocation)
		case ChargeAllocationKindAccessorial:
			target := allocation.TargetID()
			if target.IsNil() {
				continue
			}
			byCharge[target] = append(byCharge[target], allocation)
		case ChargeAllocationKindOrderCharge:
		}
	}

	if err := builder.allocate(AllocatedCharge{
		Kind:        ChargeAllocationKindFreight,
		ChargeTotal: freight,
	}, freightAllocations, "freightAllocations"); err != nil {
		return nil, err
	}

	for i, charge := range shp.AdditionalCharges {
		if charge == nil {
			continue
		}
		field := "additionalCharges[" + strconv.Itoa(i) + "].allocations"
		if err := builder.allocate(AllocatedCharge{
			Kind:        ChargeAllocationKindAccessorial,
			Charge:      charge,
			ChargeTotal: charge.Total(freight),
		}, byCharge[charge.ID], field); err != nil {
			return nil, err
		}
	}

	return builder.resolution(), nil
}

// ResolveOrderChargeShares divides order-level charges the same way, with the
// order's own payer as the default. Charges are identified by id, description
// and amount so the order package can call this without a cycle.
type OrderChargeRef struct {
	ID          pulid.ID
	Description string
	Amount      decimal.Decimal
}

func ResolveOrderChargeShares(
	charges []OrderChargeRef,
	allocations []*ChargeAllocation,
	defaultPayerID pulid.ID,
) (*ShareResolution, error) {
	builder := newShareBuilder(defaultPayerID)

	byCharge := make(map[pulid.ID][]*ChargeAllocation, len(allocations))
	for _, allocation := range allocations {
		if allocation == nil || allocation.ChargeKind != ChargeAllocationKindOrderCharge {
			continue
		}
		target := allocation.TargetID()
		if target.IsNil() {
			continue
		}
		byCharge[target] = append(byCharge[target], allocation)
	}

	for i, charge := range charges {
		field := "charges[" + strconv.Itoa(i) + "].allocations"
		if err := builder.allocate(AllocatedCharge{
			Kind:          ChargeAllocationKindOrderCharge,
			OrderChargeID: charge.ID,
			Description:   charge.Description,
			ChargeTotal:   charge.Amount,
		}, byCharge[charge.ID], field); err != nil {
			return nil, err
		}
	}

	return builder.resolution(), nil
}

// ValidateAllocationsParams carries what a validation pass needs to judge a set
// of allocations without touching storage.
type ValidateAllocationsParams struct {
	TenantOrgID pulid.ID
	TenantBuID  pulid.ID
	Allocations []*ChargeAllocation
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
			multiErr.Add(field+".billToCustomerId", errortypes.ErrInvalid, "Bill-to customer was not found")
		case payer.OrganizationID != p.TenantOrgID || payer.BusinessUnitID != p.TenantBuID:
			multiErr.Add(field+".billToCustomerId", errortypes.ErrInvalid, "Bill-to customer belongs to another organization")
		case payer.Status != domaintypes.StatusActive:
			multiErr.Add(field+".billToCustomerId", errortypes.ErrInvalid, "Bill-to customer is not active")
		}
	}

	return multiErr
}
