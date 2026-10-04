package billingqueueservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// driverMessageEvent is the notification family a dispatcher's message to a
// driver goes out as; asking for paperwork is one.
const driverMessageEvent = "dash.dispatch_message"

func errPersonDecides() error {
	return errortypes.NewValidationError(
		"actor",
		errortypes.ErrForbidden,
		"A person settles billing checks; an agent can only explain them",
	)
}

// reviewedItem reads the item the way the review shows it, which is also
// what every settlement re-checks against.
func (s *service) reviewedItem(
	ctx context.Context,
	itemID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*billingqueue.BillingQueueItem, error) {
	return s.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:                itemID,
		TenantInfo:            tenantInfo,
		ExpandShipmentDetails: true,
	})
}

// ResolveIssue settles an issue with one of its options and applies what the
// option does: billing a charge as it stands, taking it off, repricing it, or
// billing without the paperwork. Asking the driver is the one option that
// does not settle anything; the paperwork arriving does.
func (s *service) ResolveIssue(
	ctx context.Context,
	req *services.ResolveBillingQueueIssueRequest,
	actor *services.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	if s.reviewRepo == nil {
		return nil, errortypes.NewBusinessError("Billing checks are unavailable")
	}
	if actor.IsAgent() {
		return nil, errPersonDecides()
	}

	item, err := s.reviewedItem(ctx, req.ItemID, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if !billingqueue.IsReviewable(item.Status) {
		return nil, errortypes.NewValidationError(
			"status", errortypes.ErrInvalidOperation,
			"This item is past review, so its checks can't change",
		)
	}

	issue, err := s.reviewRepo.GetIssue(ctx, &repositories.GetBillingQueueIssueRequest{
		TenantInfo: req.TenantInfo,
		ItemID:     req.ItemID,
		IssueID:    req.IssueID,
	})
	if err != nil {
		return nil, err
	}
	if !issue.IsOpen() {
		return nil, errortypes.NewValidationError(
			"issueId", errortypes.ErrInvalidOperation, "This check is already settled",
		)
	}
	option := issue.Option(req.OptionKey)
	if option == nil {
		return nil, errortypes.NewValidationError(
			"optionKey", errortypes.ErrInvalid, "That isn't one of this check's choices",
		)
	}

	if option.Effect.Kind == billingqueue.EffectRequest {
		return s.requestPaperwork(ctx, item, issue, actor, req.TenantInfo)
	}

	var snapshot *billingqueue.EffectSnapshot
	switch option.Effect.Kind { //nolint:exhaustive // keep and accept change no charge
	case billingqueue.EffectDrop, billingqueue.EffectSet:
		snapshot, err = s.applyChargeEffect(ctx, item, option.Effect, actor, req.TenantInfo)
		if err != nil {
			return nil, err
		}
	}

	now := timeutils.NowUnix()
	key := option.Key
	issue.ResolutionKey = &key
	issue.ResolutionText = option.Done
	issue.ResolvedAt = &now
	issue.EffectSnapshot = snapshot
	if userID := actor.UserIDOrNil(); userID.IsNotNil() {
		issue.ResolvedByID = &userID
	}
	if _, err = s.reviewRepo.UpdateIssue(ctx, issue); err != nil {
		return nil, err
	}

	s.recordEvent(ctx, item, billingqueue.EventIssueResolved, option.Done, actor,
		map[string]any{"issueId": issue.ID.String(), "option": option.Key})
	s.publishInvalidation(ctx, item, actor.AuditActor(), "updated", item)

	return s.reviewedItem(ctx, req.ItemID, req.TenantInfo)
}

// UndoIssue takes a settlement back: the charge it removed or repriced is put
// back as it was, and the issue is open again.
func (s *service) UndoIssue(
	ctx context.Context,
	req *services.UndoBillingQueueIssueRequest,
	actor *services.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	if s.reviewRepo == nil {
		return nil, errortypes.NewBusinessError("Billing checks are unavailable")
	}
	if actor.IsAgent() {
		return nil, errPersonDecides()
	}

	item, err := s.reviewedItem(ctx, req.ItemID, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	issue, err := s.reviewRepo.GetIssue(ctx, &repositories.GetBillingQueueIssueRequest{
		TenantInfo: req.TenantInfo,
		ItemID:     req.ItemID,
		IssueID:    req.IssueID,
	})
	if err != nil {
		return nil, err
	}
	if !issue.Settled() || !billingqueue.IsReviewable(item.Status) {
		return nil, errortypes.NewValidationError(
			"issueId", errortypes.ErrInvalidOperation, "This can't be undone now",
		)
	}
	option := issue.Option(*issue.ResolutionKey)

	if option != nil && issue.EffectSnapshot != nil && issue.EffectSnapshot.Charge != nil {
		restoredID, restoreErr := s.restoreCharge(ctx, item, option.Effect.Kind, issue.EffectSnapshot, actor, req.TenantInfo)
		if restoreErr != nil {
			return nil, restoreErr
		}
		if restoredID.IsNotNil() && issue.FlaggedChargeID != nil && restoredID != *issue.FlaggedChargeID {
			// A removed charge comes back as a new row; the issue follows it,
			// so the next check finds the same issue rather than a new one.
			issue.FlaggedChargeID = &restoredID
			if issue.SubjectKey != "" {
				issue.SubjectKey = restoredID.String()
			}
		}
	}

	done := issue.ResolutionText
	issue.ResolutionKey = nil
	issue.ResolutionText = ""
	issue.ResolvedAt = nil
	issue.ResolvedByID = nil
	issue.EffectSnapshot = nil
	if _, err = s.reviewRepo.UpdateIssue(ctx, issue); err != nil {
		return nil, err
	}

	s.recordEvent(ctx, item, billingqueue.EventIssueUndone, "Undid: "+lowerFirst(done), actor,
		map[string]any{"issueId": issue.ID.String()})
	s.publishInvalidation(ctx, item, actor.AuditActor(), "updated", item)

	return s.reviewedItem(ctx, req.ItemID, req.TenantInfo)
}

// applyChargeEffect makes an option's charge edit through the same path a
// biller's own charge edit takes, and returns the charge as it was so undo
// can put it back.
func (s *service) applyChargeEffect(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	effect billingqueue.IssueEffect,
	actor *services.RequestActor,
	tenantInfo pagination.TenantInfo,
) (*billingqueue.EffectSnapshot, error) {
	if item.Shipment == nil {
		return nil, errortypes.NewBusinessError("The shipment's charges could not be read")
	}

	var target *shipment.AdditionalCharge
	charges := make([]*shipment.AdditionalCharge, 0, len(item.Shipment.AdditionalCharges))
	for _, charge := range item.Shipment.AdditionalCharges {
		if charge == nil {
			continue
		}
		clone := *charge
		if charge.ID == effect.ChargeID {
			target = &clone
			if effect.Kind == billingqueue.EffectDrop {
				continue
			}
			clone.Method = accessorialcharge.MethodFlat
			clone.Amount = effect.Amount.Decimal
			clone.Unit = 1
		}
		charges = append(charges, &clone)
	}
	if target == nil {
		return nil, errortypes.NewValidationError(
			"issueId", errortypes.ErrInvalidOperation,
			"The charge this check is about is no longer on the shipment",
		)
	}
	if effect.Kind == billingqueue.EffectSet && !effect.Amount.Valid {
		return nil, errortypes.NewBusinessError("This choice has no amount to bill")
	}

	snapshot := &billingqueue.EffectSnapshot{Charge: chargeSnapshot(target, item)}
	if _, err := s.updateCharges(ctx, &services.UpdateChargesRequest{
		ItemID:            item.ID,
		AdditionalCharges: charges,
		TenantInfo:        tenantInfo,
	}, actor, true); err != nil {
		return nil, err
	}

	return snapshot, nil
}

// restoreCharge puts a charge back the way the snapshot recorded it, and
// returns the id it has now: the same row for a repriced charge, a new one
// for a charge that had been removed.
func (s *service) restoreCharge(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	kind billingqueue.EffectKind,
	snapshot *billingqueue.EffectSnapshot,
	actor *services.RequestActor,
	tenantInfo pagination.TenantInfo,
) (pulid.ID, error) {
	if item.Shipment == nil {
		return pulid.Nil, errortypes.NewBusinessError("The shipment's charges could not be read")
	}
	snap := snapshot.Charge
	before := make(map[pulid.ID]struct{}, len(item.Shipment.AdditionalCharges))
	charges := make([]*shipment.AdditionalCharge, 0, len(item.Shipment.AdditionalCharges)+1)
	found := false
	for _, charge := range item.Shipment.AdditionalCharges {
		if charge == nil {
			continue
		}
		before[charge.ID] = struct{}{}
		clone := *charge
		if kind == billingqueue.EffectSet && charge.ID == snap.AdditionalChargeID {
			clone.Method = accessorialMethod(snap.Method)
			clone.Amount = snap.Amount
			clone.Unit = snap.Unit
			found = true
		}
		charges = append(charges, &clone)
	}
	if !found {
		charges = append(charges, &shipment.AdditionalCharge{
			OrganizationID:      item.OrganizationID,
			BusinessUnitID:      item.BusinessUnitID,
			ShipmentID:          item.ShipmentID,
			AccessorialChargeID: snap.AccessorialChargeID,
			Method:              accessorialMethod(snap.Method),
			Amount:              snap.Amount,
			Unit:                max(snap.Unit, 1),
		})
	}

	if _, err := s.updateCharges(ctx, &services.UpdateChargesRequest{
		ItemID:            item.ID,
		AdditionalCharges: charges,
		TenantInfo:        tenantInfo,
	}, actor, true); err != nil {
		return pulid.Nil, err
	}
	if found {
		return snap.AdditionalChargeID, nil
	}

	shp, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         item.ShipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return pulid.Nil, err
	}
	for _, charge := range shp.AdditionalCharges {
		if charge == nil {
			continue
		}
		if _, existed := before[charge.ID]; !existed {
			return charge.ID, nil
		}
	}

	return pulid.Nil, nil
}

func chargeSnapshot(
	charge *shipment.AdditionalCharge,
	item *billingqueue.BillingQueueItem,
) *billingqueue.ChargeSnapshot {
	snap := &billingqueue.ChargeSnapshot{
		AdditionalChargeID:  charge.ID,
		AccessorialChargeID: charge.AccessorialChargeID,
		Method:              string(charge.Method),
		Amount:              charge.Amount,
		Unit:                charge.Unit,
		Billed:              charge.Amount,
	}
	if item.Review != nil && item.Review.Charges != nil {
		for _, line := range item.Review.Charges.Lines {
			if line.AdditionalChargeID != charge.ID {
				continue
			}
			snap.Label = line.Label
			snap.Basis = line.Basis
			snap.Billed = line.Billed
			if line.Expected.Valid {
				expected := line.Expected.Decimal.String()
				snap.Expected = &expected
			}
		}
	}

	return snap
}

// requestPaperwork asks the shipment's driver to send a signed POD through
// the driver portal, and records that it was asked. The issue stays open:
// it settles when the signed copy is filed, or when a person bills without it.
func (s *service) requestPaperwork(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	issue *billingqueue.Issue,
	actor *services.RequestActor,
	tenantInfo pagination.TenantInfo,
) (*billingqueue.BillingQueueItem, error) {
	pod := podState(item.Shipment, nil, nil)
	ref := shipmentRef(item.Shipment)
	if s.drivers != nil && pod.DriverID.IsNotNil() {
		s.drivers.Notify(ctx, &drivernotificationservice.DriverNotification{
			TenantInfo: tenantInfo,
			WorkerID:   pod.DriverID,
			EventType:  driverMessageEvent,
			Priority:   notification.PriorityHigh,
			Context: documenttemplate.DriverNotificationContext{
				AlertTitle:   "Signed POD needed for " + ref,
				AlertMessage: "Billing needs the receiver's signed POD for " + ref + ". Please send it from the app.",
			},
			RelatedEntities: map[string]any{"shipmentId": item.ShipmentID.String()},
			Link:            "/loads/" + item.ShipmentID.String(),
		})
	}

	now := timeutils.NowUnix()
	issue.RequestedAt = &now
	if userID := actor.UserIDOrNil(); userID.IsNotNil() {
		issue.RequestedByID = &userID
	}
	if _, err := s.reviewRepo.UpdateIssue(ctx, issue); err != nil {
		return nil, err
	}

	who := pod.DriverName
	if who == "" {
		who = "the driver"
	}
	s.recordEvent(ctx, item, billingqueue.EventDocumentRequested,
		"Asked "+who+" for a signed POD", actor,
		map[string]any{"issueId": issue.ID.String()})
	s.publishInvalidation(ctx, item, actor.AuditActor(), "updated", item)

	return s.reviewedItem(ctx, item.ID, tenantInfo)
}

// Release takes an item off hold and back to where the hold found it.
func (s *service) Release(
	ctx context.Context,
	req *services.BillingQueueItemRequest,
	actor *services.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	item, err := s.repo.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:     req.ItemID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if item.Status != billingqueue.StatusOnHold {
		return nil, errortypes.NewValidationError(
			"status", errortypes.ErrInvalidOperation, "This item isn't on hold",
		)
	}

	target := billingqueue.StatusReadyForReview
	if before := item.StatusBeforeHold; before != nil &&
		*before != billingqueue.StatusOnHold &&
		billingqueue.IsAllowedTransition(billingqueue.StatusOnHold, *before) {
		target = *before
	}

	return s.UpdateStatus(ctx, &services.UpdateBillingQueueStatusRequest{
		ItemID:     req.ItemID,
		NewStatus:  target,
		TenantInfo: req.TenantInfo,
	}, actor)
}

// Post posts the invoice approval made for the item, which sends it to the
// customer and cannot be undone. An item approved onto a statement has no
// invoice of its own to post.
func (s *service) Post(
	ctx context.Context,
	req *services.BillingQueueItemRequest,
	actor *services.RequestActor,
) (*services.PostBillingQueueItemResult, error) {
	if actor.IsAgent() {
		return nil, errPersonDecides()
	}
	if s.invoiceSvc == nil || s.invoiceRepo == nil {
		return nil, errortypes.NewBusinessError("Posting is unavailable")
	}

	item, err := s.repo.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:     req.ItemID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if item.Status != billingqueue.StatusApproved && item.Status != billingqueue.StatusPosted {
		return nil, errortypes.NewValidationError(
			"status", errortypes.ErrInvalidOperation, "Approve the item before posting it",
		)
	}

	inv, err := s.invoiceRepo.GetByBillingQueueItemID(ctx, repositories.GetInvoiceByBillingQueueItemIDRequest{
		BillingQueueItemID: item.ID,
		TenantInfo:         req.TenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewValidationError(
				"status", errortypes.ErrInvalidOperation,
				"This item goes out on the customer's statement, so it posts with the statement",
			)
		}
		return nil, err
	}

	wasPosted := item.Status == billingqueue.StatusPosted
	posted, err := s.invoiceSvc.Post(ctx, &services.PostInvoiceRequest{
		InvoiceID:   inv.ID,
		TenantInfo:  req.TenantInfo,
		TriggeredBy: "billing_queue_item",
	}, actor)
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:     req.ItemID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	result := &services.PostBillingQueueItemResult{
		Item:          updated,
		InvoiceID:     posted.ID,
		InvoiceNumber: posted.Number,
		Recipients:    []string{},
	}
	if cus, cusErr := s.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
		ID:         item.BillToCustomerID,
		TenantInfo: req.TenantInfo,
		CustomerFilterOptions: repositories.CustomerFilterOptions{
			IncludeBillingProfile: true,
			IncludeEmailProfile:   true,
		},
	}); cusErr == nil && cus != nil {
		terms := termsFor(cus, nil)
		result.Recipients = terms.Recipients
		if len(terms.Recipients) > 0 && cus.BillingProfile != nil &&
			cus.BillingProfile.AutoSendInvoiceOnGeneration && cus.BillingProfile.EmailInvoiceEnabled {
			result.SentTo = terms.Recipients[0]
		}
	}

	if !wasPosted {
		text := "Posted as " + posted.Number
		if result.SentTo != "" {
			text += " · emailed to " + result.SentTo
		}
		s.recordEvent(ctx, updated, billingqueue.EventPosted, text, actor,
			map[string]any{"invoiceId": posted.ID.String(), "invoiceNumber": posted.Number})
		s.publishInvalidation(ctx, updated, actor.AuditActor(), "updated", updated)
	}

	return result, nil
}

// ApproveIfReady approves an item only if, read fresh, every check passes.
// It never approves on the strength of what a table showed a few seconds ago.
func (s *service) ApproveIfReady(
	ctx context.Context,
	req *services.ApproveIfReadyRequest,
	actor *services.RequestActor,
) (*services.ApproveIfReadyResult, error) {
	item, err := s.reviewedItem(ctx, req.ItemID, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if req.AssignApprover && onlyWantsBiller(item) && actor != nil && actor.UserID.IsNotNil() {
		if _, err = s.AssignBiller(ctx, &services.AssignBillerRequest{
			ItemID:     req.ItemID,
			BillerID:   actor.UserID,
			TenantInfo: req.TenantInfo,
		}, actor); err != nil {
			return nil, err
		}
		if item, err = s.reviewedItem(ctx, req.ItemID, req.TenantInfo); err != nil {
			return nil, err
		}
	}
	result := &services.ApproveIfReadyResult{Item: item}

	switch {
	case item.Status == billingqueue.StatusApproved || item.Status == billingqueue.StatusPosted:
		result.FailureCode = billingqueue.ApprovalFailureAlreadyDone
		result.Reason = "Already approved"
		return result, nil
	case item.Status == billingqueue.StatusOnHold:
		result.FailureCode = billingqueue.ApprovalFailureOnHold
		result.Reason = "On hold"
		return result, nil
	case item.Review == nil || !item.Review.Ready:
		result.FailureCode = billingqueue.ApprovalFailureNotReady
		result.Reason = blockerReason(item)
		return result, nil
	}

	approved, err := s.UpdateStatus(ctx, &services.UpdateBillingQueueStatusRequest{
		ItemID:     req.ItemID,
		NewStatus:  billingqueue.StatusApproved,
		TenantInfo: req.TenantInfo,
	}, actor)
	if err != nil {
		return nil, err
	}
	result.Item = approved
	result.Approved = true

	if s.invoiceRepo != nil {
		if inv, invErr := s.invoiceRepo.GetByBillingQueueItemID(ctx, repositories.GetInvoiceByBillingQueueItemIDRequest{
			BillingQueueItemID: approved.ID,
			TenantInfo:         req.TenantInfo,
		}); invErr == nil && inv != nil {
			id := inv.ID
			result.InvoiceID = &id
			result.InvoiceNumber = inv.Number
		}
	}

	return result, nil
}

// onlyWantsBiller is an item whose one open check is that nobody is its
// biller, so naming one is all approving it waits on. An item with anything
// else open is left as it is rather than assigned and not approved.
func onlyWantsBiller(item *billingqueue.BillingQueueItem) bool {
	return item.Review != nil &&
		item.Review.Blocker == billingqueue.BlockerBiller &&
		item.Review.NeedsCount == 1
}

func blockerReason(item *billingqueue.BillingQueueItem) string {
	if item.Review == nil {
		return "Its checks could not be read"
	}
	switch item.Review.Blocker {
	case billingqueue.BlockerBiller:
		return "Nobody is assigned as biller"
	case billingqueue.BlockerIssue:
		for _, check := range item.Review.Checks {
			if check.State != billingqueue.CheckStateOK {
				return check.Detail
			}
		}
		return "A check needs a person"
	case billingqueue.BlockerStatus:
		return "Not ready for review"
	default:
		return "Not ready"
	}
}

func accessorialMethod(method string) accessorialcharge.Method {
	if method == "" {
		return accessorialcharge.MethodFlat
	}

	return accessorialcharge.Method(method)
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}

	return strings.ToLower(s[:1]) + s[1:]
}
