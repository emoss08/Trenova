package carrierintelservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type GrantOverrideRequest struct {
	TenantInfo pagination.TenantInfo
	CarrierID  pulid.ID
	RuleCode   carrierintel.RuleCode
	Reason     string
	ExpiresAt  int64
}

func (s *Service) GrantOverride(
	ctx context.Context,
	req *GrantOverrideRequest,
) (*carrierintel.CarrierIntelOverride, error) {
	now := s.now()
	expires := req.ExpiresAt
	if expires == 0 {
		expires = now + carrierintel.DefaultOverrideDurationSecond
	}

	entity := &carrierintel.CarrierIntelOverride{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		CarrierID:      req.CarrierID,
		RuleCode:       req.RuleCode,
		Reason:         req.Reason,
		GrantedByID:    req.TenantInfo.UserID,
		GrantedAt:      now,
		ExpiresAt:      expires,
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if _, err := s.carrierRepo.GetByID(ctx, repositories.GetCarrierByIDRequest{
		ID:         req.CarrierID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, err
	}

	existing, err := s.overrideRepo.ListActiveByCarrierIDs(
		ctx, req.TenantInfo, []pulid.ID{req.CarrierID}, now,
	)
	if err != nil {
		return nil, err
	}
	for _, o := range existing {
		if o.RuleCode == req.RuleCode {
			return nil, errortypes.NewBusinessError(
				"An active override for this finding already exists until {0}",
				timeutils.FormatUnixDate(o.ExpiresAt),
			)
		}
	}

	created, err := s.overrideRepo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.auditOverride(req.TenantInfo, created, nil, "Carrier intelligence override granted")
	s.recordOverrideEvent(ctx, req.TenantInfo, created, fmt.Sprintf(
		"Override granted for %s until %s: %s",
		req.RuleCode, timeutils.FormatUnixDate(created.ExpiresAt), created.Reason,
	))
	s.refreshStoredEvaluation(ctx, req.TenantInfo, req.CarrierID)
	return created, nil
}

type RevokeOverrideRequest struct {
	TenantInfo pagination.TenantInfo
	OverrideID pulid.ID
	Reason     string
}

func (s *Service) RevokeOverride(
	ctx context.Context,
	req *RevokeOverrideRequest,
) (*carrierintel.CarrierIntelOverride, error) {
	entity, err := s.overrideRepo.GetByID(ctx, req.TenantInfo, req.OverrideID)
	if err != nil {
		return nil, err
	}
	original := *entity
	if err = entity.Revoke(req.TenantInfo.UserID, req.Reason, s.now()); err != nil {
		return nil, err
	}
	updated, err := s.overrideRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.auditOverride(req.TenantInfo, updated, &original, "Carrier intelligence override revoked")
	s.recordOverrideEvent(ctx, req.TenantInfo, updated, fmt.Sprintf(
		"Override for %s revoked: %s", updated.RuleCode, updated.RevokeReason,
	))
	s.refreshStoredEvaluation(ctx, req.TenantInfo, updated.CarrierID)
	return updated, nil
}

func (s *Service) ListOverrides(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierID pulid.ID,
) ([]*carrierintel.CarrierIntelOverride, error) {
	return s.overrideRepo.ListByCarrier(ctx, tenantInfo, carrierID)
}

func (s *Service) auditOverride(
	tenantInfo pagination.TenantInfo,
	current, previous *carrierintel.CarrierIntelOverride,
	comment string,
) {
	params := &services.LogActionParams{
		Resource:       permission.ResourceCarrierIntelligence,
		ResourceID:     current.ID.String(),
		Operation:      permission.OpApprove,
		UserID:         tenantInfo.UserID,
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    tenantInfo.UserID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Critical:       true,
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}
	if err := s.auditService.LogAction(params, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}
}

func (s *Service) recordOverrideEvent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	override *carrierintel.CarrierIntelOverride,
	summary string,
) {
	snapshot, err := s.snapshotRepo.GetCurrent(ctx, tenantInfo, repositories.CarrierIntelSubjectRef{
		SubjectType: carrierintel.SubjectTypeCarrier,
		SubjectID:   override.CarrierID.String(),
	})
	if err != nil || snapshot == nil {
		return
	}
	def, _ := carrierintel.RuleByCode(override.RuleCode)
	category := carrierintel.SectionIdentity
	if def != nil {
		category = def.Category
	}
	now := s.now()
	event := &carrierintel.CarrierIntelEvent{
		OrganizationID:   tenantInfo.OrgID,
		BusinessUnitID:   tenantInfo.BuID,
		SubjectType:      carrierintel.SubjectTypeCarrier,
		SubjectID:        override.CarrierID.String(),
		CarrierID:        override.CarrierID,
		DOTNumber:        snapshot.DOTNumber,
		Provider:         snapshot.Provider,
		Source:           carrierintel.EventSourceOverride,
		Category:         category,
		RuleCode:         override.RuleCode,
		Severity:         carrierintel.SeverityInfo,
		Summary:          summary,
		DetectedAt:       now,
		SnapshotID:       snapshot.ID,
		Status:           carrierintel.EventStatusAcknowledged,
		AcknowledgedByID: tenantInfo.UserID,
		AcknowledgedAt:   &now,
		Fingerprint: carrierintel.EventFingerprint(carrierintel.EventFingerprintInput{
			Provider:    snapshot.Provider,
			SubjectType: carrierintel.SubjectTypeCarrier,
			SubjectID:   override.CarrierID.String(),
			Discriminat: "override:" + override.ID.String() + ":" + summary,
			Day:         timeutils.DayIndexUTC(now),
		}),
	}
	if _, err = s.eventRepo.InsertIgnoreDuplicates(ctx, []*carrierintel.CarrierIntelEvent{event}); err != nil {
		s.l.Warn("failed to record override event", zap.Error(err))
	}
}

func (s *Service) refreshStoredEvaluation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierID pulid.ID,
) {
	snapshot, err := s.snapshotRepo.GetCurrent(ctx, tenantInfo, repositories.CarrierIntelSubjectRef{
		SubjectType: carrierintel.SubjectTypeCarrier,
		SubjectID:   carrierID.String(),
	})
	if err != nil || snapshot == nil {
		return
	}
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return
	}
	subject, err := s.subjectForSnapshot(ctx, tenantInfo, snapshot)
	if err != nil {
		return
	}
	findings := s.recomputeFindings(ctx, tenantInfo, control, subject, snapshot)
	prior := snapshot.Findings
	snapshot.ApplyFindings(findings, control.PolicyVersion)
	snapshot.ReviewState = nextReviewState(snapshot, prior, findings)
	if err = s.snapshotRepo.UpdateEvaluation(ctx, snapshot); err != nil {
		s.l.Warn("failed to refresh stored findings after override change", zap.Error(err))
		return
	}
	if err = s.updateCarrierSummary(ctx, tenantInfo, subject, snapshot); err != nil {
		s.l.Warn("failed to update carrier summary after override change", zap.Error(err))
	}
	s.publish(ctx, tenantInfo, "carriers", "updated", carrierID)
	s.publish(ctx, tenantInfo, "carrier_intelligence", "updated", snapshot.ID)
}
