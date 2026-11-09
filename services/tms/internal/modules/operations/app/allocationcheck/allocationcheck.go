// Package allocationcheck validates charge allocations before they are written.
// The shipment form, the billing queue and order charges all change who pays for
// which charge, and all must refuse the same things for the same reasons.
package allocationcheck

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/chargeallocationports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	defaultField         = "chargeAllocations"
	orderAllocationField = "allocations"
)

// Readers are the repositories a check reads payers and posted rows through.
type Readers struct {
	Customers   customerports.CustomerRepository
	Allocations chargeallocationports.ChargeAllocationRepository
}

// ValidateShipment checks the shipment's allocations: every payer exists, is active
// and belongs to the tenant, no row already on a posted invoice changes, and every
// charge's rows add up. A nil allocation list means the caller did not touch them
// and passes. Field prefixes every error path.
func ValidateShipment(
	ctx context.Context,
	readers Readers,
	entity *shipment.Shipment,
	field string,
) error {
	if entity == nil || entity.ChargeAllocations == nil {
		return nil
	}
	if field == "" {
		field = defaultField
	}

	tenantInfo := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
	multiErr, err := checkRows(ctx, readers, tenantInfo, entity.ChargeAllocations, field)
	if err != nil {
		return err
	}
	if multiErr.HasErrors() {
		return multiErr.Err()
	}

	preview := PreviewForResolution(entity)
	_, err = chargeallocation.ResolveShares(ctx, preview, preview.ChargeAllocations)

	return resolutionError(multiErr, err, "resolve charge shares")
}

// OrderChargeRequest is one order charge and the allocations that split it.
type OrderChargeRequest struct {
	TenantInfo  pagination.TenantInfo
	Charge      *order.OrderCharge
	Allocations []*shipment.ChargeAllocation
}

// ValidateOrderCharge checks an order charge's allocations by the same rules. It
// marks each row as belonging to the charge; a charge with no id yet is checked
// against a placeholder that is cleared again before it returns.
func ValidateOrderCharge(ctx context.Context, readers Readers, req OrderChargeRequest) error {
	if req.Allocations == nil {
		return nil
	}

	chargeID := req.Charge.ID
	if chargeID.IsNil() {
		chargeID = pulid.MustNew("ordchg_")
	}
	for _, allocation := range req.Allocations {
		if allocation == nil {
			continue
		}
		id := chargeID
		allocation.ChargeKind = shipmentvocab.ChargeAllocationKindOrderCharge
		allocation.OrderChargeID = &id
	}
	if req.Charge.ID.IsNil() {
		defer clearOrderCharge(req.Allocations)
	}

	multiErr, err := checkRows(ctx, readers, req.TenantInfo, req.Allocations, orderAllocationField)
	if err != nil {
		return err
	}
	if multiErr.HasErrors() {
		return multiErr.Err()
	}

	_, err = chargeallocation.ResolveOrderChargeShares(
		ctx,
		[]chargeallocation.OrderChargeRef{
			{ID: chargeID, Description: req.Charge.Description, Amount: req.Charge.Amount},
		},
		req.Allocations,
		pulid.Nil,
	)

	return resolutionError(multiErr, err, "split the order charge")
}

func clearOrderCharge(allocations []*shipment.ChargeAllocation) {
	for _, allocation := range allocations {
		if allocation != nil {
			allocation.OrderChargeID = nil
		}
	}
}

func checkRows(
	ctx context.Context,
	readers Readers,
	tenantInfo pagination.TenantInfo,
	allocations []*shipment.ChargeAllocation,
	field string,
) (*errortypes.MultiError, error) {
	payerIDs, rowIDs := collectIDs(allocations)

	customers, err := loadCustomers(ctx, readers, tenantInfo, payerIDs)
	if err != nil {
		return nil, err
	}
	locked, err := loadLocked(ctx, readers, tenantInfo, rowIDs)
	if err != nil {
		return nil, err
	}

	return chargeallocation.ValidateAllocations(&chargeallocation.ValidateAllocationsParams{
		TenantOrgID: tenantInfo.OrgID,
		TenantBuID:  tenantInfo.BuID,
		Allocations: allocations,
		Customers:   customers,
		Locked:      locked,
		Field:       field,
	}), nil
}

func resolutionError(multiErr *errortypes.MultiError, err error, action string) error {
	if err == nil {
		return nil
	}

	typed, ok := errors.AsType[*errortypes.Error](err)
	if !ok {
		return fmt.Errorf("%s: %w", action, err)
	}
	if multiErr == nil {
		multiErr = errortypes.NewMultiError()
	}
	multiErr.AddError(typed)

	return multiErr.Err()
}

func collectIDs(allocations []*shipment.ChargeAllocation) (payerIDs, rowIDs []pulid.ID) {
	payerIDs = make([]pulid.ID, 0, len(allocations))
	rowIDs = make([]pulid.ID, 0, len(allocations))
	seen := make(map[pulid.ID]struct{}, len(allocations))
	for _, allocation := range allocations {
		if allocation == nil {
			continue
		}
		if allocation.ID.IsNotNil() {
			rowIDs = append(rowIDs, allocation.ID)
		}
		if allocation.BillToCustomerID.IsNil() {
			continue
		}
		if _, ok := seen[allocation.BillToCustomerID]; ok {
			continue
		}
		seen[allocation.BillToCustomerID] = struct{}{}
		payerIDs = append(payerIDs, allocation.BillToCustomerID)
	}

	return payerIDs, rowIDs
}

func loadCustomers(
	ctx context.Context,
	readers Readers,
	tenantInfo pagination.TenantInfo,
	payerIDs []pulid.ID,
) (map[pulid.ID]*customer.Customer, error) {
	customers := make(map[pulid.ID]*customer.Customer, len(payerIDs))
	if len(payerIDs) == 0 {
		return customers, nil
	}

	found, err := readers.Customers.GetByIDs(ctx, customerports.GetCustomersByIDsRequest{
		TenantInfo:  tenantInfo,
		CustomerIDs: payerIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("load bill-to customers: %w", err)
	}
	for _, cus := range found {
		if cus != nil {
			customers[cus.ID] = cus
		}
	}

	return customers, nil
}

func loadLocked(
	ctx context.Context,
	readers Readers,
	tenantInfo pagination.TenantInfo,
	rowIDs []pulid.ID,
) (map[pulid.ID]struct{}, error) {
	if len(rowIDs) == 0 {
		return map[pulid.ID]struct{}{}, nil
	}

	found, err := readers.Allocations.LockedIDs(ctx, tenantInfo, rowIDs)
	if err != nil {
		return nil, fmt.Errorf("load invoiced charge allocations: %w", err)
	}

	return found, nil
}

// PreviewForResolution returns a shallow copy of the shipment whose id-less
// charges carry throwaway ids and whose allocations point at them, so a split
// on a charge that has not been inserted yet can be resolved. The caller's
// entity is left untouched: the repository mints the real ids.
func PreviewForResolution(entity *shipment.Shipment) *shipment.Shipment {
	preview := *entity
	preview.AdditionalCharges = make([]*shipment.AdditionalCharge, len(entity.AdditionalCharges))
	for i, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}
		copied := *charge
		if copied.ID.IsNil() {
			copied.ID = pulid.MustNew("ac_")
		}
		preview.AdditionalCharges[i] = &copied
	}

	allocations := make([]*shipment.ChargeAllocation, 0, len(entity.ChargeAllocations))
	for _, allocation := range entity.ChargeAllocations {
		if allocation == nil {
			continue
		}
		copied := *allocation
		if copied.ChargeKind == shipmentvocab.ChargeAllocationKindAccessorial &&
			copied.AdditionalChargeIndex != nil {
			idx := *copied.AdditionalChargeIndex
			if idx >= 0 && idx < len(preview.AdditionalCharges) &&
				preview.AdditionalCharges[idx] != nil {
				id := preview.AdditionalCharges[idx].ID
				copied.AdditionalChargeID = &id
			}
		}
		allocations = append(allocations, &copied)
	}
	preview.ChargeAllocations = allocations

	return &preview
}
