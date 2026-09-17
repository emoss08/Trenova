package carrierintelservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type VerifyEquipmentRequest struct {
	TenantInfo          pagination.TenantInfo
	CarrierAssignmentID pulid.ID
	UnitType            carrierintel.UnitType
	VIN                 string
	PlateNumber         string
	PlateState          string
	UnitNumber          string
}

func (s *Service) VerifyEquipment(
	ctx context.Context,
	req *VerifyEquipmentRequest,
) (*carrierintel.CarrierEquipmentVerification, error) {
	ctx = carrierintel.WithPurpose(ctx, carrierintel.PurposeVerify)

	assignment, err := s.assignmentRepo.GetByID(ctx, &repositories.GetCarrierAssignmentByIDRequest{
		TenantInfo:          req.TenantInfo,
		CarrierAssignmentID: req.CarrierAssignmentID,
	})
	if err != nil {
		return nil, err
	}

	carrierEntity, err := s.carrierRepo.GetByID(ctx, repositories.GetCarrierByIDRequest{
		ID:         assignment.CarrierID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	entity := &carrierintel.CarrierEquipmentVerification{
		OrganizationID:      req.TenantInfo.OrgID,
		BusinessUnitID:      req.TenantInfo.BuID,
		CarrierAssignmentID: assignment.ID,
		ShipmentMoveID:      assignment.ShipmentMoveID,
		CarrierID:           carrierEntity.ID,
		ExpectedDOTNumber:   carrierEntity.DOTNumber,
		UnitType:            req.UnitType,
		VIN:                 req.VIN,
		PlateNumber:         req.PlateNumber,
		PlateState:          req.PlateState,
		UnitNumber:          req.UnitNumber,
		VerifiedByID:        req.TenantInfo.UserID,
		VerifiedAt:          s.now(),
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	s.runEquipmentLookup(ctx, req.TenantInfo, entity, carrierEntity.Name)

	created, err := s.verifyRepo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceEquipmentVerification,
		ResourceID:     created.ID.String(),
		Operation:      permission.OpCreate,
		UserID:         req.TenantInfo.UserID,
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    req.TenantInfo.UserID,
		CurrentState:   jsonutils.MustToJSON(created),
		OrganizationID: created.OrganizationID,
		BusinessUnitID: created.BusinessUnitID,
	}, auditservice.WithComment("Equipment verified: "+created.Result.String())); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	if created.Result == carrierintel.VerificationResultMismatch {
		s.flagEquipmentMismatch(ctx, req.TenantInfo, created, carrierEntity.Name)
	}

	s.publish(ctx, req.TenantInfo, "carrier_equipment_verifications", "created", created.ID)
	return created, nil
}

func (s *Service) runEquipmentLookup(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *carrierintel.CarrierEquipmentVerification,
	carrierName string,
) {
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		entity.Result = carrierintel.VerificationResultProviderError
		entity.MismatchReason = "Carrier intelligence settings could not be loaded"
		return
	}
	bound, err := s.resolvePrimary(ctx, tenantInfo, control)
	if err != nil {
		entity.Result = carrierintel.VerificationResultUnverifiable
		entity.MismatchReason = "No carrier intelligence provider is enabled"
		return
	}
	entity.Provider = bound.provider

	lookup, ok := bound.client.(services.CarrierIntelEquipmentLookup)
	if !ok || !bound.capabilities().Has(carrierintel.CapabilityEquipmentLookup) {
		entity.Result = carrierintel.VerificationResultUnverifiable
		entity.MismatchReason = bound.provider.String() + " cannot look up equipment by VIN or plate"
		return
	}
	if entity.ExpectedDOTNumber == "" {
		entity.Result = carrierintel.VerificationResultUnverifiable
		entity.MismatchReason = carrierName + " has no DOT number to compare against"
		return
	}

	if err = s.guardBudget(ctx, &budgetCheck{
		tenant:   tenantInfo,
		control:  control,
		bound:    bound,
		endpoint: carrierintel.EndpointEquipment,
	}); err != nil {
		entity.Result = carrierintel.VerificationResultProviderError
		entity.MismatchReason = errorSummary(err)
		return
	}

	var result *services.CarrierIntelEquipmentResult
	err = s.withRateLimitWait(ctx, func() error {
		var callErr error
		result, callErr = lookup.FindByEquipment(ctx, &services.CarrierIntelEquipmentRequest{
			VIN:         entity.VIN,
			PlateNumber: entity.PlateNumber,
			PlateState:  entity.PlateState,
			UnitNumber:  entity.UnitNumber,
			UnitType:    entity.UnitType,
		})
		return callErr
	})
	if err != nil {
		if services.IsCarrierIntelErrorKind(err, services.CarrierIntelErrorNotFound) {
			entity.Result = carrierintel.VerificationResultNotFound
			entity.MismatchReason = "No carrier is registered to this equipment"
			return
		}
		entity.Result = carrierintel.VerificationResultProviderError
		entity.MismatchReason = errorSummary(toBusinessError(bound.provider, err))
		return
	}

	applyEquipmentMatches(entity, result, carrierName)
}

func applyEquipmentMatches(
	entity *carrierintel.CarrierEquipmentVerification,
	result *services.CarrierIntelEquipmentResult,
	carrierName string,
) {
	if result == nil || len(result.Matches) == 0 {
		entity.Result = carrierintel.VerificationResultNotFound
		entity.MismatchReason = "No carrier is registered to this equipment"
		return
	}

	expected := stringutils.DigitsOnly(entity.ExpectedDOTNumber)
	dots := make([]string, 0, len(result.Matches))
	names := make([]string, 0, len(result.Matches))
	for _, match := range result.Matches {
		dot := stringutils.DigitsOnly(match.DOTNumber)
		if dot == "" || slices.Contains(dots, dot) {
			continue
		}
		dots = append(dots, dot)
		if match.LegalName != "" {
			names = append(names, match.LegalName)
		}
		if dot == expected && match.Unit != nil {
			unit := *match.Unit
			entity.Detail = &unit
		}
	}
	entity.MatchedDOTNumbers = dots
	if len(names) > 0 {
		entity.MatchedLegalName = stringutils.TruncateRunes(strings.Join(names, "; "), 255)
	}

	if slices.Contains(dots, expected) {
		entity.Result = carrierintel.VerificationResultMatch
		return
	}
	entity.Result = carrierintel.VerificationResultMismatch
	entity.MismatchReason = fmt.Sprintf(
		"Equipment is registered to USDOT %s (%s), not %s (USDOT %s)",
		strings.Join(dots, ", "), entity.MatchedLegalName, carrierName, expected,
	)
	if entity.Detail == nil && result.Matches[0].Unit != nil {
		unit := *result.Matches[0].Unit
		entity.Detail = &unit
	}
}

func (s *Service) flagEquipmentMismatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	verification *carrierintel.CarrierEquipmentVerification,
	carrierName string,
) {
	now := s.now()
	provider := verification.Provider
	event := &carrierintel.CarrierIntelEvent{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		SubjectType:    carrierintel.SubjectTypeCarrier,
		SubjectID:      verification.CarrierID.String(),
		CarrierID:      verification.CarrierID,
		DOTNumber:      verification.ExpectedDOTNumber,
		SubjectName:    carrierName,
		Provider:       provider,
		Source:         carrierintel.EventSourceEquipmentVerification,
		Category:       carrierintel.SectionEquipment,
		FieldPath:      "equipment.verification",
		Severity:       carrierintel.SeverityHigh,
		CurrentValue:   verification.MatchedDOTNumbers,
		Summary:        verification.MismatchReason,
		DetectedAt:     now,
		Status:         carrierintel.EventStatusOpen,
		Fingerprint: carrierintel.EventFingerprint(carrierintel.EventFingerprintInput{
			Provider:    provider,
			SubjectType: carrierintel.SubjectTypeCarrier,
			SubjectID:   verification.CarrierID.String(),
			Discriminat: "equipment:" + verification.ID.String(),
			Current:     verification.MatchedDOTNumbers,
			Day:         timeutils.DayIndexUTC(now),
		}),
	}
	if _, err := s.eventRepo.InsertIgnoreDuplicates(ctx, []*carrierintel.CarrierIntelEvent{event}); err != nil {
		s.l.Warn("failed to record equipment mismatch event", zap.Error(err))
	}

	s.sendNotification(ctx, &notificationRequest{
		tenant:      tenantInfo,
		eventType:   EventCarrierEquipmentMismatch,
		correlation: "ci-equipment-" + verification.ID.String(),
		priority:    notification.PriorityCritical,
		title:       "Equipment does not belong to " + carrierName,
		message:     verification.MismatchReason + ". Hold the load until the carrier is confirmed.",
		link:        carrierLinkPrefix + verification.CarrierID.String(),
		related: map[string]any{
			"carrierId":           verification.CarrierID.String(),
			"carrierAssignmentId": verification.CarrierAssignmentID.String(),
			"shipmentMoveId":      verification.ShipmentMoveID.String(),
		},
	})
}

type OverrideVerificationRequest struct {
	TenantInfo     pagination.TenantInfo
	VerificationID pulid.ID
	Reason         string
}

func (s *Service) OverrideVerification(
	ctx context.Context,
	req *OverrideVerificationRequest,
) (*carrierintel.CarrierEquipmentVerification, error) {
	entity, err := s.verifyRepo.GetByID(ctx, req.TenantInfo, req.VerificationID)
	if err != nil {
		return nil, err
	}
	original := *entity
	if err = entity.Override(req.TenantInfo.UserID, req.Reason, s.now()); err != nil {
		return nil, err
	}
	updated, err := s.verifyRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceEquipmentVerification,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpApprove,
		UserID:         req.TenantInfo.UserID,
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    req.TenantInfo.UserID,
		CurrentState:   jsonutils.MustToJSON(updated),
		PreviousState:  jsonutils.MustToJSON(&original),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
		Critical:       true,
	}, auditservice.WithComment("Equipment verification overridden")); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	s.publish(ctx, req.TenantInfo, "carrier_equipment_verifications", "updated", updated.ID)
	return updated, nil
}

func (s *Service) VerificationsByAssignmentIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assignmentIDs []pulid.ID,
) ([]*carrierintel.CarrierEquipmentVerification, error) {
	return s.verifyRepo.ListByAssignmentIDs(ctx, tenantInfo, assignmentIDs)
}
