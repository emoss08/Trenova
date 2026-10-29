//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package shipmentservice

import (
	"context"
	"slices"
	"sort"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentinvalidationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type rateValidationResult struct {
	Code    string
	Field   string
	Message string
}

func (s *service) GetBillingReadiness(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*shipmentbillingports.ShipmentBillingReadiness, error) {
	entity, err := s.repo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	return s.evaluateBillingReadiness(ctx, entity)
}

//nolint:funlen // existing workflow or route registration is intentionally kept together
func (s *service) AutoMarkReadyToInvoiceIfEligible(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*shipment.Shipment, error) {
	readiness, err := s.GetBillingReadiness(ctx, shipmentID, tenantInfo)
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
		readiness.ShipmentStatus != shipment.StatusCompleted {
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

type markReadyToInvoiceParams struct {
	ShipmentID        pulid.ID
	TenantInfo        pagination.TenantInfo
	Actor             kernelports.AuditActor
	Comment           string
	Entity            *shipment.Shipment
	RecordStatusEvent bool
}

func (s *service) markReadyToInvoice(
	ctx context.Context,
	p *markReadyToInvoiceParams,
) (*shipment.Shipment, bool, error) {
	entity, err := s.shipmentForMarkReady(ctx, p)
	if err != nil {
		return nil, false, err
	}

	switch entity.Status { //nolint:exhaustive // only the billing lifecycle statuses decide this
	case shipment.StatusReadyToInvoice, shipment.StatusInvoiced:
		return entity, false, nil
	case shipment.StatusCompleted:
	default:
		return nil, false, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Shipment must be completed before it can be marked ready to invoice",
		)
	}

	previousEntity := *entity
	entity.Status = shipment.StatusReadyToInvoice
	now := timeutils.NowUnix()
	entity.MarkedReadyToBillAt = &now

	updatedEntity, err := s.repo.MarkReadyToInvoice(ctx, entity)
	if err != nil {
		*entity = previousEntity
		return nil, false, err
	}

	if !p.RecordStatusEvent {
		if err = s.recomputeOrdersForShipments(
			ctx,
			p.TenantInfo,
			[]*shipment.Shipment{updatedEntity},
		); err != nil {
			s.l.Warn("failed to recompute order after marking ready to invoice", zap.Error(err))
		}
	}

	if err = s.logShipmentAction(
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
		s.emitStatusChangeEvent(ctx, &previousEntity, updatedEntity, p.Actor)
	}

	return updatedEntity, true, nil
}

func (s *service) shipmentForMarkReady(
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

func (s *service) validateBillingReadinessForStatusChange(
	ctx context.Context,
	entity *shipment.Shipment,
) *errortypes.MultiError {
	if entity.Status != shipment.StatusReadyToInvoice ||
		s.customerRepo == nil ||
		s.documentRepo == nil ||
		s.billingRepo == nil {
		return nil
	}

	readiness, err := s.evaluateBillingReadiness(ctx, entity)
	if err != nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add(
			"status",
			errortypes.ErrInvalid,
			"Unable to evaluate shipment billing readiness",
		)
		return multiErr
	}

	if readiness.CanMarkReadyToInvoice {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		"status",
		errortypes.ErrInvalidOperation,
		"Shipment cannot be marked ready to invoice until billing requirements are satisfied",
	)

	for _, failure := range readiness.ValidationFailures {
		field := failure.Field
		if field == "" {
			field = "status"
		}

		multiErr.Add(field, errortypes.ErrInvalidOperation, failure.Message)
	}

	for _, requirement := range readiness.MissingRequirements {
		multiErr.Add(
			"status",
			errortypes.ErrInvalidOperation,
			"Missing required document: {0}", requirement.DocumentTypeName,
		)
	}

	return multiErr
}

// billingReadinessCache carries what every shipment in one bulk transfer shares.
// It is scoped to a single call and never outlives it, so it cannot go stale
// against a policy change the way a process-wide cache could.
type billingReadinessCache struct {
	billingControl      *billingcontrol.BillingControl
	billingControlKnown bool
}

// billingControlFor reads the organization's billing policy, at most once per
// bulk transfer. The policy is one row per organization and every shipment in a
// run asks the same question of it.
func (s *service) billingControlFor(
	ctx context.Context,
	orgID pulid.ID,
	cache *billingReadinessCache,
) (*billingcontrol.BillingControl, error) {
	if cache != nil && cache.billingControlKnown {
		return cache.billingControl, nil
	}

	control, err := s.billingRepo.GetByOrgID(ctx, orgID)
	switch {
	case err == nil:
	case errortypes.IsNotFoundError(err):
		// A tenant that has not configured billing has no policy, which is an
		// answer worth caching rather than re-asking for every shipment.
		control = nil
	default:
		return nil, err
	}

	if cache != nil {
		cache.billingControl = control
		cache.billingControlKnown = true
	}

	return control, nil
}

func (s *service) evaluateBillingReadiness(
	ctx context.Context,
	entity *shipment.Shipment,
) (*shipmentbillingports.ShipmentBillingReadiness, error) {
	return s.evaluateBillingReadinessCached(ctx, entity, nil, false)
}

// evaluateBillingReadinessCached is the readiness check with somewhere to put
// the answers that do not vary by shipment.
//
// A bulk transfer asks the same organization-level questions once per shipment,
// which is the difference between one query and five thousand. A nil cache means
// a single shipment is being evaluated on its own, and nothing is shared.
func (s *service) evaluateBillingReadinessCached(
	ctx context.Context,
	entity *shipment.Shipment,
	cache *billingReadinessCache,
	detailsLoaded bool,
) (*shipmentbillingports.ShipmentBillingReadiness, error) {
	if s.customerRepo == nil || s.documentRepo == nil || s.billingRepo == nil {
		return nil, errortypes.NewConflictError("Shipment billing readiness service is unavailable")
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	resolution, err := s.resolvePayerShares(ctx, entity, tenantInfo, detailsLoaded)
	if err != nil {
		return nil, err
	}

	payers, err := s.customerRepo.GetByIDs(ctx, customerports.GetCustomersByIDsRequest{
		TenantInfo:  tenantInfo,
		CustomerIDs: resolution.PayerIDs(),
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeBillingProfile: true,
		},
	})
	if err != nil {
		return nil, err
	}

	sources := &readinessSources{
		payers:    payersByID(payers),
		documents: make(map[string][]*document.Document, 1),
		failures:  make(map[pulid.ID][]*servicefailure.ServiceFailure, 1),
	}
	err = s.completePayers(ctx, tenantInfo, sources.payers, resolution.DefaultPayerID)
	if err != nil {
		return nil, err
	}

	sources.control, err = s.billingControlFor(ctx, entity.OrganizationID, cache)
	if err != nil {
		return nil, err
	}

	docs, err := s.documentRepo.GetByResourceID(ctx, &documentports.GetDocumentsByResourceRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		ResourceID:   entity.ID.String(),
		ResourceType: "shipment",
	})
	if err != nil {
		return nil, err
	}
	sources.documents[entity.ID.String()] = docs

	if failures, failureErr := s.unresolvedServiceFailures(ctx, entity); failureErr != nil {
		s.l.Warn("failed to apply service failure billing context",
			zap.String("shipmentId", entity.ID.String()),
			zap.Error(failureErr),
		)
	} else {
		sources.failures[entity.ID] = failures
	}

	return s.readinessFrom(entity, resolution, sources), nil
}

// readinessSources is what a readiness evaluation reads beyond the shipment
// itself. One shipment reads them for itself; a plan over many reads each
// once for all of them, and both decide from the same function.
type readinessSources struct {
	control   *billingcontrol.BillingControl
	payers    map[pulid.ID]*customer.Customer
	documents map[string][]*document.Document
	failures  map[pulid.ID][]*servicefailure.ServiceFailure
}

func payersByID(payers []*customer.Customer) map[pulid.ID]*customer.Customer {
	byID := make(map[pulid.ID]*customer.Customer, len(payers))
	for _, payer := range payers {
		if payer != nil {
			byID[payer.ID] = payer
		}
	}

	return byID
}

// completePayers reads a default payer the batched read did not return, which
// readiness cannot be decided without.
func (s *service) completePayers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	payers map[pulid.ID]*customer.Customer,
	defaultPayerIDs ...pulid.ID,
) error {
	for _, payerID := range defaultPayerIDs {
		if _, ok := payers[payerID]; ok {
			continue
		}
		payer, err := s.customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
			ID:         payerID,
			TenantInfo: tenantInfo,
			CustomerFilterOptions: customerports.CustomerFilterOptions{
				IncludeBillingProfile: true,
			},
		})
		if err != nil {
			return err
		}
		payers[payer.ID] = payer
	}

	return nil
}

// readinessFrom decides a shipment's billing readiness from what was read for
// it. It reads nothing itself.
func (s *service) readinessFrom(
	entity *shipment.Shipment,
	resolution *chargeallocation.ShareResolution,
	sources *readinessSources,
) *shipmentbillingports.ShipmentBillingReadiness {
	var profile *customer.CustomerBillingProfile
	if payer := sources.payers[resolution.DefaultPayerID]; payer != nil {
		profile = payer.BillingProfile
	}
	docs := sources.documents[entity.ID.String()]

	readiness := buildShipmentBillingReadiness(entity, profile, sources.control, docs)
	applyPayerReadiness(readiness, entity, resolution, sources.payers, docs)
	applyUnresolvedServiceFailures(readiness, sources.failures[entity.ID])
	s.l.Debug(
		"evaluated shipment billing readiness",
		zap.String("shipmentId", entity.ID.String()),
		zap.String("shipmentStatus", string(entity.Status)),
		zap.String(
			"shipmentBillingRequirementEnforcement",
			string(readiness.Policy.ShipmentBillingRequirementEnforcement),
		),
		zap.String("readyToBillAssignmentMode", string(readiness.Policy.ReadyToBillAssignmentMode)),
		zap.String("billingQueueTransferMode", string(readiness.Policy.BillingQueueTransferMode)),
		zap.Bool("canMarkReadyToInvoice", readiness.CanMarkReadyToInvoice),
		zap.Bool("shouldAutoMarkReadyToInvoice", readiness.ShouldAutoMarkReadyToInvoice),
		zap.Bool("shouldAutoTransferToBilling", readiness.ShouldAutoTransferToBilling),
		zap.Int("requirementCount", len(readiness.Requirements)),
		zap.Int("missingRequirementCount", len(readiness.MissingRequirements)),
		zap.Int("validationFailureCount", len(readiness.ValidationFailures)),
	)

	return readiness
}

func (s *service) unresolvedServiceFailures(
	ctx context.Context,
	entity *shipment.Shipment,
) ([]*servicefailure.ServiceFailure, error) {
	if s.serviceFailureRepo == nil || entity == nil {
		return nil, nil
	}

	return s.serviceFailureRepo.ListUnresolvedByShipment(
		ctx,
		&servicefailureports.ServiceFailuresByShipmentRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			ShipmentID: entity.ID,
		},
	)
}

func applyUnresolvedServiceFailures(
	readiness *shipmentbillingports.ShipmentBillingReadiness,
	unresolved []*servicefailure.ServiceFailure,
) {
	if readiness == nil || len(unresolved) == 0 {
		return
	}

	ids := make([]string, 0, len(unresolved))
	for _, failure := range unresolved {
		if failure == nil {
			continue
		}
		ids = append(ids, failure.ID.String())
	}
	if len(ids) == 0 {
		return
	}
	readiness.ServiceFailureContext.HasUnresolved = true
	readiness.ServiceFailureContext.UnresolvedCount = len(ids)
	readiness.ServiceFailureContext.ServiceFailureIDs = ids
	readiness.Warnings = append(readiness.Warnings, shipmentbillingports.ShipmentBillingWarning{
		Code:    "unresolved_service_failures",
		Message: "Shipment has unresolved service failures",
		Context: map[string]any{
			"serviceFailureIds": ids,
			"unresolvedCount":   len(ids),
		},
	})
}

// resolvePayerShares divides the shipment among its payers, reloading it with
// charges and allocations when the caller handed over a bare header.
func (s *service) resolvePayerShares(
	ctx context.Context,
	entity *shipment.Shipment,
	tenantInfo pagination.TenantInfo,
	detailsLoaded bool,
) (*chargeallocation.ShareResolution, error) {
	source := entity
	if !detailsLoaded &&
		(entity.AdditionalCharges == nil || entity.ChargeAllocations == nil) &&
		s.repo != nil && entity.ID.IsNotNil() {
		full, err := s.repo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
			ID:         entity.ID,
			TenantInfo: tenantInfo,
			ShipmentOptions: shipmentports.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
		})
		if err != nil {
			return nil, err
		}
		source = full
	}

	resolution, err := chargeallocation.ResolveShares(ctx, source, source.ChargeAllocations)
	if err != nil {
		return nil, err
	}

	return resolution, nil
}

// applyPayerReadiness folds every other payer into the readiness the primary
// payer's profile produced: their document and BOL requirements join the list,
// a payer on credit hold blocks under the same enforcement as a missing
// document, and auto-approval needs every payer's consent.
func applyPayerReadiness(
	readiness *shipmentbillingports.ShipmentBillingReadiness,
	entity *shipment.Shipment,
	resolution *chargeallocation.ShareResolution,
	payersByID map[pulid.ID]*customer.Customer,
	docs []*document.Document,
) {
	if readiness == nil || resolution == nil {
		return
	}
	readiness.Payers = make(
		[]shipmentbillingports.ShipmentBillingPayerReadiness,
		0,
		len(resolution.Shares),
	)

	requirementIssues := hasShipmentRequirementIssues(readiness)
	rateIssues := hasRateIssues(readiness)
	autoApprove := true
	known := make(map[string]struct{}, len(readiness.Requirements))
	for _, requirement := range readiness.Requirements {
		known[requirement.DocumentTypeID] = struct{}{}
	}

	for _, share := range resolution.Shares {
		payer := payersByID[share.PayerID]
		var profile *customer.CustomerBillingProfile
		entry := shipmentbillingports.ShipmentBillingPayerReadiness{
			PayerID:     share.PayerID,
			IsPrimary:   share.PayerID == resolution.DefaultPayerID,
			ShareAmount: share.TotalAmount,
		}
		if payer != nil {
			entry.PayerName = payer.Name
			entry.PayerCode = payer.Code
			profile = payer.BillingProfile
		}
		if profile != nil {
			entry.CreditStatus = profile.CreditStatus
			entry.CreditHold = profile.BlocksBilling()
		}

		if !entry.IsPrimary && profile != nil {
			for _, requirement := range buildDocumentRequirements(profile.DocumentTypes, docs) {
				if _, ok := known[requirement.DocumentTypeID]; ok {
					continue
				}
				known[requirement.DocumentTypeID] = struct{}{}
				readiness.Requirements = append(readiness.Requirements, requirement)
				if !requirement.Satisfied {
					readiness.MissingRequirements = append(
						readiness.MissingRequirements,
						requirement,
					)
				}
			}
			if readiness.Policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelBlock &&
				profile.RequireBOLNumber &&
				entity.BOL == "" &&
				!hasValidationCode(readiness.ValidationFailures, "missing_bol") {
				readiness.ValidationFailures = append(readiness.ValidationFailures,
					shipmentbillingports.ShipmentBillingValidation{
						Field:   "bol",
						Code:    "missing_bol",
						Message: "BOL is required before the shipment can be invoiced",
					})
			}
		}

		if entry.CreditHold {
			readiness.ValidationFailures = append(readiness.ValidationFailures,
				shipmentbillingports.ShipmentBillingValidation{
					Field:   "billToCustomerId",
					Code:    "credit_hold",
					Message: entry.PayerName + " is on credit hold and cannot be billed",
				})
		}

		readiness.Payers = append(readiness.Payers, entry)
	}

	requirementIssues = hasShipmentRequirementIssues(readiness)
	rateIssues = hasRateIssues(readiness)
	for i := range readiness.Payers {
		payer := payersByID[readiness.Payers[i].PayerID]
		var profile *customer.CustomerBillingProfile
		if payer != nil {
			profile = payer.BillingProfile
		}
		readiness.Payers[i].ShouldAutoApproveBilling = shouldAutoApproveBilling(
			readiness.Policy,
			profile,
			requirementIssues,
			rateIssues,
		)
		autoApprove = autoApprove && readiness.Payers[i].ShouldAutoApproveBilling
	}

	readiness.CanMarkReadyToInvoice = isBillingReadyStatus(entity.Status) &&
		canProceedManually(readiness.Policy, requirementIssues, rateIssues)
	readiness.ShouldAutoMarkReadyToInvoice = readiness.Policy.ReadyToBillAssignmentMode == billingcontrol.ReadyToBillAssignmentModeAutomaticWhenEligible &&
		entity.Status == shipment.StatusCompleted &&
		canAutoProgress(readiness.Policy, requirementIssues, rateIssues)
	readiness.ShouldAutoTransferToBilling = readiness.ShouldAutoMarkReadyToInvoice &&
		readiness.Policy.BillingQueueTransferMode == billingcontrol.BillingQueueTransferModeAutomaticWhenReady
	readiness.ShouldAutoApproveBilling = autoApprove && len(readiness.Payers) > 0
}

func hasValidationCode(
	failures []shipmentbillingports.ShipmentBillingValidation,
	code string,
) bool {
	for _, failure := range failures {
		if failure.Code == code {
			return true
		}
	}

	return false
}

func buildShipmentBillingReadiness(
	entity *shipment.Shipment,
	billingProfile *customer.CustomerBillingProfile,
	billingControl *billingcontrol.BillingControl,
	docs []*document.Document,
) *shipmentbillingports.ShipmentBillingReadiness {
	readiness := &shipmentbillingports.ShipmentBillingReadiness{
		ShipmentID:          entity.ID.String(),
		ShipmentStatus:      entity.Status,
		Requirements:        []shipmentbillingports.ShipmentBillingRequirement{},
		MissingRequirements: []shipmentbillingports.ShipmentBillingRequirement{},
		ValidationFailures:  []shipmentbillingports.ShipmentBillingValidation{},
		Warnings:            []shipmentbillingports.ShipmentBillingWarning{},
		ServiceFailureContext: shipmentbillingports.ShipmentServiceFailureBillingContext{
			ServiceFailureIDs: []string{},
		},
		Policy: shipmentbillingports.ShipmentBillingReadinessPolicy{
			ShipmentBillingRequirementEnforcement: resolveShipmentBillingRequirementEnforcement(
				billingProfile,
				billingControl,
			),
			RateValidationEnforcement: resolveRateValidationEnforcement(
				billingProfile,
				billingControl,
			),
			BillingExceptionDisposition: resolveBillingExceptionDisposition(
				billingControl,
			),
			NotifyOnBillingExceptions: resolveNotifyOnBillingExceptions(billingControl),
			ReadyToBillAssignmentMode: resolveReadyToBillAssignmentMode(
				billingProfile,
				billingControl,
			),
			BillingQueueTransferMode: resolveBillingQueueTransferMode(
				billingProfile,
				billingControl,
			),
		},
	}

	if billingProfile == nil {
		appendRateValidationFailure(entity, billingControl, readiness)
		appendRateDepartureReasonFailure(entity, billingControl, readiness)
		requirementIssues := hasShipmentRequirementIssues(readiness)
		rateIssues := hasRateIssues(readiness)
		readiness.CanMarkReadyToInvoice = entity.Status == shipment.StatusCompleted &&
			canProceedManually(readiness.Policy, requirementIssues, rateIssues)
		readiness.ShouldAutoMarkReadyToInvoice = readiness.Policy.ReadyToBillAssignmentMode == billingcontrol.ReadyToBillAssignmentModeAutomaticWhenEligible &&
			entity.Status == shipment.StatusCompleted &&
			canAutoProgress(readiness.Policy, requirementIssues, rateIssues)
		readiness.ShouldAutoTransferToBilling = readiness.ShouldAutoMarkReadyToInvoice &&
			readiness.Policy.BillingQueueTransferMode == billingcontrol.BillingQueueTransferModeAutomaticWhenReady
		// A customer with no billing profile has not opted into auto-approval.
		return readiness
	}

	requirements := buildDocumentRequirements(billingProfile.DocumentTypes, docs)
	readiness.Requirements = requirements
	readiness.MissingRequirements = slices.DeleteFunc(
		slices.Clone(requirements),
		func(item shipmentbillingports.ShipmentBillingRequirement) bool {
			return item.Satisfied
		},
	)

	if readiness.Policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelBlock {
		if billingProfile.RequireBOLNumber && entity.BOL == "" {
			readiness.ValidationFailures = append(
				readiness.ValidationFailures,
				shipmentbillingports.ShipmentBillingValidation{
					Field:   "bol",
					Code:    "missing_bol",
					Message: "BOL is required before the shipment can be invoiced",
				},
			)
		}
	}

	appendRateValidationFailure(entity, billingControl, readiness)
	appendRateDepartureReasonFailure(entity, billingControl, readiness)

	requirementIssues := hasShipmentRequirementIssues(readiness)
	rateIssues := hasRateIssues(readiness)

	readiness.CanMarkReadyToInvoice = isBillingReadyStatus(entity.Status) &&
		canProceedManually(readiness.Policy, requirementIssues, rateIssues)
	readiness.ShouldAutoMarkReadyToInvoice = readiness.Policy.ReadyToBillAssignmentMode == billingcontrol.ReadyToBillAssignmentModeAutomaticWhenEligible &&
		entity.Status == shipment.StatusCompleted &&
		canAutoProgress(readiness.Policy, requirementIssues, rateIssues)
	readiness.ShouldAutoTransferToBilling = readiness.ShouldAutoMarkReadyToInvoice &&
		readiness.Policy.BillingQueueTransferMode == billingcontrol.BillingQueueTransferModeAutomaticWhenReady
	readiness.ShouldAutoApproveBilling = shouldAutoApproveBilling(
		readiness.Policy,
		billingProfile,
		requirementIssues,
		rateIssues,
	)

	return readiness
}

// shouldAutoApproveBilling reports whether this shipment may clear the billing
// queue without a biller looking at it.
//
// Deliberately stricter than auto-transfer. canAutoProgress lets a shipment move
// while an enforcement level is Ignore, issues and all — which is defensible for
// getting freight into a work queue, and is not defensible for approving it,
// because approval is the step that puts the freight in front of the customer.
// So any requirement or rate issue at all stops auto-approval, whatever the
// organization's enforcement level says.
//
// The organization's queue-transfer mode is the organization-level gate, so a
// shop that has not enabled automatic transfer cannot be auto-approving
// anything; AutoApprove on the profile is the customer's opt-in within it. It
// reads the mode rather than ShouldAutoTransferToBilling because that flag also
// requires the shipment to still be Completed, and by the time anything asks
// this question the shipment is already ReadyToInvoice.
func shouldAutoApproveBilling(
	policy shipmentbillingports.ShipmentBillingReadinessPolicy,
	billingProfile *customer.CustomerBillingProfile,
	requirementIssues bool,
	rateIssues bool,
) bool {
	return policy.BillingQueueTransferMode == billingcontrol.BillingQueueTransferModeAutomaticWhenReady &&
		billingProfile != nil &&
		billingProfile.AutoApprove &&
		!requirementIssues &&
		!rateIssues
}

func isBillingReadyStatus(status shipment.Status) bool {
	return status == shipment.StatusCompleted ||
		status == shipment.StatusReadyToInvoice ||
		status == shipment.StatusInvoiced
}

func resolveShipmentBillingRequirementEnforcement(
	billingProfile *customer.CustomerBillingProfile,
	billingControl *billingcontrol.BillingControl,
) tenant.EnforcementLevel {
	orgLevel := tenant.EnforcementLevelIgnore
	if billingControl != nil {
		orgLevel = billingControl.ShipmentBillingRequirementEnforcement
	}

	customerLevel := tenant.EnforcementLevelIgnore
	if billingProfile != nil && billingProfile.EnforceCustomerBillingReq {
		customerLevel = tenant.EnforcementLevelBlock
	}

	return stricterEnforcementLevel(orgLevel, customerLevel)
}

func resolveReadyToBillAssignmentMode(
	billingProfile *customer.CustomerBillingProfile,
	billingControl *billingcontrol.BillingControl,
) billingcontrol.ReadyToBillAssignmentMode {
	if billingControl == nil {
		if billingProfile != nil && billingProfile.AutoMarkReadyToBill {
			return billingcontrol.ReadyToBillAssignmentModeAutomaticWhenEligible
		}

		return billingcontrol.ReadyToBillAssignmentModeManualOnly
	}

	orgMode := billingControl.ReadyToBillAssignmentMode

	if orgMode != billingcontrol.ReadyToBillAssignmentModeAutomaticWhenEligible {
		return billingcontrol.ReadyToBillAssignmentModeManualOnly
	}

	if billingProfile == nil || billingProfile.AutoMarkReadyToBill {
		return billingcontrol.ReadyToBillAssignmentModeAutomaticWhenEligible
	}

	return billingcontrol.ReadyToBillAssignmentModeManualOnly
}

func resolveRateValidationEnforcement(
	billingProfile *customer.CustomerBillingProfile,
	billingControl *billingcontrol.BillingControl,
) tenant.EnforcementLevel {
	orgLevel := tenant.EnforcementLevelIgnore
	if billingControl != nil {
		orgLevel = billingControl.RateValidationEnforcement
	}

	customerLevel := tenant.EnforcementLevelIgnore
	if billingProfile != nil && billingProfile.ValidateCustomerRates {
		customerLevel = tenant.EnforcementLevelRequireReview
	}

	return stricterEnforcementLevel(orgLevel, customerLevel)
}

func resolveBillingExceptionDisposition(
	billingControl *billingcontrol.BillingControl,
) billingcontrol.BillingExceptionDisposition {
	if billingControl == nil {
		return billingcontrol.BillingExceptionDispositionRouteToBillingReview
	}

	return billingControl.BillingExceptionDisposition
}

func resolveNotifyOnBillingExceptions(
	billingControl *billingcontrol.BillingControl,
) bool {
	if billingControl == nil {
		return false
	}

	return billingControl.NotifyOnBillingExceptions
}

func resolveBillingQueueTransferMode(
	billingProfile *customer.CustomerBillingProfile,
	billingControl *billingcontrol.BillingControl,
) billingcontrol.BillingQueueTransferMode {
	if billingControl == nil {
		if billingProfile != nil && billingProfile.AutoTransfer {
			return billingcontrol.BillingQueueTransferModeAutomaticWhenReady
		}

		return billingcontrol.BillingQueueTransferModeManualOnly
	}

	orgMode := billingControl.BillingQueueTransferMode

	if orgMode != billingcontrol.BillingQueueTransferModeAutomaticWhenReady {
		return billingcontrol.BillingQueueTransferModeManualOnly
	}

	if billingProfile == nil || billingProfile.AutoTransfer {
		return billingcontrol.BillingQueueTransferModeAutomaticWhenReady
	}

	return billingcontrol.BillingQueueTransferModeManualOnly
}

func stricterEnforcementLevel(
	left tenant.EnforcementLevel,
	right tenant.EnforcementLevel,
) tenant.EnforcementLevel {
	if enforcementRank(left) >= enforcementRank(right) {
		return left
	}

	return right
}

func enforcementRank(level tenant.EnforcementLevel) int {
	switch level { //nolint:exhaustive // only actionable enum states require explicit handling here
	case tenant.EnforcementLevelBlock:
		return 3
	case tenant.EnforcementLevelRequireReview:
		return 2
	case tenant.EnforcementLevelWarn:
		return 1
	default:
		return 0
	}
}

func buildDocumentRequirements(
	documentTypes []*documenttype.DocumentType,
	docs []*document.Document,
) []shipmentbillingports.ShipmentBillingRequirement {
	if len(documentTypes) == 0 {
		return []shipmentbillingports.ShipmentBillingRequirement{}
	}

	now := timeutils.NowUnix()
	accepted := make(map[pulid.ID][]string, len(documentTypes))
	ineligible := make(
		map[pulid.ID][]shipmentbillingports.ShipmentBillingIneligibleDocument,
		len(documentTypes),
	)
	for _, doc := range docs {
		if doc == nil || doc.DocumentTypeID == nil {
			continue
		}

		typeID := *doc.DocumentTypeID
		standing := doc.StandingAt(now)
		if standing == document.StandingAccepted {
			accepted[typeID] = append(accepted[typeID], doc.ID.String())
			continue
		}

		ineligible[typeID] = append(
			ineligible[typeID],
			shipmentbillingports.ShipmentBillingIneligibleDocument{
				DocumentID: doc.ID.String(),
				Standing:   standing,
			},
		)
	}

	requirements := make([]shipmentbillingports.ShipmentBillingRequirement, 0, len(documentTypes))
	for _, docType := range documentTypes {
		if docType == nil {
			continue
		}

		documentIDs := accepted[docType.ID]
		if documentIDs == nil {
			documentIDs = []string{}
		}
		rejected := ineligible[docType.ID]
		if rejected == nil {
			rejected = []shipmentbillingports.ShipmentBillingIneligibleDocument{}
		}

		requirements = append(requirements, shipmentbillingports.ShipmentBillingRequirement{
			DocumentTypeID:      docType.ID.String(),
			DocumentTypeCode:    docType.Code,
			DocumentTypeName:    docType.Name,
			Satisfied:           len(documentIDs) > 0,
			DocumentCount:       len(documentIDs),
			DocumentIDs:         documentIDs,
			IneligibleDocuments: rejected,
		})
	}

	sort.SliceStable(requirements, func(i, j int) bool {
		if requirements[i].DocumentTypeCode == requirements[j].DocumentTypeCode {
			return requirements[i].DocumentTypeName < requirements[j].DocumentTypeName
		}

		return requirements[i].DocumentTypeCode < requirements[j].DocumentTypeCode
	})

	return requirements
}

func appendRateValidationFailure(
	entity *shipment.Shipment,
	billingControl *billingcontrol.BillingControl,
	readiness *shipmentbillingports.ShipmentBillingReadiness,
) {
	result := evaluateRateValidation(
		entity,
		billingControl,
		readiness.Policy.RateValidationEnforcement,
	)
	if result == nil {
		return
	}

	readiness.ValidationFailures = append(
		readiness.ValidationFailures,
		shipmentbillingports.ShipmentBillingValidation{
			Field:   result.Field,
			Code:    result.Code,
			Message: result.Message,
		},
	)
}

// appendRateDepartureReasonFailure holds a shipment priced away from its
// contract to the organization's own policy.
//
// A rate that departs from the contract is set by editing the shipment's rating
// fields, which is an ordinary save and cannot be made to demand an explanation
// at the moment it happens. So the demand lands here instead, where it has to
// be satisfied before the shipment can bill — which is the point at which the
// reason is actually needed, since it is what the invoice is explained with.
func appendRateDepartureReasonFailure(
	entity *shipment.Shipment,
	billingControl *billingcontrol.BillingControl,
	readiness *shipmentbillingports.ShipmentBillingReadiness,
) {
	if billingControl == nil || !billingControl.RequireRateOverrideReason {
		return
	}

	if entity == nil || entity.AutoRated || !entity.HasRateOverride() ||
		entity.RateOverrideReason != "" {
		return
	}

	readiness.ValidationFailures = append(
		readiness.ValidationFailures,
		shipmentbillingports.ShipmentBillingValidation{
			Field:   "rateOverrideReason",
			Code:    "rate_departure_reason_missing",
			Message: "Your organization requires a reason when a shipment is not billed at its contract rate",
		},
	)
}

func evaluateRateValidation(
	entity *shipment.Shipment,
	billingControl *billingcontrol.BillingControl,
	enforcement tenant.EnforcementLevel,
) *rateValidationResult {
	if enforcement == tenant.EnforcementLevelIgnore {
		return nil
	}

	if entity == nil || entity.RatingDetail == nil ||
		(entity.RatingDetail.Bill == nil && !entity.RatingDetail.RatedByHand()) {
		return &rateValidationResult{
			Field:   "ratingDetail",
			Code:    "rate_missing_basis",
			Message: "A calculated rate basis is required before billing can proceed",
		}
	}
	if entity.RatingDetail.Bill == nil {
		return nil
	}

	expected := entity.RatingDetail.Bill.FreightTotal
	actual := entity.FreightChargeAmount.Decimal
	variancePercent := calculateVariancePercent(expected, actual)
	if variancePercent.IsZero() {
		return nil
	}
	if billingControl != nil &&
		enforcement == tenant.EnforcementLevelRequireReview &&
		billingControl.RateVarianceAutoResolutionMode == billingcontrol.RateVarianceAutoResolutionModeBypassReviewWithinTolerance &&
		!variancePercent.GreaterThan(billingControl.RateVarianceTolerancePercent) {
		return nil
	}

	return &rateValidationResult{
		Field:   "freightChargeAmount",
		Code:    "rate_variance_requires_action",
		Message: "Rate validation requires billing review before billing can proceed",
	}
}

func calculateVariancePercent(
	expected decimal.Decimal,
	actual decimal.Decimal,
) decimal.Decimal {
	if expected.IsZero() {
		if actual.IsZero() {
			return decimal.Zero
		}

		return decimal.NewFromInt(100)
	}

	return actual.Sub(expected).Abs().Div(expected.Abs()).Mul(decimal.NewFromInt(100))
}

func hasValidationCodePrefix(
	failures []shipmentbillingports.ShipmentBillingValidation,
	prefix string,
) bool {
	for _, failure := range failures {
		if len(failure.Code) >= len(prefix) && failure.Code[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

func hasShipmentRequirementIssues(
	readiness *shipmentbillingports.ShipmentBillingReadiness,
) bool {
	if readiness == nil {
		return false
	}

	if len(readiness.MissingRequirements) > 0 {
		return true
	}

	for _, failure := range readiness.ValidationFailures {
		if len(failure.Code) < len("rate_") || failure.Code[:len("rate_")] != "rate_" {
			return true
		}
	}

	return false
}

func hasRateIssues(
	readiness *shipmentbillingports.ShipmentBillingReadiness,
) bool {
	if readiness == nil {
		return false
	}

	return hasValidationCodePrefix(readiness.ValidationFailures, "rate_")
}

func canProceedManually(
	policy shipmentbillingports.ShipmentBillingReadinessPolicy,
	requirementIssues bool,
	rateIssues bool,
) bool {
	if policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelBlock &&
		requirementIssues {
		return false
	}

	if policy.RateValidationEnforcement == tenant.EnforcementLevelBlock && rateIssues {
		return false
	}

	if policy.BillingExceptionDisposition == billingcontrol.BillingExceptionDispositionReturnToOperations {
		if policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelRequireReview &&
			requirementIssues {
			return false
		}

		if policy.RateValidationEnforcement == tenant.EnforcementLevelRequireReview && rateIssues {
			return false
		}
	}

	return true
}

func canAutoProgress(
	policy shipmentbillingports.ShipmentBillingReadinessPolicy,
	requirementIssues bool,
	rateIssues bool,
) bool {
	if policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelBlock &&
		requirementIssues {
		return false
	}
	if policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelRequireReview &&
		requirementIssues {
		return false
	}
	if policy.RateValidationEnforcement == tenant.EnforcementLevelBlock && rateIssues {
		return false
	}
	if policy.RateValidationEnforcement == tenant.EnforcementLevelRequireReview && rateIssues {
		return false
	}

	return true
}

func (s *service) TransferToBilling(
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
func (s *service) TransferToBillingItems(
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

func (s *service) autoTransferToBillingQueue(
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

func billingTransferPolicyViolation(
	readiness *shipmentbillingports.ShipmentBillingReadiness,
) (shipmentbillingports.BillingTransferFailureCode, error) {
	if readiness.Policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelBlock &&
		hasShipmentRequirementIssues(readiness) {
		return shipmentbillingports.BillingTransferFailureRequirementsUnmet, errortypes.NewValidationError(
			"billingReadiness",
			errortypes.ErrInvalidOperation,
			"Shipment billing requirements must be resolved before transfer to billing",
		)
	}

	if readiness.Policy.RateValidationEnforcement == tenant.EnforcementLevelBlock &&
		hasRateIssues(readiness) {
		return shipmentbillingports.BillingTransferFailureRateValidation, errortypes.NewValidationError(
			"rateValidation",
			errortypes.ErrInvalidOperation,
			"Rate validation must be resolved before transfer to billing",
		)
	}

	if readiness.Policy.BillingExceptionDisposition == billingcontrol.BillingExceptionDispositionReturnToOperations &&
		(readiness.Policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelRequireReview ||
			readiness.Policy.RateValidationEnforcement == tenant.EnforcementLevelRequireReview) &&
		(hasShipmentRequirementIssues(readiness) || hasRateIssues(readiness)) {
		return shipmentbillingports.BillingTransferFailureReturnToOperations, errortypes.NewValidationError(
			"billingExceptionDisposition",
			errortypes.ErrInvalidOperation,
			"Shipment must be corrected in operations before it can be transferred to billing",
		)
	}

	return "", nil
}

func (s *service) notifyBillingExceptions(
	ctx context.Context,
	entity *shipment.Shipment,
	readiness *shipmentbillingports.ShipmentBillingReadiness,
) {
	requirementIssues := hasShipmentRequirementIssues(readiness)
	rateIssues := hasRateIssues(readiness)
	if !requirementIssues && !rateIssues {
		return
	}

	s.createNotification(ctx, &notification.Notification{
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
