package billingqueueservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/allocationcheck"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const reassignCancelReason = "Charges reassigned to other payers"

// reassignableStatuses are the states in which no invoice can exist for an item,
// so moving a charge between payers cannot contradict a document already cut.
var reassignableStatuses = map[billingqueue.Status]struct{}{
	billingqueue.StatusReadyForReview: {},
	billingqueue.StatusInReview:       {},
	billingqueue.StatusOnHold:         {},
}

// reassignPlan is what a reassignment will write, worked out before the
// transaction opens so every refusal happens without touching storage.
type reassignPlan struct {
	shipment    *shipment.Shipment
	allocations []*shipment.ChargeAllocation
	resolution  *shipment.ShareResolution
	active      []*billingqueue.BillingQueueItem
	toCreate    []*shipment.PayerShare
	toCancel    []*billingqueue.BillingQueueItem
	description string
}

// ReassignCharge changes who pays for one freight or accessorial charge while
// the shipment is still in review. Payers who gain a share get a queue item;
// payers left with nothing have theirs canceled; everyone else's allocated
// total is refreshed.
func (s *service) ReassignCharge(
	ctx context.Context,
	req *services.ReassignChargeRequest,
	actor *services.RequestActor,
) (*services.ReassignChargeResult, error) {
	if multiErr := validateReassignRequest(req, actor); multiErr != nil {
		return nil, multiErr
	}

	item, err := s.repo.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:     req.ItemID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if item.BillType != billingqueue.BillTypeInvoice || item.ShipmentID.IsNil() {
		return nil, errortypes.NewValidationError(
			"itemId",
			errortypes.ErrInvalidOperation,
			"Only a shipment invoice item can have its charges reassigned",
		)
	}
	if _, ok := reassignableStatuses[item.Status]; !ok {
		return nil, errortypes.NewValidationError(
			"itemId",
			errortypes.ErrInvalidOperation,
			"Charges can be reassigned only while the item is ready for review, in review or on hold",
		)
	}
	if err = s.guardSiblingPayersUnposted(ctx, item, req.TenantInfo); err != nil {
		return nil, err
	}

	plan, err := s.planReassignment(ctx, item, req)
	if err != nil {
		return nil, err
	}

	created, canceled, err := s.applyReassignment(ctx, plan, req, actor)
	if err != nil {
		return nil, err
	}

	return s.reassignmentResult(ctx, item, plan, created, canceled, req, actor)
}

func validateReassignRequest(
	req *services.ReassignChargeRequest,
	actor *services.RequestActor,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if req == nil {
		multiErr.Add("request", errortypes.ErrRequired, "Request is required")
		return multiErr
	}
	if actor == nil {
		multiErr.Add("actor", errortypes.ErrRequired, "Actor is required")
	} else if actor.IsAgent() {
		multiErr.Add("actor", errortypes.ErrForbidden, "Agent principals cannot reassign billing charges")
	}
	if req.ItemID.IsNil() {
		multiErr.Add("itemId", errortypes.ErrRequired, "Billing queue item is required")
	}
	switch req.ChargeKind {
	case shipment.ChargeAllocationKindFreight:
	case shipment.ChargeAllocationKindAccessorial:
		if req.AdditionalChargeID.IsNil() {
			multiErr.Add("additionalChargeId", errortypes.ErrRequired, "Choose the accessorial charge to reassign")
		}
	default:
		multiErr.Add("chargeKind", errortypes.ErrInvalid, "Only freight or an accessorial charge can be reassigned")
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *service) planReassignment(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	req *services.ReassignChargeRequest,
) (*reassignPlan, error) {
	shp, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         item.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	chargeTotal, description, err := reassignTarget(shp, req)
	if err != nil {
		return nil, err
	}

	rows, err := reassignRows(shp, req, chargeTotal)
	if err != nil {
		return nil, err
	}

	merged := make([]*shipment.ChargeAllocation, 0, len(rows)+len(shp.ChargeAllocations))
	merged = append(merged, rows...)
	for _, allocation := range shp.ChargeAllocations {
		if allocation == nil || targetsCharge(allocation, req) {
			continue
		}
		merged = append(merged, allocation)
	}

	candidate := *shp
	candidate.ChargeAllocations = merged
	if multiErr := allocationcheck.Validate(ctx, allocationcheck.Deps{
		CustomerRepo:         s.customerRepo,
		ChargeAllocationRepo: s.chargeAllocationRepo,
		Logger:               s.l,
	}, &candidate, "allocations"); multiErr != nil {
		return nil, multiErr
	}

	resolution, err := shipment.ResolveShares(&candidate, merged)
	if err != nil {
		return nil, err
	}

	activeByShipment, err := s.repo.ListActiveInvoiceItemsByShipmentIDs(
		ctx,
		&repositories.ListActiveInvoiceItemsRequest{
			TenantInfo:  req.TenantInfo,
			ShipmentIDs: []pulid.ID{shp.ID},
		},
	)
	if err != nil {
		return nil, err
	}
	active := activeByShipment[shp.ID]
	for _, sibling := range active {
		if sibling == nil {
			continue
		}
		if _, ok := reassignableStatuses[sibling.Status]; !ok {
			return nil, errortypes.NewValidationError(
				"itemId",
				errortypes.ErrInvalidOperation,
				"Queue item {0} is {1}; move it back to review before reassigning this shipment's charges",
				sibling.Number,
				string(sibling.Status),
			)
		}
	}

	shp.ChargeAllocations = merged
	plan := &reassignPlan{
		shipment:    shp,
		allocations: merged,
		resolution:  resolution,
		active:      active,
		description: description,
	}
	plan.toCreate, plan.toCancel = reconcileQueueItems(resolution, active)

	return plan, nil
}

// reassignTarget finds the charge being reassigned and its total.
func reassignTarget(
	shp *shipment.Shipment,
	req *services.ReassignChargeRequest,
) (total decimal.Decimal, description string, err error) {
	freight := shp.FreightChargeAmount.Decimal
	if req.ChargeKind == shipment.ChargeAllocationKindFreight {
		return freight, "Freight", nil
	}
	for _, charge := range shp.AdditionalCharges {
		if charge == nil || charge.ID != req.AdditionalChargeID {
			continue
		}
		name := "Accessorial charge"
		if charge.AccessorialCharge != nil {
			name = charge.AccessorialCharge.Description
			if name == "" {
				name = charge.AccessorialCharge.Code
			}
		}
		return charge.Total(freight), name, nil
	}

	return decimal.Zero, "", errortypes.NewValidationError(
		"additionalChargeId",
		errortypes.ErrInvalid,
		"That accessorial charge is not on this shipment",
	)
}

func targetsCharge(allocation *shipment.ChargeAllocation, req *services.ReassignChargeRequest) bool {
	if allocation.ChargeKind != req.ChargeKind {
		return false
	}
	if req.ChargeKind == shipment.ChargeAllocationKindFreight {
		return true
	}

	return allocation.TargetID() == req.AdditionalChargeID
}

// reassignRows turns the requested split into allocation rows for the charge.
// A row may keep the id of a row that already divides this charge; any other id
// is refused so a reassignment cannot move another charge's rows. A single row
// giving the whole charge to the shipment's own payer is the same as no rows.
func reassignRows(
	shp *shipment.Shipment,
	req *services.ReassignChargeRequest,
	chargeTotal decimal.Decimal,
) ([]*shipment.ChargeAllocation, error) {
	existing := make(map[pulid.ID]*shipment.ChargeAllocation, len(shp.ChargeAllocations))
	for _, allocation := range shp.ChargeAllocations {
		if allocation != nil && targetsCharge(allocation, req) {
			existing[allocation.ID] = allocation
		}
	}

	shipmentID := shp.ID
	rows := make([]*shipment.ChargeAllocation, 0, len(req.Allocations))
	for i, input := range req.Allocations {
		if input == nil {
			continue
		}
		row := &shipment.ChargeAllocation{
			OrganizationID:   shp.OrganizationID,
			BusinessUnitID:   shp.BusinessUnitID,
			ShipmentID:       &shipmentID,
			ChargeKind:       req.ChargeKind,
			BillToCustomerID: input.BillToCustomerID,
			Method:           input.Method,
			Percent:          input.Percent,
			Amount:           input.Amount,
			Sequence:         int16(i), //nolint:gosec // bounded by the request's row count
		}
		if req.ChargeKind == shipment.ChargeAllocationKindAccessorial {
			chargeID := req.AdditionalChargeID
			row.AdditionalChargeID = &chargeID
		}
		if input.ID.IsNotNil() {
			previous, ok := existing[input.ID]
			if !ok {
				return nil, errortypes.NewValidationError(
					"allocations",
					errortypes.ErrInvalid,
					"An allocation in the request does not belong to this charge",
				)
			}
			row.ID = previous.ID
			row.Version = previous.Version
			row.InvoiceID = previous.InvoiceID
			row.InvoicedAt = previous.InvoicedAt
		}
		rows = append(rows, row)
	}

	if len(rows) == 1 && rows[0].BillToCustomerID == shp.PayerID() && wholeCharge(rows[0], chargeTotal) {
		return []*shipment.ChargeAllocation{}, nil
	}

	return rows, nil
}

func wholeCharge(row *shipment.ChargeAllocation, total decimal.Decimal) bool {
	switch row.Method {
	case shipment.ChargeAllocationMethodPercent:
		return row.Percent.Valid && row.Percent.Decimal.Equal(decimalutils.Percent100)
	case shipment.ChargeAllocationMethodAmount:
		return row.Amount.Valid &&
			decimalutils.SumEquals([]decimal.Decimal{row.Amount.Decimal}, total, shipment.SharePlaces)
	default:
		return false
	}
}

// reconcileQueueItems compares who pays after the reassignment with who has an
// open queue item: payers who gained a share need one, and items whose payer now
// pays nothing are canceled.
func reconcileQueueItems(
	resolution *shipment.ShareResolution,
	active []*billingqueue.BillingQueueItem,
) (toCreate []*shipment.PayerShare, toCancel []*billingqueue.BillingQueueItem) {
	paying := make(map[pulid.ID]*shipment.PayerShare, len(resolution.Shares))
	for _, share := range resolution.Shares {
		if share != nil && len(share.Charges) > 0 {
			paying[share.PayerID] = share
		}
	}

	covered := make(map[pulid.ID]struct{}, len(active))
	for _, item := range active {
		if item == nil {
			continue
		}
		if _, ok := paying[item.BillToCustomerID]; ok {
			covered[item.BillToCustomerID] = struct{}{}
			continue
		}
		toCancel = append(toCancel, item)
	}
	for _, share := range resolution.Shares {
		if share == nil || len(share.Charges) == 0 {
			continue
		}
		if _, ok := covered[share.PayerID]; !ok {
			toCreate = append(toCreate, share)
		}
	}

	return toCreate, toCancel
}

func (s *service) applyReassignment(
	ctx context.Context,
	plan *reassignPlan,
	req *services.ReassignChargeRequest,
	actor *services.RequestActor,
) (created, canceled []*billingqueue.BillingQueueItem, err error) {
	newItems := make([]*billingqueue.BillingQueueItem, 0, len(plan.toCreate))
	for _, share := range plan.toCreate {
		number, numberErr := s.generateBillingNumber(
			ctx,
			billingqueue.BillTypeInvoice,
			req.TenantInfo.OrgID,
			req.TenantInfo.BuID,
		)
		if numberErr != nil {
			return nil, nil, numberErr
		}
		entity := &billingqueue.BillingQueueItem{
			OrganizationID:       req.TenantInfo.OrgID,
			BusinessUnitID:       req.TenantInfo.BuID,
			ShipmentID:           plan.shipment.ID,
			OrderID:              plan.shipment.OrderID,
			BillToCustomerID:     share.PayerID,
			AllocatedTotalAmount: share.TotalAmount,
			Status:               billingqueue.StatusReadyForReview,
			BillType:             billingqueue.BillTypeInvoice,
			Number:               number,
		}
		entity.SyncAllocatedMinor()
		if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
			return nil, nil, multiErr
		}
		newItems = append(newItems, entity)
	}

	now := timeutils.NowUnix()
	userID := actor.UserID
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		if txErr := s.chargeAllocationRepo.SyncForShipment(txCtx, tx, plan.shipment); txErr != nil {
			return txErr
		}
		for _, entity := range newItems {
			item, txErr := s.repo.Create(txCtx, entity)
			if txErr != nil {
				return txErr
			}
			created = append(created, item)
		}
		for _, entity := range plan.toCancel {
			entity.Status = billingqueue.StatusCanceled
			entity.CanceledByID = &userID
			entity.CanceledAt = &now
			entity.CancelReason = reassignCancelReason
			if multiErr := s.validator.ValidateUpdate(txCtx, entity); multiErr != nil {
				return multiErr
			}
			item, txErr := s.repo.Update(txCtx, entity)
			if txErr != nil {
				return txErr
			}
			canceled = append(canceled, item)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return created, canceled, nil
}

func (s *service) reassignmentResult(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	plan *reassignPlan,
	created, canceled []*billingqueue.BillingQueueItem,
	req *services.ReassignChargeRequest,
	actor *services.RequestActor,
) (*services.ReassignChargeResult, error) {
	auditActor := actor.AuditActor()
	comment := "Charge reassigned from billing queue: " + plan.description

	result := &services.ReassignChargeResult{
		CreatedItemIDs:  make([]pulid.ID, 0, len(created)),
		CanceledItemIDs: make([]pulid.ID, 0, len(canceled)),
	}
	for _, entity := range created {
		s.autoAssignDefaultBiller(ctx, entity, entity.BillToCustomerID, req.TenantInfo, actor)
		s.logAction(entity, auditActor, permission.OpCreate, nil, entity, comment)
		s.publishInvalidation(ctx, entity, auditActor, "created", entity)
		result.CreatedItemIDs = append(result.CreatedItemIDs, entity.ID)
	}
	for _, entity := range canceled {
		s.logAction(entity, auditActor, permission.OpUpdate, nil, entity, comment+". "+reassignCancelReason)
		s.publishInvalidation(ctx, entity, auditActor, "updated", entity)
		result.CanceledItemIDs = append(result.CanceledItemIDs, entity.ID)
	}
	if !containsID(result.CanceledItemIDs, item.ID) {
		s.logAction(item, auditActor, permission.OpUpdate, nil, nil, comment)
		s.publishInvalidation(ctx, item, auditActor, "updated", item)
	}

	reloaded, err := s.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:                item.ID,
		TenantInfo:            req.TenantInfo,
		ExpandShipmentDetails: true,
	})
	if err != nil {
		return nil, err
	}
	result.Item = reloaded

	active, err := s.repo.ListActiveInvoiceItemsByShipmentIDs(ctx, &repositories.ListActiveInvoiceItemsRequest{
		TenantInfo:  req.TenantInfo,
		ShipmentIDs: []pulid.ID{plan.shipment.ID},
	})
	if err != nil {
		return nil, err
	}
	result.Items = active[plan.shipment.ID]
	if result.Items == nil {
		result.Items = []*billingqueue.BillingQueueItem{}
	}

	return result, nil
}

func containsID(ids []pulid.ID, id pulid.ID) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}

	return false
}
