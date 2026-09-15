package shipmentservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// validateChargeAllocations checks a save's allocations before anything is
// written: every payer exists and is active, no invoiced row changes hands, and
// every charge's rows add up. A nil list means the caller did not touch them.
func (s *service) validateChargeAllocations(
	ctx context.Context,
	entity *shipment.Shipment,
) *errortypes.MultiError {
	if entity == nil || entity.ChargeAllocations == nil {
		return nil
	}
	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	payerIDs := make([]pulid.ID, 0, len(entity.ChargeAllocations))
	rowIDs := make([]pulid.ID, 0, len(entity.ChargeAllocations))
	seen := make(map[pulid.ID]struct{}, len(entity.ChargeAllocations))
	for _, allocation := range entity.ChargeAllocations {
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

	customers := make(map[pulid.ID]*customer.Customer, len(payerIDs))
	if len(payerIDs) > 0 && s.customerRepo != nil {
		found, err := s.customerRepo.GetByIDs(ctx, repositories.GetCustomersByIDsRequest{
			TenantInfo:  tenantInfo,
			CustomerIDs: payerIDs,
		})
		if err != nil {
			multiErr := errortypes.NewMultiError()
			multiErr.Add("chargeAllocations", errortypes.ErrInvalid, "Unable to verify bill-to customers")
			return multiErr
		}
		for _, cus := range found {
			if cus != nil {
				customers[cus.ID] = cus
			}
		}
	}

	locked := make(map[pulid.ID]struct{})
	if len(rowIDs) > 0 && s.chargeAllocationRepo != nil {
		found, err := s.chargeAllocationRepo.LockedIDs(ctx, tenantInfo, rowIDs)
		if err != nil {
			multiErr := errortypes.NewMultiError()
			multiErr.Add("chargeAllocations", errortypes.ErrInvalid, "Unable to verify invoiced allocations")
			return multiErr
		}
		locked = found
	}

	multiErr := shipment.ValidateAllocations(&shipment.ValidateAllocationsParams{
		TenantOrgID: entity.OrganizationID,
		TenantBuID:  entity.BusinessUnitID,
		Allocations: entity.ChargeAllocations,
		Customers:   customers,
		Locked:      locked,
		Field:       "chargeAllocations",
	})
	if multiErr.HasErrors() {
		return multiErr
	}

	if _, err := shipment.ResolveShares(previewForResolution(entity), entity.ChargeAllocations); err != nil {
		var typed *errortypes.Error
		if errors.As(err, &typed) {
			multiErr.Add(typed.Field, typed.Code, typed.Message)
		} else {
			multiErr.Add("chargeAllocations", errortypes.ErrInvalid, err.Error())
		}
		return multiErr
	}

	return nil
}

// previewForResolution returns a shallow copy of the shipment whose id-less
// charges carry throwaway ids, so allocations that point at a charge by position
// can be resolved before the charge has been inserted. The caller's entity is
// left untouched: the repository mints the real ids.
func previewForResolution(entity *shipment.Shipment) *shipment.Shipment {
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
