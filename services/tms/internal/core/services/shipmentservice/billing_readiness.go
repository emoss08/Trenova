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

	s.l.Info("evaluated shipment billing readiness for auto-mark",
		zap.String("shipmentId", shipmentID.String()),
		zap.String("shipmentStatus", string(readiness.ShipmentStatus)),
		zap.String("shipmentBillingRequirementEnforcement", string(readiness.Policy.ShipmentBillingRequirementEnforcement)),
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
		return nil, nil
	}

	entity, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	if entity.Status == shipment.StatusReadyToInvoice || entity.Status == shipment.StatusInvoiced {
		s.l.Info("shipment already past auto-mark target status",
			zap.String("shipmentId", shipmentID.String()),
			zap.String("shipmentStatus", string(entity.Status)),
		)
		return entity, nil
	}

	previousEntity := *entity
	entity.Status = shipment.StatusReadyToInvoice
	now := timeutils.NowUnix()
	entity.MarkedReadyToBillAt = &now

	s.l.Info("auto-marking shipment ready to invoice",
		zap.String("shipmentId", shipmentID.String()),
		zap.String("currentStatus", string(readiness.ShipmentStatus)),
	)

	auditActor := (&services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
	}).AuditActor()

	updatedEntity, err := s.repo.UpdateDerivedState(ctx, entity)
	if err != nil {
		return nil, err
	}

	if err = s.logShipmentAction(
		updatedEntity,
		auditActor,
		permission.OpUpdate,
		&previousEntity,
		updatedEntity,
		auditservice.WithComment("Shipment auto-marked ready to invoice"),
		auditservice.WithDiff(&previousEntity, updatedEntity),
	); err != nil {
		s.l.Error("failed to log auto-mark shipment action", zap.Error(err))
	}

	if err = s.publishShipmentInvalidation(
		ctx,
		updatedEntity,
		auditActor,
		"updated",
		updatedEntity,
	); err != nil {
		s.l.Warn("failed to publish shipment invalidation after auto-mark", zap.Error(err))
	}

	if readiness.ShouldAutoTransferToBilling {
		s.autoTransferToBillingQueue(ctx, updatedEntity, &services.RequestActor{
			PrincipalType:  services.PrincipalTypeUser,
			PrincipalID:    userID,
			UserID:         userID,
			BusinessUnitID: tenantInfo.BuID,
			OrganizationID: tenantInfo.OrgID,
		})
	}

	return updatedEntity, nil
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
			"Missing required document: "+requirement.DocumentTypeName,
		)
	}

	return multiErr
}

func (s *service) evaluateBillingReadiness(
	ctx context.Context,
	entity *shipment.Shipment,
) (*services.ShipmentBillingReadiness, error) {
	if s.customerRepo == nil || s.documentRepo == nil || s.billingRepo == nil {
		return nil, errortypes.NewConflictError("Shipment billing readiness service is unavailable")
	}

	customerEntity, err := s.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
		ID: entity.CustomerID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		CustomerFilterOptions: repositories.CustomerFilterOptions{
			IncludeBillingProfile: true,
		},
	})
	if err != nil {
		return nil, err
	}

	billingControl, err := s.billingRepo.GetByOrgID(ctx, entity.OrganizationID)
	switch {
	case err == nil:
	case errortypes.IsNotFoundError(err):
		billingControl = nil
	default:
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
	s.l.Debug("evaluated shipment billing readiness",
		zap.String("shipmentId", entity.ID.String()),
		zap.String("shipmentStatus", string(entity.Status)),
		zap.String("shipmentBillingRequirementEnforcement", string(readiness.Policy.ShipmentBillingRequirementEnforcement)),
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
		Policy: services.ShipmentBillingReadinessPolicy{
			ShipmentBillingRequirementEnforcement: resolveShipmentBillingRequirementEnforcement(billingProfile, billingControl),
			RateValidationEnforcement:             resolveRateValidationEnforcement(billingProfile, billingControl),
			BillingExceptionDisposition:           resolveBillingExceptionDisposition(billingControl),
			NotifyOnBillingExceptions:             resolveNotifyOnBillingExceptions(billingControl),
			ReadyToBillAssignmentMode:             resolveReadyToBillAssignmentMode(billingProfile, billingControl),
			BillingQueueTransferMode:              resolveBillingQueueTransferMode(billingProfile, billingControl),
		},
	}

	if billingProfile == nil {
		appendRateValidationFailure(entity, billingProfile, billingControl, readiness)
		requirementIssues := hasShipmentRequirementIssues(readiness)
		rateIssues := hasRateIssues(readiness)
		readiness.CanMarkReadyToInvoice =
			entity.Status == shipment.StatusCompleted &&
				canProceedManually(readiness.Policy, requirementIssues, rateIssues)
		readiness.ShouldAutoMarkReadyToInvoice =
			readiness.Policy.ReadyToBillAssignmentMode == tenant.ReadyToBillAssignmentModeAutomaticWhenEligible &&
				entity.Status == shipment.StatusCompleted &&
				canAutoProgress(readiness.Policy, requirementIssues, rateIssues)
		readiness.ShouldAutoTransferToBilling =
			readiness.ShouldAutoMarkReadyToInvoice &&
				readiness.Policy.BillingQueueTransferMode == tenant.BillingQueueTransferModeAutomaticWhenReady
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

	appendRateValidationFailure(entity, billingProfile, billingControl, readiness)

	requirementIssues := hasShipmentRequirementIssues(readiness)
	rateIssues := hasRateIssues(readiness)

	readiness.CanMarkReadyToInvoice =
		isBillingReadyStatus(entity.Status) &&
			canProceedManually(readiness.Policy, requirementIssues, rateIssues)
	readiness.ShouldAutoMarkReadyToInvoice =
		readiness.Policy.ReadyToBillAssignmentMode == tenant.ReadyToBillAssignmentModeAutomaticWhenEligible &&
			entity.Status == shipment.StatusCompleted &&
			canAutoProgress(readiness.Policy, requirementIssues, rateIssues)
	readiness.ShouldAutoTransferToBilling =
		readiness.ShouldAutoMarkReadyToInvoice &&
			readiness.Policy.BillingQueueTransferMode == tenant.BillingQueueTransferModeAutomaticWhenReady

	return readiness
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

	orgMode := tenant.ReadyToBillAssignmentModeManualOnly
	orgMode = billingControl.ReadyToBillAssignmentMode

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

	orgMode := tenant.BillingQueueTransferModeManualOnly
	orgMode = billingControl.BillingQueueTransferMode

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
	switch level {
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
	billingProfile *customer.CustomerBillingProfile,
	billingControl *tenant.BillingControl,
	readiness *services.ShipmentBillingReadiness,
) {
	result := evaluateRateValidation(entity, billingProfile, billingControl, readiness.Policy.RateValidationEnforcement)
	if result == nil {
		return
	}

	readiness.ValidationFailures = append(readiness.ValidationFailures, services.ShipmentBillingValidation{
		Field:   result.Field,
		Code:    result.Code,
		Message: result.Message,
	})
}

func evaluateRateValidation(
	entity *shipment.Shipment,
	billingProfile *customer.CustomerBillingProfile,
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
	if policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelBlock && requirementIssues {
		return false
	}
	if policy.RateValidationEnforcement == tenant.EnforcementLevelBlock && rateIssues {
		return false
	}
	if policy.BillingExceptionDisposition == tenant.BillingExceptionDispositionReturnToOperations {
		if policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelRequireReview && requirementIssues {
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
	if policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelBlock && requirementIssues {
		return false
	}
	if policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelRequireReview && requirementIssues {
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

	log := s.l.With(
		zap.String("operation", "TransferToBilling"),
		zap.String("shipmentID", req.ShipmentID.String()),
	)

	entity, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID: req.ShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: actor.OrganizationID,
			BuID:  actor.BusinessUnitID,
		},
	})
	if err != nil {
		log.Error("failed to get shipment for billing transfer", zap.Error(err))
		return nil, err
	}

	if entity.BillingTransferStatus != shipment.BillingTransferNone &&
		entity.BillingTransferStatus != shipment.BillingTransferSentBackToOps {
		return nil, errortypes.NewValidationError(
			"billingTransferStatus",
			errortypes.ErrInvalidOperation,
			"Shipment has already been transferred to billing",
		)
	}

	readiness, err := s.evaluateBillingReadiness(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.notifyBillingExceptions(ctx, entity, readiness)

	if transferErr := validateBillingTransferPolicy(readiness); transferErr != nil {
		return nil, transferErr
	}

	billType := req.BillType
	if billType == "" {
		billType = billingqueue.BillTypeInvoice
	}

	item, err := s.billingQueueService.TransferToBilling(ctx, &services.TransferToBillingRequest{
		ShipmentID: req.ShipmentID,
		BillType:   billType,
		TenantInfo: pagination.TenantInfo{
			OrgID: actor.OrganizationID,
			BuID:  actor.BusinessUnitID,
		},
	}, actor)
	if err != nil {
		log.Error("failed to transfer shipment to billing", zap.Error(err))
		return nil, err
	}

	return item, nil
}

func (s *service) BulkTransferToBilling(
	ctx context.Context,
	req *services.BulkTransferShipmentToBillingRequest,
	actor *services.RequestActor,
) (*services.BulkTransferToBillingResponse, error) {
	if len(req.ShipmentIDs) == 0 {
		return &services.BulkTransferToBillingResponse{}, nil
	}

	results := make([]services.BulkTransferToBillingResult, 0, len(req.ShipmentIDs))
	successCount := 0

	for _, shipmentID := range req.ShipmentIDs {
		item, err := s.TransferToBilling(ctx, &services.TransferShipmentToBillingRequest{
			ShipmentID: shipmentID,
			BillType:   req.BillType,
		}, actor)

		result := services.BulkTransferToBillingResult{
			ShipmentID: shipmentID,
		}
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Success = true
			result.Item = item
			successCount++
		}

		results = append(results, result)
	}

	return &services.BulkTransferToBillingResponse{
		Results:      results,
		TotalCount:   len(req.ShipmentIDs),
		SuccessCount: successCount,
		ErrorCount:   len(req.ShipmentIDs) - successCount,
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

func validateBillingTransferPolicy(
	readiness *services.ShipmentBillingReadiness,
) error {
	if readiness == nil {
		return nil
	}

	if readiness.Policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelBlock &&
		hasShipmentRequirementIssues(readiness) {
		return errortypes.NewValidationError(
			"billingReadiness",
			errortypes.ErrInvalidOperation,
			"Shipment billing requirements must be resolved before transfer to billing",
		)
	}

	if readiness.Policy.RateValidationEnforcement == tenant.EnforcementLevelBlock &&
		hasValidationCodePrefix(readiness.ValidationFailures, "rate_") {
		return errortypes.NewValidationError(
			"rateValidation",
			errortypes.ErrInvalidOperation,
			"Rate validation must be resolved before transfer to billing",
		)
	}

	if readiness.Policy.BillingExceptionDisposition == tenant.BillingExceptionDispositionReturnToOperations &&
		(readiness.Policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelRequireReview ||
			readiness.Policy.RateValidationEnforcement == tenant.EnforcementLevelRequireReview) &&
		(hasShipmentRequirementIssues(readiness) || hasRateIssues(readiness)) {
		return errortypes.NewValidationError(
			"billingExceptionDisposition",
			errortypes.ErrInvalidOperation,
			"Shipment must be corrected in operations before it can be transferred to billing",
		)
	}

	return nil
}

func (s *service) notifyBillingExceptions(
	ctx context.Context,
	entity *shipment.Shipment,
	readiness *services.ShipmentBillingReadiness,
) {
	if entity == nil || readiness == nil || !readiness.Policy.NotifyOnBillingExceptions {
		return
	}

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
