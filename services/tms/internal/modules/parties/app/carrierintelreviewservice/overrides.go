package carrierintelreviewservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelcontrol"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelevent"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierinteloverride"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelrule"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
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
	RuleCode   carrierintelrule.RuleCode
	Reason     string
	ExpiresAt  int64
}

func (s *Service) GrantOverride(
	ctx context.Context,
	req *GrantOverrideRequest,
) (*carrierinteloverride.CarrierIntelOverride, error) {
	now := s.now()
	expires := req.ExpiresAt
	if expires == 0 {
		expires = now + carrierintelcontrol.DefaultOverrideDurationSecond
	}

	entity := &carrierinteloverride.CarrierIntelOverride{
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

	if _, err := s.carrierRepo.GetByID(ctx, carrierports.GetCarrierByIDRequest{
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
	s.intel.RefreshStoredEvaluation(ctx, req.TenantInfo, req.CarrierID)
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
) (*carrierinteloverride.CarrierIntelOverride, error) {
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
	s.intel.RefreshStoredEvaluation(ctx, req.TenantInfo, updated.CarrierID)
	return updated, nil
}

func (s *Service) ListOverrides(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierID pulid.ID,
) ([]*carrierinteloverride.CarrierIntelOverride, error) {
	return s.overrideRepo.ListByCarrier(ctx, tenantInfo, carrierID)
}

func (s *Service) auditOverride(
	tenantInfo pagination.TenantInfo,
	current, previous *carrierinteloverride.CarrierIntelOverride,
	comment string,
) {
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceCarrierIntelligence,
		ResourceID:     current.ID.String(),
		Operation:      permission.OpApprove,
		UserID:         tenantInfo.UserID,
		PrincipalType:  permissionports.PrincipalTypeUser,
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
	override *carrierinteloverride.CarrierIntelOverride,
	summary string,
) {
	snapshot, err := s.snapshotRepo.GetCurrent(
		ctx,
		tenantInfo,
		carrierintelports.CarrierIntelSubjectRef{
			SubjectType: carrierintel.SubjectTypeCarrier,
			SubjectID:   override.CarrierID.String(),
		},
	)
	if err != nil || snapshot == nil {
		return
	}
	def, _ := carrierintelrule.RuleByCode(override.RuleCode)
	category := carrierintel.SectionIdentity
	if def != nil {
		category = def.Category
	}
	now := s.now()
	event := &carrierintelevent.CarrierIntelEvent{
		OrganizationID:   tenantInfo.OrgID,
		BusinessUnitID:   tenantInfo.BuID,
		SubjectType:      carrierintel.SubjectTypeCarrier,
		SubjectID:        override.CarrierID.String(),
		CarrierID:        override.CarrierID,
		DOTNumber:        snapshot.DOTNumber,
		Provider:         snapshot.Provider,
		Source:           carrierintelevent.EventSourceOverride,
		Category:         category,
		RuleCode:         override.RuleCode,
		Severity:         carrierintel.SeverityInfo,
		Summary:          summary,
		DetectedAt:       now,
		SnapshotID:       snapshot.ID,
		Status:           carrierintelevent.EventStatusAcknowledged,
		AcknowledgedByID: tenantInfo.UserID,
		AcknowledgedAt:   &now,
		Fingerprint: carrierintelevent.EventFingerprint(&carrierintelevent.EventFingerprintInput{
			Provider:    snapshot.Provider,
			SubjectType: carrierintel.SubjectTypeCarrier,
			SubjectID:   override.CarrierID.String(),
			Discriminat: "override:" + override.ID.String() + ":" + summary,
			Day:         timeutils.DayIndexUTC(now),
		}),
	}
	if _, err = s.eventRepo.InsertIgnoreDuplicates(
		ctx,
		[]*carrierintelevent.CarrierIntelEvent{event},
	); err != nil {
		s.l.Warn("failed to record override event", zap.Error(err))
	}
}
