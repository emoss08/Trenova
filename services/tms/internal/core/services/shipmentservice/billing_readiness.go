package shipmentservice

import (
	"context"
	"slices"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

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
		zap.Bool("enforceCustomerBillingReq", readiness.Policy.EnforceCustomerBillingReq),
		zap.Bool("autoMarkReadyToBill", readiness.Policy.AutoMarkReadyToBill),
		zap.Bool("canMarkReadyToInvoice", readiness.CanMarkReadyToInvoice),
		zap.Bool("shouldAutoMarkReadyToInvoice", readiness.ShouldAutoMarkReadyToInvoice),
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
		zap.Bool("enforceCustomerBillingReq", readiness.Policy.EnforceCustomerBillingReq),
		zap.Bool("autoMarkReadyToBill", readiness.Policy.AutoMarkReadyToBill),
		zap.Bool("canMarkReadyToInvoice", readiness.CanMarkReadyToInvoice),
		zap.Bool("shouldAutoMarkReadyToInvoice", readiness.ShouldAutoMarkReadyToInvoice),
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
	}

	if billingProfile == nil {
		readiness.CanMarkReadyToInvoice = entity.Status == shipment.StatusCompleted
		return readiness
	}

	readiness.Policy = services.ShipmentBillingReadinessPolicy{
		EnforceCustomerBillingReq: resolveEnforceCustomerBillingReq(billingProfile, billingControl),
		AutoMarkReadyToBill:       resolveAutoMarkReadyToBill(billingProfile, billingControl),
	}

	requirements := buildDocumentRequirements(billingProfile.DocumentTypes, docs)
	readiness.Requirements = requirements
	readiness.MissingRequirements = slices.DeleteFunc(
		slices.Clone(requirements),
		func(item services.ShipmentBillingRequirement) bool {
			return item.Satisfied
		},
	)

	if readiness.Policy.EnforceCustomerBillingReq {
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

	requirementsSatisfied := !readiness.Policy.EnforceCustomerBillingReq ||
		(len(readiness.MissingRequirements) == 0 && len(readiness.ValidationFailures) == 0)

	readiness.CanMarkReadyToInvoice = requirementsSatisfied && isBillingReadyStatus(entity.Status)
	readiness.ShouldAutoMarkReadyToInvoice = readiness.Policy.AutoMarkReadyToBill &&
		requirementsSatisfied &&
		entity.Status == shipment.StatusCompleted

	return readiness
}

func isBillingReadyStatus(status shipment.Status) bool {
	return status == shipment.StatusCompleted ||
		status == shipment.StatusReadyToInvoice ||
		status == shipment.StatusInvoiced
}

func resolveEnforceCustomerBillingReq(
	billingProfile *customer.CustomerBillingProfile,
	billingControl *tenant.BillingControl,
) bool {
	if billingControl != nil && billingControl.EnforceCustomerBillingReq {
		return true
	}

	return billingProfile != nil && billingProfile.EnforceCustomerBillingReq
}

func resolveAutoMarkReadyToBill(
	billingProfile *customer.CustomerBillingProfile,
	billingControl *tenant.BillingControl,
) bool {
	if billingControl != nil && billingControl.AutoMarkReadyToBill {
		return true
	}

	return billingProfile != nil && billingProfile.AutoMarkReadyToBill
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
