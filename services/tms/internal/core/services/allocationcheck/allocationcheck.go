// Package allocationcheck validates charge allocations before they are written.
// The shipment form and the billing queue both change who pays for which charge,
// and both must refuse the same things for the same reasons.
package allocationcheck

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const defaultField = "chargeAllocations"

// Deps is what validation reads. A nil ChargeAllocationRepo skips the posted
// invoice lock check. Payers are always checked, so without a CustomerRepo every
// named payer reads as not found.
type Deps struct {
	CustomerRepo         repositories.CustomerRepository
	ChargeAllocationRepo repositories.ChargeAllocationRepository
	Logger               *zap.Logger
}

// Validate checks the shipment's allocations: every payer exists, is active and
// belongs to the tenant, no row already on a posted invoice changes, and every
// charge's rows add up. A nil allocation list means the caller did not touch
// them and passes. Field prefixes every error path.
func Validate(
	ctx context.Context,
	deps Deps,
	entity *shipment.Shipment,
	field string,
) *errortypes.MultiError {
	if entity == nil || entity.ChargeAllocations == nil {
		return nil
	}
	if field == "" {
		field = defaultField
	}
	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	payerIDs, rowIDs := collectIDs(entity.ChargeAllocations)

	customers, multiErr := loadCustomers(ctx, deps, tenantInfo, payerIDs, field)
	if multiErr != nil {
		return multiErr
	}
	locked, multiErr := loadLocked(ctx, deps, tenantInfo, rowIDs, field)
	if multiErr != nil {
		return multiErr
	}

	multiErr = shipment.ValidateAllocations(&shipment.ValidateAllocationsParams{
		TenantOrgID: entity.OrganizationID,
		TenantBuID:  entity.BusinessUnitID,
		Allocations: entity.ChargeAllocations,
		Customers:   customers,
		Locked:      locked,
		Field:       field,
	})
	if multiErr.HasErrors() {
		return multiErr
	}

	preview := PreviewForResolution(entity)
	if _, err := shipment.ResolveShares(preview, preview.ChargeAllocations); err != nil {
		var typed *errortypes.Error
		if errors.As(err, &typed) {
			multiErr.Add(typed.Field, typed.Code, typed.Message, typed.Args...)
		} else {
			multiErr.Add(field, errortypes.ErrInvalid, err.Error())
		}
		return multiErr
	}

	return nil
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
	deps Deps,
	tenantInfo pagination.TenantInfo,
	payerIDs []pulid.ID,
	field string,
) (map[pulid.ID]*customer.Customer, *errortypes.MultiError) {
	customers := make(map[pulid.ID]*customer.Customer, len(payerIDs))
	if len(payerIDs) == 0 || deps.CustomerRepo == nil {
		return customers, nil
	}
	found, err := deps.CustomerRepo.GetByIDs(ctx, repositories.GetCustomersByIDsRequest{
		TenantInfo:  tenantInfo,
		CustomerIDs: payerIDs,
	})
	if err != nil {
		logError(deps, "failed to load bill-to customers for allocation check", err)
		multiErr := errortypes.NewMultiError()
		multiErr.Add(field, errortypes.ErrInvalid, "Unable to verify bill-to customers")
		return nil, multiErr
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
	deps Deps,
	tenantInfo pagination.TenantInfo,
	rowIDs []pulid.ID,
	field string,
) (map[pulid.ID]struct{}, *errortypes.MultiError) {
	if len(rowIDs) == 0 || deps.ChargeAllocationRepo == nil {
		return map[pulid.ID]struct{}{}, nil
	}
	found, err := deps.ChargeAllocationRepo.LockedIDs(ctx, tenantInfo, rowIDs)
	if err != nil {
		logError(deps, "failed to look up invoiced charge allocations", err)
		multiErr := errortypes.NewMultiError()
		multiErr.Add(field, errortypes.ErrInvalid, "Unable to verify invoiced allocations")
		return nil, multiErr
	}

	return found, nil
}

func logError(deps Deps, message string, err error) {
	if deps.Logger != nil {
		deps.Logger.Error(message, zap.Error(err))
	}
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
		if copied.ChargeKind == shipment.ChargeAllocationKindAccessorial &&
			copied.AdditionalChargeIndex != nil {
			idx := *copied.AdditionalChargeIndex
			if idx >= 0 && idx < len(preview.AdditionalCharges) && preview.AdditionalCharges[idx] != nil {
				id := preview.AdditionalCharges[idx].ID
				copied.AdditionalChargeID = &id
			}
		}
		allocations = append(allocations, &copied)
	}
	preview.ChargeAllocations = allocations

	return &preview
}
