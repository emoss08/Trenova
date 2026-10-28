package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/zap"
)

// transferGate is what a transfer checks before it reads a shipment's
// readiness: the shipment is outside the queue and ready to invoice, or
// completed and the caller asked for it to be marked ready first.
func transferGate(
	entity *shipment.Shipment,
	markCompleted bool,
) (markReady bool, code shipmentbillingports.BillingTransferFailureCode, err error) {
	if !entity.BillingTransferStatus.IsOutsideBillingQueue() {
		return false, shipmentbillingports.BillingTransferFailureAlreadyTransferred,
			errortypes.NewValidationError(
				"billingTransferStatus",
				errortypes.ErrInvalidOperation,
				"Shipment has already been transferred to billing",
			)
	}

	markReady = markCompleted && entity.Status == shipment.StatusCompleted
	if entity.Status != shipment.StatusReadyToInvoice && !markReady {
		return false, shipmentbillingports.BillingTransferFailureInvalidStatus, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrInvalidOperation,
			"Shipment must be in ReadyToInvoice status to transfer to billing",
		)
	}

	return markReady, "", nil
}

func (s *service) PlanBillingTransfers(
	ctx context.Context,
	req *shipmentbillingports.PlanBillingTransfersRequest,
) (*shipmentbillingports.BillingTransferPlan, error) {
	ids := sliceutils.Dedupe(req.ShipmentIDs)
	switch {
	case len(ids) == 0:
		return nil, errortypes.NewValidationError(
			"shipmentIds",
			errortypes.ErrRequired,
			"At least one shipment ID is required",
		)
	case len(ids) > shipmentbillingports.MaxBillingTransferCandidateIDs:
		return nil, errortypes.NewValidationError(
			"shipmentIds",
			errortypes.ErrInvalid,
			"A transfer can include at most {0} shipments",
			shipmentbillingports.MaxBillingTransferCandidateIDs,
		)
	}

	return s.planTransfers(ctx, req.TenantInfo, ids, req.MarkCompletedReadyToInvoice)
}

func (s *service) ListBillingTransferCandidates(
	ctx context.Context,
	req *shipmentbillingports.ListBillingTransferCandidatesRequest,
) (*shipmentbillingports.BillingTransferCandidates, error) {
	if req.Status != "" && !req.Status.IsBillingTransferCandidate() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Only completed or ready to invoice shipments can be transferred to billing",
		)
	}

	limit := req.Filter.Pagination.SafeLimit()
	offset := req.Filter.Pagination.SafeOffset()
	candidates, err := s.repo.ListBillingTransferCandidateIDs(
		ctx,
		&shipmentports.ListBillingTransferCandidateIDsQuery{
			Filter: req.Filter,
			Status: req.Status,
			Limit:  min(offset+limit+1, shipmentbillingports.MaxBillingTransferCandidateIDs),
		},
	)
	if err != nil {
		s.l.Error("failed to list billing transfer candidates", zap.Error(err))
		return nil, err
	}

	result := &shipmentbillingports.BillingTransferCandidates{
		Decisions:  []shipmentbillingports.BillingTransferDecision{},
		TotalCount: candidates.TotalCount,
		HasMore:    len(candidates.IDs) > offset+limit,
	}
	if offset >= len(candidates.IDs) {
		return result, nil
	}

	page := candidates.IDs[offset:min(offset+limit, len(candidates.IDs))]
	plan, err := s.planTransfers(
		ctx,
		req.Filter.TenantInfo,
		page,
		req.MarkCompletedReadyToInvoice,
	)
	if err != nil {
		return nil, err
	}
	result.Decisions = plan.Decisions

	return result, nil
}

// pendingReadiness is a shipment that passed the gate and waits on its
// readiness.
type pendingReadiness struct {
	index      int
	entity     *shipment.Shipment
	resolution *chargeallocation.ShareResolution
	markReady  bool
}

// planTransfers decides each shipment by the checks attemptBillingTransfer
// makes, reading the shipments, their payers, the policy, their documents
// and their service failures once for the whole set.
func (s *service) planTransfers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
	markCompleted bool,
) (*shipmentbillingports.BillingTransferPlan, error) {
	if s.customerRepo == nil || s.documentRepo == nil || s.billingRepo == nil {
		return nil, errortypes.NewConflictError("Shipment billing readiness service is unavailable")
	}

	shipments, err := s.repo.GetByIDs(ctx, &shipmentports.GetShipmentsByIDsRequest{
		TenantInfo:     tenantInfo,
		ShipmentIDs:    ids,
		IncludeCharges: true,
	})
	if err != nil {
		return nil, err
	}
	byID := make(map[pulid.ID]*shipment.Shipment, len(shipments))
	for _, entity := range shipments {
		if entity != nil {
			byID[entity.ID] = entity
		}
	}

	decisions := make([]shipmentbillingports.BillingTransferDecision, len(ids))
	pending := make([]pendingReadiness, 0, len(ids))
	for idx, id := range ids {
		entity := byID[id]
		decisions[idx] = decisionFor(id, entity)
		if entity == nil {
			refuse(&decisions[idx], shipmentbillingports.BillingTransferFailureNotFound,
				errortypes.NewNotFoundError("Shipment not found"))
			continue
		}

		markReady, code, gateErr := transferGate(entity, markCompleted)
		if gateErr != nil {
			refuse(&decisions[idx], code, gateErr)
			continue
		}

		resolution, shareErr := chargeallocation.ResolveShares(
			ctx,
			entity,
			entity.ChargeAllocations,
		)
		if shareErr != nil {
			refuse(&decisions[idx], shipmentbillingports.BillingTransferFailureUnexpected, shareErr)
			continue
		}
		pending = append(pending, pendingReadiness{
			index:      idx,
			entity:     entity,
			resolution: resolution,
			markReady:  markReady,
		})
	}

	sources, err := s.readinessSourcesFor(ctx, tenantInfo, pending)
	if err != nil {
		return nil, err
	}
	for _, entry := range pending {
		readiness := s.readinessFrom(entry.entity, entry.resolution, sources)
		decide(&decisions[entry.index], readiness, entry.markReady)
	}
	if err = s.stampCurrencies(ctx, tenantInfo, decisions, pending, sources.payers); err != nil {
		return nil, err
	}

	return summarize(decisions), nil
}

func (s *service) stampCurrencies(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	decisions []shipmentbillingports.BillingTransferDecision,
	pending []pendingReadiness,
	payers map[pulid.ID]*customer.Customer,
) error {
	billedTo := make([]pulid.ID, len(decisions))
	for idx := range decisions {
		billedTo[idx] = decisions[idx].CustomerID
	}
	for _, entry := range pending {
		billedTo[entry.index] = entry.resolution.DefaultPayerID
	}

	missing := make([]pulid.ID, 0, len(decisions))
	for _, id := range billedTo {
		if _, known := payers[id]; id.IsNotNil() && !known {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		loaded, err := s.customerRepo.GetByIDs(ctx, customerports.GetCustomersByIDsRequest{
			TenantInfo:  tenantInfo,
			CustomerIDs: sliceutils.Dedupe(missing),
			CustomerFilterOptions: customerports.CustomerFilterOptions{
				IncludeBillingProfile: true,
			},
		})
		if err != nil {
			return err
		}
		for _, entry := range loaded {
			if entry != nil {
				payers[entry.ID] = entry
			}
		}
	}

	for idx := range decisions {
		decisions[idx].CurrencyCode = payers[billedTo[idx]].BillingCurrencyCode()
	}

	return nil
}

func (s *service) readinessSourcesFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	pending []pendingReadiness,
) (*readinessSources, error) {
	sources := &readinessSources{
		payers:    map[pulid.ID]*customer.Customer{},
		documents: make(map[string][]*document.Document, len(pending)),
		failures:  make(map[pulid.ID][]*servicefailure.ServiceFailure, len(pending)),
	}
	if len(pending) == 0 {
		return sources, nil
	}

	payerIDs := make([]pulid.ID, 0, len(pending))
	defaultPayers := make([]pulid.ID, 0, len(pending))
	shipmentIDs := make([]pulid.ID, 0, len(pending))
	resourceIDs := make([]string, 0, len(pending))
	for _, entry := range pending {
		payerIDs = append(payerIDs, entry.resolution.PayerIDs()...)
		defaultPayers = append(defaultPayers, entry.resolution.DefaultPayerID)
		shipmentIDs = append(shipmentIDs, entry.entity.ID)
		resourceIDs = append(resourceIDs, entry.entity.ID.String())
	}

	payers, err := s.customerRepo.GetByIDs(ctx, customerports.GetCustomersByIDsRequest{
		TenantInfo:  tenantInfo,
		CustomerIDs: sliceutils.Dedupe(payerIDs),
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeBillingProfile: true,
		},
	})
	if err != nil {
		return nil, err
	}
	sources.payers = payersByID(payers)
	if err = s.completePayers(
		ctx,
		tenantInfo,
		sources.payers,
		sliceutils.Dedupe(defaultPayers)...,
	); err != nil {
		return nil, err
	}

	if sources.control, err = s.billingControlFor(
		ctx,
		tenantInfo.OrgID,
		new(billingReadinessCache),
	); err != nil {
		return nil, err
	}

	docs, err := s.documentRepo.GetByResourceIDs(
		ctx,
		&documentports.GetDocumentsByResourceIDsRequest{
			TenantInfo:   tenantInfo,
			ResourceType: "shipment",
			ResourceIDs:  resourceIDs,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, doc := range docs {
		if doc != nil {
			sources.documents[doc.ResourceID] = append(sources.documents[doc.ResourceID], doc)
		}
	}

	s.readServiceFailures(ctx, tenantInfo, shipmentIDs, sources)

	return sources, nil
}

// readServiceFailures adds the unresolved service failures, which only warn;
// a failure to read them is logged and costs the warning, as it does for one
// shipment.
func (s *service) readServiceFailures(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
	sources *readinessSources,
) {
	if s.serviceFailureRepo == nil {
		return
	}

	failures, err := s.serviceFailureRepo.ListUnresolvedByShipmentIDs(
		ctx,
		&servicefailureports.ServiceFailuresByShipmentIDsRequest{
			TenantInfo:  tenantInfo,
			ShipmentIDs: shipmentIDs,
		},
	)
	if err != nil {
		s.l.Warn("failed to read service failures for billing transfer plan", zap.Error(err))
		return
	}
	for _, failure := range failures {
		if failure != nil {
			sources.failures[failure.ShipmentID] = append(
				sources.failures[failure.ShipmentID],
				failure,
			)
		}
	}
}

func decisionFor(
	id pulid.ID,
	entity *shipment.Shipment,
) shipmentbillingports.BillingTransferDecision {
	decision := shipmentbillingports.BillingTransferDecision{
		ShipmentID:          id,
		MissingRequirements: []shipmentbillingports.ShipmentBillingRequirement{},
		ValidationFailures:  []shipmentbillingports.ShipmentBillingValidation{},
		Warnings:            []shipmentbillingports.ShipmentBillingWarning{},
	}
	if entity == nil {
		return decision
	}

	decision.ProNumber = entity.ProNumber
	decision.Status = entity.Status
	decision.BillingTransferStatus = entity.BillingTransferStatus
	decision.CustomerID = entity.CustomerID
	decision.TotalCharge = entity.TotalChargeAmount
	decision.DeliveredAt = entity.ActualDeliveryDate
	decision.Version = entity.Version
	if entity.Customer != nil {
		decision.CustomerName = entity.Customer.Name
	}

	return decision
}

func refuse(
	decision *shipmentbillingports.BillingTransferDecision,
	code shipmentbillingports.BillingTransferFailureCode,
	err error,
) {
	decision.Outcome = shipmentbillingports.BillingTransferOutcomeRefused
	if code == shipmentbillingports.BillingTransferFailureReturnToOperations {
		decision.Outcome = shipmentbillingports.BillingTransferOutcomeReturnToOperations
	}
	decision.FailureCode = code
	decision.Reason = err.Error()
}

func decide(
	decision *shipmentbillingports.BillingTransferDecision,
	readiness *shipmentbillingports.ShipmentBillingReadiness,
	markReady bool,
) {
	decision.MissingRequirements = readiness.MissingRequirements
	decision.ValidationFailures = readiness.ValidationFailures
	decision.Warnings = readiness.Warnings
	decision.PayerCount = len(readiness.Payers)
	decision.AutoApprove = readiness.ShouldAutoApproveBilling ||
		len(autoApprovePayerIDs(readiness)) > 0
	if decision.CustomerName == "" {
		decision.CustomerName = primaryPayer(readiness)
	}

	if code, err := billingTransferPolicyViolation(readiness); err != nil {
		refuse(decision, code, err)
		decision.AutoApprove = false

		return
	}

	decision.Outcome = shipmentbillingports.BillingTransferOutcomeTransfer
	if markReady {
		decision.Outcome = shipmentbillingports.BillingTransferOutcomeMarkReadyAndTransfer
	}
}

func primaryPayer(readiness *shipmentbillingports.ShipmentBillingReadiness) string {
	for _, payer := range readiness.Payers {
		if payer.IsPrimary {
			return payer.PayerName
		}
	}

	return ""
}

func summarize(
	decisions []shipmentbillingports.BillingTransferDecision,
) *shipmentbillingports.BillingTransferPlan {
	plan := &shipmentbillingports.BillingTransferPlan{Decisions: decisions}
	for idx := range decisions {
		switch outcome := decisions[idx].Outcome; {
		case outcome.Transfers():
			plan.Transfer++
		case outcome == shipmentbillingports.BillingTransferOutcomeReturnToOperations:
			plan.Returned++
		default:
			plan.Refused++
		}
	}

	return plan
}
