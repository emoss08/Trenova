package shipmentbillingtransfer

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentbillingreadiness"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteffects"
	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentinvalidationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Service struct {
	l                   *zap.Logger
	repo                shipmentports.ShipmentRepository
	customerRepo        customerports.CustomerRepository
	documentRepo        documentports.DocumentRepository
	billingRepo         tenantports.BillingControlRepository
	billingQueueService billingqueueports.BillingQueueService
	invalidator         shipmentinvalidationports.ShipmentInvalidator
	effects             *shipmenteffects.Effects
	readiness           *shipmentbillingreadiness.Service
}

// ListAutoTransferCandidateIDs finds the shipments an automatic transfer may
// move into billing now: ready to invoice already, or completed and eligible to
// be marked ready without a person, where the payer's transfer mode is
// automatic and nothing in billing policy would refuse the transfer.
//
// Shipments that fail those checks are left out rather than attempted. A
// manual transfer reports why a shipment could not go; an automatic one runs
// every few minutes, and reporting the same unbillable shipment each time would
// bury the failures worth reading.
func (s *Service) ListAutoTransferCandidateIDs(
	ctx context.Context,
	req *shipmentbillingports.ListAutoTransferCandidateIDsRequest,
) ([]pulid.ID, error) {
	if req.Limit <= 0 {
		return []pulid.ID{}, nil
	}

	scanned, err := s.repo.ListBillingTransferCandidateIDs(
		ctx,
		&shipmentports.ListBillingTransferCandidateIDsQuery{
			Filter:      &pagination.QueryOptions{TenantInfo: req.TenantInfo},
			Limit:       max(req.ScanLimit, req.Limit),
			RecentFirst: true,
		},
	)
	if err != nil {
		return nil, err
	}

	cache := new(shipmentbillingreadiness.BillingReadinessCache)
	eligible := make([]pulid.ID, 0, min(req.Limit, len(scanned.IDs)))
	for _, id := range scanned.IDs {
		if len(eligible) == req.Limit {
			break
		}
		if err = ctx.Err(); err != nil {
			return nil, err
		}

		ok, evalErr := s.autoTransferEligible(ctx, req.TenantInfo, id, cache)
		if evalErr != nil {
			s.l.Warn("skipped a shipment the automatic transfer could not evaluate",
				zap.String("shipmentID", id.String()), zap.Error(evalErr))
			continue
		}
		if ok {
			eligible = append(eligible, id)
		}
	}

	return eligible, nil
}

func (s *Service) autoTransferEligible(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	cache *shipmentbillingreadiness.BillingReadinessCache,
) (bool, error) {
	entity, err := s.repo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if errortypes.IsNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, _, gateErr := transferGate(entity, true); gateErr != nil {
		return false, nil //nolint:nilerr // a shipment the gate refuses is not ready to transfer
	}

	readiness, err := s.readiness.EvaluateBillingReadinessCached(ctx, entity, cache, true)
	if err != nil {
		return false, err
	}

	return readinessAllowsAutoTransfer(entity.Status, readiness), nil
}

func (s *Service) AutoMarkReadyToInvoiceIfEligible(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*shipment.Shipment, error) {
	readiness, err := s.readiness.GetBillingReadiness(ctx, shipmentID, tenantInfo)
	if err != nil {
		s.l.Warn("failed to evaluate shipment billing readiness for auto-mark",
			zap.String("shipmentId", shipmentID.String()),
			zap.Error(err),
		)
		return nil, err
	}

	s.l.Info(
		"evaluated shipment billing readiness for auto-mark",
		zap.String("shipmentId", shipmentID.String()),
		zap.String("shipmentStatus", string(readiness.ShipmentStatus)),
		zap.String(
			"shipmentBillingRequirementEnforcement",
			string(readiness.Policy.ShipmentBillingRequirementEnforcement),
		),
		zap.String("readyToBillAssignmentMode", string(readiness.Policy.ReadyToBillAssignmentMode)),
		zap.String("billingQueueTransferMode", string(readiness.Policy.BillingQueueTransferMode)),
		zap.Bool("canMarkReadyToInvoice", readiness.CanMarkReadyToInvoice),
		zap.Bool("shouldAutoMarkReadyToInvoice", readiness.ShouldAutoMarkReadyToInvoice),
		zap.Bool("shouldAutoTransferToBilling", readiness.ShouldAutoTransferToBilling),
		zap.Int("missingRequirementCount", len(readiness.MissingRequirements)),
		zap.Int("validationFailureCount", len(readiness.ValidationFailures)),
	)

	if !readiness.ShouldAutoMarkReadyToInvoice ||
		!readiness.CanMarkReadyToInvoice ||
		readiness.ShipmentStatus != shipmentvocab.StatusCompleted {
		s.l.Info("shipment not eligible for auto-mark ready to invoice",
			zap.String("shipmentId", shipmentID.String()),
			zap.String("shipmentStatus", string(readiness.ShipmentStatus)),
			zap.Bool("canMarkReadyToInvoice", readiness.CanMarkReadyToInvoice),
			zap.Bool("shouldAutoMarkReadyToInvoice", readiness.ShouldAutoMarkReadyToInvoice),
			zap.Int("missingRequirementCount", len(readiness.MissingRequirements)),
			zap.Int("validationFailureCount", len(readiness.ValidationFailures)),
		)
		return nil, nil //nolint:nilnil // nil result represents an optional absence in this API
	}

	actor := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
	}

	s.l.Info("auto-marking shipment ready to invoice",
		zap.String("shipmentId", shipmentID.String()),
		zap.String("currentStatus", string(readiness.ShipmentStatus)),
	)

	updatedEntity, marked, err := s.markReadyToInvoice(ctx, &markReadyToInvoiceParams{
		ShipmentID: shipmentID,
		TenantInfo: tenantInfo,
		Actor:      actor.AuditActor(),
		Comment:    "Shipment auto-marked ready to invoice",
	})
	if err != nil {
		return nil, err
	}
	if !marked {
		s.l.Info("shipment already past auto-mark target status",
			zap.String("shipmentId", shipmentID.String()),
			zap.String("shipmentStatus", string(updatedEntity.Status)),
		)
		return updatedEntity, nil
	}

	if readiness.ShouldAutoTransferToBilling {
		s.autoTransferToBillingQueue(ctx, updatedEntity, actor)
	}

	return updatedEntity, nil
}

func (s *Service) markReadyToInvoice(
	ctx context.Context,
	p *markReadyToInvoiceParams,
) (*shipment.Shipment, bool, error) {
	entity, err := s.shipmentForMarkReady(ctx, p)
	if err != nil {
		return nil, false, err
	}

	switch entity.Status { //nolint:exhaustive // only the billing lifecycle statuses decide this
	case shipmentvocab.StatusReadyToInvoice, shipmentvocab.StatusInvoiced:
		return entity, false, nil
	case shipmentvocab.StatusCompleted:
	default:
		return nil, false, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Shipment must be completed before it can be marked ready to invoice",
		)
	}

	previousEntity := *entity
	entity.Status = shipmentvocab.StatusReadyToInvoice
	now := timeutils.NowUnix()
	entity.MarkedReadyToBillAt = &now

	updatedEntity, err := s.repo.MarkReadyToInvoice(ctx, entity)
	if err != nil {
		*entity = previousEntity
		return nil, false, err
	}

	if !p.RecordStatusEvent {
		if err = s.effects.RecomputeOrdersForShipments(
			ctx,
			p.TenantInfo,
			[]*shipment.Shipment{updatedEntity},
		); err != nil {
			s.l.Warn("failed to recompute order after marking ready to invoice", zap.Error(err))
		}
	}

	if err = s.effects.LogShipmentAction(
		updatedEntity,
		p.Actor,
		permission.OpUpdate,
		&previousEntity,
		updatedEntity,
		auditservice.WithComment(p.Comment),
		auditservice.WithDiff(&previousEntity, updatedEntity),
	); err != nil {
		s.l.Error("failed to log mark ready to invoice shipment action", zap.Error(err))
	}

	shipmentinvalidationports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentinvalidationports.ShipmentInvalidationForRecord(
			updatedEntity,
			p.Actor,
			"updated",
			updatedEntity,
		),
	)

	if p.RecordStatusEvent {
		s.effects.EmitStatusChangeEvent(ctx, &previousEntity, updatedEntity, p.Actor)
	}

	return updatedEntity, true, nil
}

func (s *Service) shipmentForMarkReady(
	ctx context.Context,
	p *markReadyToInvoiceParams,
) (*shipment.Shipment, error) {
	if e := p.Entity; e != nil && e.ID == p.ShipmentID &&
		e.OrganizationID == p.TenantInfo.OrgID && e.BusinessUnitID == p.TenantInfo.BuID {
		return e, nil
	}

	return s.repo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         p.ShipmentID,
		TenantInfo: p.TenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
}

func (s *Service) TransferToBilling(
	ctx context.Context,
	req *shipmentbillingports.TransferShipmentToBillingRequest,
	actor *permissionports.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	if s.billingQueueService == nil {
		return nil, errortypes.NewConflictError("Billing queue service is unavailable")
	}

	attempt := s.attemptBillingTransfer(ctx, &billingTransferAttemptParams{
		ShipmentID: req.ShipmentID,
		BillType:   req.BillType,
		Actor:      actor,
	})
	if attempt.err != nil {
		return nil, attempt.err
	}

	return attempt.item, nil
}

// TransferToBillingItems queues a shipment and returns every payer's item.
func (s *Service) TransferToBillingItems(
	ctx context.Context,
	req *shipmentbillingports.TransferShipmentToBillingRequest,
	actor *permissionports.RequestActor,
) (*billingqueueports.TransferToBillingResult, error) {
	if s.billingQueueService == nil {
		return nil, errortypes.NewConflictError("Billing queue service is unavailable")
	}

	attempt := s.attemptBillingTransfer(ctx, &billingTransferAttemptParams{
		ShipmentID: req.ShipmentID,
		BillType:   req.BillType,
		Actor:      actor,
	})
	if attempt.err != nil {
		return nil, attempt.err
	}

	return &billingqueueports.TransferToBillingResult{
		Items:   attempt.items,
		Primary: attempt.item,
	}, nil
}

func (s *Service) autoTransferToBillingQueue(
	ctx context.Context,
	entity *shipment.Shipment,
	actor *permissionports.RequestActor,
) {
	if _, err := s.TransferToBilling(ctx, &shipmentbillingports.TransferShipmentToBillingRequest{
		ShipmentID: entity.ID,
		BillType:   billingqueue.BillTypeInvoice,
	}, actor); err != nil {
		s.l.Warn("failed to auto-transfer shipment to billing queue",
			zap.String("shipmentId", entity.ID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) notifyBillingExceptions(
	ctx context.Context,
	entity *shipment.Shipment,
	readiness *shipmentbillingports.ShipmentBillingReadiness,
) {
	requirementIssues := shipmentbillingreadiness.HasShipmentRequirementIssues(readiness)
	rateIssues := shipmentbillingreadiness.HasRateIssues(readiness)
	if !requirementIssues && !rateIssues {
		return
	}

	s.effects.CreateNotification(ctx, &notification.Notification{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: &entity.BusinessUnitID,
		EventType:      "billing_exception_recorded",
		Priority:       notification.PriorityMedium,
		Channel:        notification.ChannelGlobal,
		Title:          "Billing exception recorded",
		Message:        "Shipment billing validation requires attention before billing can continue cleanly.",
		Data: map[string]any{
			"shipmentBillingRequirementEnforcement": readiness.Policy.ShipmentBillingRequirementEnforcement,
			"rateValidationEnforcement":             readiness.Policy.RateValidationEnforcement,
			"billingExceptionDisposition":           readiness.Policy.BillingExceptionDisposition,
			"missingRequirementCount":               len(readiness.MissingRequirements),
			"validationFailureCount":                len(readiness.ValidationFailures),
		},
		RelatedEntities: map[string]any{
			"shipmentId": entity.ID.String(),
		},
		Source: "shipmentservice.TransferToBilling",
	})
}

func (s *Service) PlanBillingTransfers(
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

func (s *Service) ListBillingTransferCandidates(
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

// planTransfers decides each shipment by the checks attemptBillingTransfer
// makes, reading the shipments, their payers, the policy, their documents
// and their service failures once for the whole set.
func (s *Service) planTransfers(
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
	pending := make([]shipmentbillingreadiness.PendingReadiness, 0, len(ids))
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
		pending = append(pending, shipmentbillingreadiness.PendingReadiness{
			Index:      idx,
			Entity:     entity,
			Resolution: resolution,
			MarkReady:  markReady,
		})
	}

	sources, err := s.readiness.ReadinessSourcesFor(ctx, tenantInfo, pending)
	if err != nil {
		return nil, err
	}
	for _, entry := range pending {
		readiness := s.readiness.ReadinessFrom(entry.Entity, entry.Resolution, sources)
		decide(&decisions[entry.Index], readiness, entry.MarkReady)
	}
	if err = s.stampCurrencies(ctx, tenantInfo, decisions, pending, sources.Payers); err != nil {
		return nil, err
	}

	return summarize(decisions), nil
}

func (s *Service) stampCurrencies(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	decisions []shipmentbillingports.BillingTransferDecision,
	pending []shipmentbillingreadiness.PendingReadiness,
	payers map[pulid.ID]*customer.Customer,
) error {
	billedTo := make([]pulid.ID, len(decisions))
	for idx := range decisions {
		billedTo[idx] = decisions[idx].CustomerID
	}
	for _, entry := range pending {
		billedTo[entry.Index] = entry.Resolution.DefaultPayerID
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

func (s *Service) BulkTransferToBilling(
	ctx context.Context,
	req *shipmentbillingports.BulkTransferShipmentToBillingRequest,
	actor *permissionports.RequestActor,
) (*shipmentbillingports.BulkTransferToBillingResponse, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}
	if s.billingQueueService == nil {
		return nil, errortypes.NewConflictError("Billing queue service is unavailable")
	}

	shipmentIDs := sliceutils.Dedupe(req.ShipmentIDs)
	cache := new(shipmentbillingreadiness.BillingReadinessCache)
	response := &shipmentbillingports.BulkTransferToBillingResponse{
		Results:    make([]shipmentbillingports.BulkTransferToBillingResult, 0, len(shipmentIDs)),
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

func (s *Service) ListBillingTransferCandidateIDs(
	ctx context.Context,
	req *shipmentbillingports.ListBillingTransferCandidateIDsRequest,
) (*shipmentbillingports.BillingTransferCandidateIDsResponse, error) {
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
			Limit:  shipmentbillingports.MaxBillingTransferCandidateIDs,
		},
	)
	if err != nil {
		s.l.Error("failed to list billing transfer candidate ids", zap.Error(err))
		return nil, err
	}

	return &shipmentbillingports.BillingTransferCandidateIDsResponse{
		IDs:        result.IDs,
		TotalCount: result.TotalCount,
		Truncated:  result.TotalCount > len(result.IDs),
	}, nil
}

func (s *Service) attemptBillingTransfer(
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
			return attempt.fail(shipmentbillingports.BillingTransferFailureNotFound, err)
		}
		log.Error("failed to get shipment for billing transfer", zap.Error(err))
		return attempt.fail(shipmentbillingports.BillingTransferFailureUnexpected, err)
	}
	attempt.entity = entity

	markReady, code, gateErr := transferGate(entity, p.MarkCompletedReadyToInvoice)
	if gateErr != nil {
		return attempt.fail(code, gateErr)
	}

	readiness, err := s.readiness.EvaluateBillingReadinessCached(ctx, entity, p.Cache, true)
	if err != nil {
		log.Error("failed to evaluate billing readiness for transfer", zap.Error(err))
		return attempt.fail(shipmentbillingports.BillingTransferFailureUnexpected, err)
	}
	attempt.readiness = readiness
	if !p.SuppressExceptionNotices {
		s.notifyBillingExceptions(ctx, entity, readiness)
	}

	if policyCode, policyErr := billingTransferPolicyViolation(readiness); policyErr != nil {
		return attempt.fail(policyCode, policyErr)
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
				return attempt.fail(
					shipmentbillingports.BillingTransferFailureInvalidStatus,
					markErr,
				)
			}
			log.Error("failed to mark shipment ready to invoice for transfer", zap.Error(markErr))
			return attempt.fail(shipmentbillingports.BillingTransferFailureUnexpected, markErr)
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
			return attempt.fail(shipmentbillingports.BillingTransferFailureAlreadyTransferred, err)
		}
		log.Error("failed to transfer shipment to billing", zap.Error(err))
		return attempt.fail(shipmentbillingports.BillingTransferFailureUnexpected, err)
	}
	attempt.item = transferred.Primary
	attempt.items = transferred.Items

	return attempt
}

type Params struct {
	fx.In

	Logger              *zap.Logger
	Repo                shipmentports.ShipmentRepository
	CustomerRepo        customerports.CustomerRepository
	DocumentRepo        documentports.DocumentRepository
	BillingRepo         tenantports.BillingControlRepository
	BillingQueueService billingqueueports.BillingQueueService `optional:"true"`
	Invalidator         shipmentinvalidationports.ShipmentInvalidator
	Effects             *shipmenteffects.Effects
	Readiness           *shipmentbillingreadiness.Service
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		l:                   p.Logger.Named("service.shipment"),
		repo:                p.Repo,
		customerRepo:        p.CustomerRepo,
		documentRepo:        p.DocumentRepo,
		billingRepo:         p.BillingRepo,
		billingQueueService: p.BillingQueueService,
		invalidator:         p.Invalidator,
		effects:             p.Effects,
		readiness:           p.Readiness,
	}
}

var _ shipmentserviceports.ShipmentBillingTransferService = (*Service)(nil)
