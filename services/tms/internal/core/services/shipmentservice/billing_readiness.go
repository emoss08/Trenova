//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package shipmentservice

import (
	"context"
	"slices"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
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
) (*services.ShipmentBillingReadiness, error) {
	entity, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
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

	actor := &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
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
	ShipmentID pulid.ID
	TenantInfo pagination.TenantInfo
	Actor      services.AuditActor
	Comment    string
}

func (s *service) markReadyToInvoice(
	ctx context.Context,
	p *markReadyToInvoiceParams,
) (*shipment.Shipment, bool, error) {
	entity, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         p.ShipmentID,
		TenantInfo: p.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
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

	updatedEntity, err := s.repo.UpdateDerivedState(ctx, entity)
	if err != nil {
		return nil, false, err
	}

	if err = s.recomputeOrdersForShipments(
		ctx,
		p.TenantInfo,
		[]*shipment.Shipment{updatedEntity},
	); err != nil {
		s.l.Warn("failed to recompute order after marking ready to invoice", zap.Error(err))
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

	if err = s.publishShipmentInvalidation(
		ctx,
		updatedEntity,
		p.Actor,
		"updated",
		updatedEntity,
	); err != nil {
		s.l.Warn(
			"failed to publish shipment invalidation after marking ready to invoice",
			zap.Error(err),
		)
	}

	return updatedEntity, true, nil
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
	billingControl      *tenant.BillingControl
	billingControlKnown bool
}

// billingControlFor reads the organization's billing policy, at most once per
// bulk transfer. The policy is one row per organization and every shipment in a
// run asks the same question of it.
func (s *service) billingControlFor(
	ctx context.Context,
	orgID pulid.ID,
	cache *billingReadinessCache,
) (*tenant.BillingControl, error) {
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
) (*services.ShipmentBillingReadiness, error) {
	return s.evaluateBillingReadinessCached(ctx, entity, nil)
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
) (*services.ShipmentBillingReadiness, error) {
	if s.customerRepo == nil || s.documentRepo == nil || s.billingRepo == nil {
		return nil, errortypes.NewConflictError("Shipment billing readiness service is unavailable")
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	resolution, err := s.resolvePayerShares(ctx, entity, tenantInfo)
	if err != nil {
		return nil, err
	}

	payers, err := s.customerRepo.GetByIDs(ctx, repositories.GetCustomersByIDsRequest{
		TenantInfo:  tenantInfo,
		CustomerIDs: resolution.PayerIDs(),
		CustomerFilterOptions: repositories.CustomerFilterOptions{
			IncludeBillingProfile: true,
		},
	})
	if err != nil {
		return nil, err
	}
	payersByID := make(map[pulid.ID]*customer.Customer, len(payers))
	for _, payer := range payers {
		if payer != nil {
			payersByID[payer.ID] = payer
		}
	}
	customerEntity, ok := payersByID[resolution.DefaultPayerID]
	if !ok {
		customerEntity, err = s.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
			ID:         resolution.DefaultPayerID,
			TenantInfo: tenantInfo,
			CustomerFilterOptions: repositories.CustomerFilterOptions{
				IncludeBillingProfile: true,
			},
		})
		if err != nil {
			return nil, err
		}
		payersByID[customerEntity.ID] = customerEntity
	}

	billingControl, err := s.billingControlFor(ctx, entity.OrganizationID, cache)
	if err != nil {
		return nil, err
	}

	docs, err := s.documentRepo.GetByResourceID(ctx, &repositories.GetDocumentsByResourceRequest{
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

	readiness := buildShipmentBillingReadiness(
		entity,
		customerEntity.BillingProfile,
		billingControl,
		docs,
	)
	applyPayerReadiness(readiness, entity, resolution, payersByID, docs)
	if err = s.applyServiceFailureBillingContext(ctx, entity, readiness); err != nil {
		s.l.Warn("failed to apply service failure billing context",
			zap.String("shipmentId", entity.ID.String()),
			zap.Error(err),
		)
	}
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
	return readiness, nil
}

func (s *service) applyServiceFailureBillingContext(
	ctx context.Context,
	entity *shipment.Shipment,
	readiness *services.ShipmentBillingReadiness,
) error {
	if s.serviceFailureRepo == nil || entity == nil || readiness == nil {
		return nil
	}

	unresolved, err := s.serviceFailureRepo.ListUnresolvedByShipment(
		ctx,
		&repositories.ServiceFailuresByShipmentRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			ShipmentID: entity.ID,
		},
	)
	if err != nil {
		return err
	}
	if len(unresolved) == 0 {
		return nil
	}

	ids := make([]string, 0, len(unresolved))
	for _, failure := range unresolved {
		if failure == nil {
			continue
		}
		ids = append(ids, failure.ID.String())
	}
	readiness.ServiceFailureContext.HasUnresolved = len(ids) > 0
	readiness.ServiceFailureContext.UnresolvedCount = len(ids)
	readiness.ServiceFailureContext.ServiceFailureIDs = ids
	readiness.Warnings = append(readiness.Warnings, services.ShipmentBillingWarning{
		Code:    "unresolved_service_failures",
		Message: "Shipment has unresolved service failures",
		Context: map[string]any{
			"serviceFailureIds": ids,
			"unresolvedCount":   len(ids),
		},
	})
	return nil
}

// resolvePayerShares divides the shipment among its payers, reloading it with
// charges and allocations when the caller handed over a bare header.
func (s *service) resolvePayerShares(
	ctx context.Context,
	entity *shipment.Shipment,
	tenantInfo pagination.TenantInfo,
) (*shipment.ShareResolution, error) {
	source := entity
	if (entity.AdditionalCharges == nil || entity.ChargeAllocations == nil) &&
		s.repo != nil && entity.ID.IsNotNil() {
		full, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
			ID:         entity.ID,
			TenantInfo: tenantInfo,
			ShipmentOptions: repositories.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
		})
		if err != nil {
			return nil, err
		}
		source = full
	}

	resolution, err := shipment.ResolveShares(source, source.ChargeAllocations)
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
	readiness *services.ShipmentBillingReadiness,
	entity *shipment.Shipment,
	resolution *shipment.ShareResolution,
	payersByID map[pulid.ID]*customer.Customer,
	docs []*document.Document,
) {
	if readiness == nil || resolution == nil {
		return
	}
	readiness.Payers = make([]services.ShipmentBillingPayerReadiness, 0, len(resolution.Shares))

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
		entry := services.ShipmentBillingPayerReadiness{
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
			entry.CreditHold = profile.EnforceCreditLimit &&
				(profile.CreditStatus == customer.CreditStatusHold ||
					profile.CreditStatus == customer.CreditStatusSuspended)
		}

		if !entry.IsPrimary && profile != nil {
			for _, requirement := range buildDocumentRequirements(profile.DocumentTypes, docs) {
				if _, ok := known[requirement.DocumentTypeID]; ok {
					continue
				}
				known[requirement.DocumentTypeID] = struct{}{}
				readiness.Requirements = append(readiness.Requirements, requirement)
				if !requirement.Satisfied {
					readiness.MissingRequirements = append(readiness.MissingRequirements, requirement)
				}
			}
			if readiness.Policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelBlock &&
				profile.RequireBOLNumber && entity.BOL == "" &&
				!hasValidationCode(readiness.ValidationFailures, "missing_bol") {
				readiness.ValidationFailures = append(readiness.ValidationFailures,
					services.ShipmentBillingValidation{
						Field:   "bol",
						Code:    "missing_bol",
						Message: "BOL is required before the shipment can be invoiced",
					})
			}
		}

		if entry.CreditHold {
			readiness.ValidationFailures = append(readiness.ValidationFailures,
				services.ShipmentBillingValidation{
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
	readiness.ShouldAutoMarkReadyToInvoice = readiness.Policy.ReadyToBillAssignmentMode == tenant.ReadyToBillAssignmentModeAutomaticWhenEligible &&
		entity.Status == shipment.StatusCompleted &&
		canAutoProgress(readiness.Policy, requirementIssues, rateIssues)
	readiness.ShouldAutoTransferToBilling = readiness.ShouldAutoMarkReadyToInvoice &&
		readiness.Policy.BillingQueueTransferMode == tenant.BillingQueueTransferModeAutomaticWhenReady
	readiness.ShouldAutoApproveBilling = autoApprove && len(readiness.Payers) > 0
}

func hasValidationCode(failures []services.ShipmentBillingValidation, code string) bool {
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
	billingControl *tenant.BillingControl,
	docs []*document.Document,
) *services.ShipmentBillingReadiness {
	readiness := &services.ShipmentBillingReadiness{
		ShipmentID:          entity.ID.String(),
		ShipmentStatus:      entity.Status,
		Requirements:        []services.ShipmentBillingRequirement{},
		MissingRequirements: []services.ShipmentBillingRequirement{},
		ValidationFailures:  []services.ShipmentBillingValidation{},
		Warnings:            []services.ShipmentBillingWarning{},
		ServiceFailureContext: services.ShipmentServiceFailureBillingContext{
			ServiceFailureIDs: []string{},
		},
		Policy: services.ShipmentBillingReadinessPolicy{
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
		readiness.ShouldAutoMarkReadyToInvoice = readiness.Policy.ReadyToBillAssignmentMode == tenant.ReadyToBillAssignmentModeAutomaticWhenEligible &&
			entity.Status == shipment.StatusCompleted &&
			canAutoProgress(readiness.Policy, requirementIssues, rateIssues)
		readiness.ShouldAutoTransferToBilling = readiness.ShouldAutoMarkReadyToInvoice &&
			readiness.Policy.BillingQueueTransferMode == tenant.BillingQueueTransferModeAutomaticWhenReady
		// A customer with no billing profile has not opted into auto-approval.
		return readiness
	}

	requirements := buildDocumentRequirements(billingProfile.DocumentTypes, docs)
	readiness.Requirements = requirements
	readiness.MissingRequirements = slices.DeleteFunc(
		slices.Clone(requirements),
		func(item services.ShipmentBillingRequirement) bool {
			return item.Satisfied
		},
	)

	if readiness.Policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelBlock {
		if billingProfile.RequireBOLNumber && entity.BOL == "" {
			readiness.ValidationFailures = append(
				readiness.ValidationFailures,
				services.ShipmentBillingValidation{
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
	readiness.ShouldAutoMarkReadyToInvoice = readiness.Policy.ReadyToBillAssignmentMode == tenant.ReadyToBillAssignmentModeAutomaticWhenEligible &&
		entity.Status == shipment.StatusCompleted &&
		canAutoProgress(readiness.Policy, requirementIssues, rateIssues)
	readiness.ShouldAutoTransferToBilling = readiness.ShouldAutoMarkReadyToInvoice &&
		readiness.Policy.BillingQueueTransferMode == tenant.BillingQueueTransferModeAutomaticWhenReady
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
	policy services.ShipmentBillingReadinessPolicy,
	billingProfile *customer.CustomerBillingProfile,
	requirementIssues bool,
	rateIssues bool,
) bool {
	return policy.BillingQueueTransferMode == tenant.BillingQueueTransferModeAutomaticWhenReady &&
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
	billingControl *tenant.BillingControl,
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
	billingControl *tenant.BillingControl,
) tenant.ReadyToBillAssignmentMode {
	if billingControl == nil {
		if billingProfile != nil && billingProfile.AutoMarkReadyToBill {
			return tenant.ReadyToBillAssignmentModeAutomaticWhenEligible
		}

		return tenant.ReadyToBillAssignmentModeManualOnly
	}

	orgMode := billingControl.ReadyToBillAssignmentMode

	if orgMode != tenant.ReadyToBillAssignmentModeAutomaticWhenEligible {
		return tenant.ReadyToBillAssignmentModeManualOnly
	}

	if billingProfile == nil || billingProfile.AutoMarkReadyToBill {
		return tenant.ReadyToBillAssignmentModeAutomaticWhenEligible
	}

	return tenant.ReadyToBillAssignmentModeManualOnly
}

func resolveRateValidationEnforcement(
	billingProfile *customer.CustomerBillingProfile,
	billingControl *tenant.BillingControl,
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
	billingControl *tenant.BillingControl,
) tenant.BillingExceptionDisposition {
	if billingControl == nil {
		return tenant.BillingExceptionDispositionRouteToBillingReview
	}

	return billingControl.BillingExceptionDisposition
}

func resolveNotifyOnBillingExceptions(
	billingControl *tenant.BillingControl,
) bool {
	if billingControl == nil {
		return false
	}

	return billingControl.NotifyOnBillingExceptions
}

func resolveBillingQueueTransferMode(
	billingProfile *customer.CustomerBillingProfile,
	billingControl *tenant.BillingControl,
) tenant.BillingQueueTransferMode {
	if billingControl == nil {
		if billingProfile != nil && billingProfile.AutoTransfer {
			return tenant.BillingQueueTransferModeAutomaticWhenReady
		}

		return tenant.BillingQueueTransferModeManualOnly
	}

	orgMode := billingControl.BillingQueueTransferMode

	if orgMode != tenant.BillingQueueTransferModeAutomaticWhenReady {
		return tenant.BillingQueueTransferModeManualOnly
	}

	if billingProfile == nil || billingProfile.AutoTransfer {
		return tenant.BillingQueueTransferModeAutomaticWhenReady
	}

	return tenant.BillingQueueTransferModeManualOnly
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
) []services.ShipmentBillingRequirement {
	if len(documentTypes) == 0 {
		return []services.ShipmentBillingRequirement{}
	}

	documentsByType := make(map[pulid.ID][]*document.Document, len(documentTypes))
	for _, doc := range docs {
		if doc == nil || doc.DocumentTypeID == nil {
			continue
		}

		documentsByType[*doc.DocumentTypeID] = append(documentsByType[*doc.DocumentTypeID], doc)
	}

	requirements := make([]services.ShipmentBillingRequirement, 0, len(documentTypes))
	for _, docType := range documentTypes {
		if docType == nil {
			continue
		}

		matches := documentsByType[docType.ID]
		documentIDs := make([]string, 0, len(matches))
		for _, doc := range matches {
			documentIDs = append(documentIDs, doc.ID.String())
		}

		requirements = append(requirements, services.ShipmentBillingRequirement{
			DocumentTypeID:   docType.ID.String(),
			DocumentTypeCode: docType.Code,
			DocumentTypeName: docType.Name,
			Satisfied:        len(matches) > 0,
			DocumentCount:    len(matches),
			DocumentIDs:      documentIDs,
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
	billingControl *tenant.BillingControl,
	readiness *services.ShipmentBillingReadiness,
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
		services.ShipmentBillingValidation{
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
	billingControl *tenant.BillingControl,
	readiness *services.ShipmentBillingReadiness,
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
		services.ShipmentBillingValidation{
			Field:   "rateOverrideReason",
			Code:    "rate_departure_reason_missing",
			Message: "Your organization requires a reason when a shipment is not billed at its contract rate",
		},
	)
}

func evaluateRateValidation(
	entity *shipment.Shipment,
	billingControl *tenant.BillingControl,
	enforcement tenant.EnforcementLevel,
) *rateValidationResult {
	if enforcement == tenant.EnforcementLevelIgnore {
		return nil
	}

	if entity == nil || entity.RatingDetail == nil {
		return &rateValidationResult{
			Field:   "ratingDetail",
			Code:    "rate_missing_basis",
			Message: "A calculated rate basis is required before billing can proceed",
		}
	}

	expected := decimal.NewFromFloat(entity.RatingDetail.Result)
	actual := entity.FreightChargeAmount.Decimal
	variancePercent := calculateVariancePercent(expected, actual)
	if variancePercent.IsZero() {
		return nil
	}
	if billingControl != nil &&
		enforcement == tenant.EnforcementLevelRequireReview &&
		billingControl.RateVarianceAutoResolutionMode == tenant.RateVarianceAutoResolutionModeBypassReviewWithinTolerance &&
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
	failures []services.ShipmentBillingValidation,
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
	readiness *services.ShipmentBillingReadiness,
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
	readiness *services.ShipmentBillingReadiness,
) bool {
	if readiness == nil {
		return false
	}

	return hasValidationCodePrefix(readiness.ValidationFailures, "rate_")
}

func canProceedManually(
	policy services.ShipmentBillingReadinessPolicy,
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

	if policy.BillingExceptionDisposition == tenant.BillingExceptionDispositionReturnToOperations {
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
	policy services.ShipmentBillingReadinessPolicy,
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
	req *services.TransferShipmentToBillingRequest,
	actor *services.RequestActor,
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
	req *services.TransferShipmentToBillingRequest,
	actor *services.RequestActor,
) (*services.TransferToBillingResult, error) {
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

	return &services.TransferToBillingResult{
		Items:   attempt.items,
		Primary: attempt.item,
	}, nil
}

func (s *service) autoTransferToBillingQueue(
	ctx context.Context,
	entity *shipment.Shipment,
	actor *services.RequestActor,
) {
	if _, err := s.TransferToBilling(ctx, &services.TransferShipmentToBillingRequest{
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
	readiness *services.ShipmentBillingReadiness,
) (services.BillingTransferFailureCode, error) {
	if readiness.Policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelBlock &&
		hasShipmentRequirementIssues(readiness) {
		return services.BillingTransferFailureRequirementsUnmet, errortypes.NewValidationError(
			"billingReadiness",
			errortypes.ErrInvalidOperation,
			"Shipment billing requirements must be resolved before transfer to billing",
		)
	}

	if readiness.Policy.RateValidationEnforcement == tenant.EnforcementLevelBlock &&
		hasRateIssues(readiness) {
		return services.BillingTransferFailureRateValidation, errortypes.NewValidationError(
			"rateValidation",
			errortypes.ErrInvalidOperation,
			"Rate validation must be resolved before transfer to billing",
		)
	}

	if readiness.Policy.BillingExceptionDisposition == tenant.BillingExceptionDispositionReturnToOperations &&
		(readiness.Policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelRequireReview ||
			readiness.Policy.RateValidationEnforcement == tenant.EnforcementLevelRequireReview) &&
		(hasShipmentRequirementIssues(readiness) || hasRateIssues(readiness)) {
		return services.BillingTransferFailureReturnToOperations, errortypes.NewValidationError(
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
	readiness *services.ShipmentBillingReadiness,
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
