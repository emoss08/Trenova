package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/zap"
)

const unexpectedBillingTransferMessage = "The shipment could not be transferred to billing because of an unexpected error"

type billingTransferAttemptParams struct {
	ShipmentID                  pulid.ID
	BillType                    billingqueue.BillType
	MarkCompletedReadyToInvoice bool
	Actor                       *permissionports.RequestActor
	SuppressExceptionNotices    bool
	// Cache holds the answers that do not vary by shipment, so a run of five
	// thousand does not ask the same organization-level question five thousand
	// times. It lives for exactly one bulk call.
	Cache *billingReadinessCache
}

type billingTransferAttempt struct {
	entity      *shipment.Shipment
	readiness   *shipmentports.ShipmentBillingReadiness
	item        *billingqueue.BillingQueueItem
	items       []*billingqueue.BillingQueueItem
	markedReady bool
	failure     shipmentports.BillingTransferFailureCode
	err         error
}

func (a *billingTransferAttempt) fail(
	code shipmentports.BillingTransferFailureCode,
	err error,
) *billingTransferAttempt {
	a.failure = code
	a.err = err
	return a
}

func (s *service) BulkTransferToBilling(
	ctx context.Context,
	req *shipmentports.BulkTransferShipmentToBillingRequest,
	actor *permissionports.RequestActor,
) (*shipmentports.BulkTransferToBillingResponse, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}
	if s.billingQueueService == nil {
		return nil, errortypes.NewConflictError("Billing queue service is unavailable")
	}

	shipmentIDs := sliceutils.Dedupe(req.ShipmentIDs)
	cache := new(billingReadinessCache)
	response := &shipmentports.BulkTransferToBillingResponse{
		Results:    make([]shipmentports.BulkTransferToBillingResult, 0, len(shipmentIDs)),
		TotalCount: len(shipmentIDs),
	}

	for _, shipmentID := range shipmentIDs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		attempt := s.attemptBillingTransfer(ctx, &billingTransferAttemptParams{
			ShipmentID:                  shipmentID,
			BillType:                    req.BillType,
			MarkCompletedReadyToInvoice: req.MarkCompletedReadyToInvoice,
			Actor:                       actor,
			SuppressExceptionNotices:    req.SuppressExceptionNotifications,
			Cache:                       cache,
		})

		result := newBulkTransferResult(shipmentID, attempt)
		if result.Success {
			response.SuccessCount++
		}
		response.Results = append(response.Results, result)

		// Reported before the loop continues, so a caller watching a long run
		// sees each shipment as it happens rather than only at the end — and
		// still holds every answer given if the request dies partway.
		if req.OnResult != nil {
			req.OnResult(result)
		}
	}

	response.ErrorCount = response.TotalCount - response.SuccessCount

	return response, nil
}

func (s *service) ListBillingTransferCandidateIDs(
	ctx context.Context,
	req *shipmentports.ListBillingTransferCandidateIDsRequest,
) (*shipmentports.BillingTransferCandidateIDsResponse, error) {
	if req.Status != "" && !req.Status.IsBillingTransferCandidate() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Only completed or ready to invoice shipments can be transferred to billing",
		)
	}

	result, err := s.repo.ListBillingTransferCandidateIDs(
		ctx,
		&shipmentports.ListBillingTransferCandidateIDsQuery{
			Filter: req.Filter,
			Status: req.Status,
			Limit:  shipmentports.MaxBillingTransferCandidateIDs,
		},
	)
	if err != nil {
		s.l.Error("failed to list billing transfer candidate ids", zap.Error(err))
		return nil, err
	}

	return &shipmentports.BillingTransferCandidateIDsResponse{
		IDs:        result.IDs,
		TotalCount: result.TotalCount,
		Truncated:  result.TotalCount > len(result.IDs),
	}, nil
}

func (s *service) attemptBillingTransfer(
	ctx context.Context,
	p *billingTransferAttemptParams,
) *billingTransferAttempt {
	attempt := new(billingTransferAttempt)
	tenantInfo := pagination.TenantInfo{
		OrgID: p.Actor.OrganizationID,
		BuID:  p.Actor.BusinessUnitID,
	}
	log := s.l.With(
		zap.String("operation", "TransferToBilling"),
		zap.String("shipmentID", p.ShipmentID.String()),
	)

	entity, err := s.repo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         p.ShipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return attempt.fail(shipmentports.BillingTransferFailureNotFound, err)
		}
		log.Error("failed to get shipment for billing transfer", zap.Error(err))
		return attempt.fail(shipmentports.BillingTransferFailureUnexpected, err)
	}
	attempt.entity = entity

	markReady, code, gateErr := transferGate(entity, p.MarkCompletedReadyToInvoice)
	if gateErr != nil {
		return attempt.fail(code, gateErr)
	}

	readiness, err := s.evaluateBillingReadinessCached(ctx, entity, p.Cache, true)
	if err != nil {
		log.Error("failed to evaluate billing readiness for transfer", zap.Error(err))
		return attempt.fail(shipmentports.BillingTransferFailureUnexpected, err)
	}
	attempt.readiness = readiness
	if !p.SuppressExceptionNotices {
		s.notifyBillingExceptions(ctx, entity, readiness)
	}

	if code, policyErr := billingTransferPolicyViolation(readiness); policyErr != nil {
		return attempt.fail(code, policyErr)
	}

	if markReady {
		updated, marked, markErr := s.markReadyToInvoice(ctx, &markReadyToInvoiceParams{
			ShipmentID:        entity.ID,
			TenantInfo:        tenantInfo,
			Actor:             p.Actor.AuditActor(),
			Comment:           "Shipment marked ready to invoice for billing transfer",
			Entity:            entity,
			RecordStatusEvent: true,
		})
		if markErr != nil {
			if errortypes.IsError(markErr) {
				return attempt.fail(shipmentports.BillingTransferFailureInvalidStatus, markErr)
			}
			log.Error("failed to mark shipment ready to invoice for transfer", zap.Error(markErr))
			return attempt.fail(shipmentports.BillingTransferFailureUnexpected, markErr)
		}
		attempt.markedReady = marked
		entity = updated
		attempt.entity = updated
	}

	billType := p.BillType
	if billType == "" {
		billType = billingqueue.BillTypeInvoice
	}

	transferred, err := s.billingQueueService.TransferToBillingItems(
		ctx,
		&billingqueueports.TransferToBillingRequest{
			ShipmentID:          entity.ID,
			BillType:            billType,
			AutoApprove:         readiness.ShouldAutoApproveBilling,
			AutoApprovePayerIDs: autoApprovePayerIDs(readiness),
			TenantInfo:          tenantInfo,
			DetailedShipment:    entity,
		},
		p.Actor,
	)
	if err != nil {
		if errortypes.IsConflictError(err) {
			return attempt.fail(shipmentports.BillingTransferFailureAlreadyTransferred, err)
		}
		log.Error("failed to transfer shipment to billing", zap.Error(err))
		return attempt.fail(shipmentports.BillingTransferFailureUnexpected, err)
	}
	attempt.item = transferred.Primary
	attempt.items = transferred.Items

	return attempt
}

// autoApprovePayerIDs lists the payers whose own profile clears clean freight
// through the queue, so a split shipment can auto-approve one payer's item while
// another payer's waits for a biller.
func autoApprovePayerIDs(readiness *shipmentports.ShipmentBillingReadiness) []pulid.ID {
	if readiness == nil {
		return nil
	}
	ids := make([]pulid.ID, 0, len(readiness.Payers))
	for _, payer := range readiness.Payers {
		if payer.ShouldAutoApproveBilling {
			ids = append(ids, payer.PayerID)
		}
	}

	return ids
}

func newBulkTransferResult(
	shipmentID pulid.ID,
	attempt *billingTransferAttempt,
) shipmentports.BulkTransferToBillingResult {
	result := shipmentports.BulkTransferToBillingResult{
		ShipmentID:           shipmentID,
		MarkedReadyToInvoice: attempt.markedReady,
		Item:                 attempt.item,
		Items:                attempt.items,
		MissingRequirements:  []shipmentports.ShipmentBillingRequirement{},
		ValidationFailures:   []shipmentports.ShipmentBillingValidation{},
	}
	if attempt.entity != nil {
		result.ProNumber = attempt.entity.ProNumber
	}
	if attempt.readiness != nil {
		result.MissingRequirements = attempt.readiness.MissingRequirements
		result.ValidationFailures = attempt.readiness.ValidationFailures
	}

	if attempt.err == nil {
		result.Success = true
		return result
	}

	result.FailureCode = attempt.failure
	result.Err = attempt.err
	result.Error = unexpectedBillingTransferMessage
	if attempt.failure != shipmentports.BillingTransferFailureUnexpected {
		result.Error = attempt.err.Error()
	}

	return result
}
