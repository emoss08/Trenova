package carrierintelreviewservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelevent"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelsnapshot"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const maxBulkEventIDs = 500

func (s *Service) ListEvents(
	ctx context.Context,
	req *carrierintelports.ListCarrierIntelEventsRequest,
) (*pagination.CursorListResult[*carrierintelevent.CarrierIntelEvent], error) {
	return s.eventRepo.ListConnection(ctx, req)
}

func (s *Service) GetEvent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	eventID pulid.ID,
) (*carrierintelevent.CarrierIntelEvent, error) {
	events, err := s.eventRepo.GetByIDs(ctx, tenantInfo, []pulid.ID{eventID})
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, errortypes.NewNotFoundError("Carrier intelligence event not found")
	}
	return events[0], nil
}

func (s *Service) EventCounts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*carrierintelports.CarrierIntelEventCounts, error) {
	return s.eventRepo.CountOpen(ctx, tenantInfo)
}

func (s *Service) AcknowledgeEvents(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	eventIDs []pulid.ID,
) (int, error) {
	if len(eventIDs) == 0 {
		return 0, errortypes.NewValidationError("eventIds", errortypes.ErrRequired,
			"Select at least one event")
	}
	if len(eventIDs) > maxBulkEventIDs {
		return 0, errortypes.NewValidationError("eventIds", errortypes.ErrInvalid,
			"Acknowledge at most {0} events at a time", maxBulkEventIDs)
	}

	count, err := s.eventRepo.BulkAcknowledge(
		ctx,
		&carrierintelports.BulkAcknowledgeCarrierIntelEventsRequest{
			TenantInfo:     tenantInfo,
			EventIDs:       eventIDs,
			UserID:         tenantInfo.UserID,
			AcknowledgedAt: s.now(),
		},
	)
	if err != nil {
		return 0, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:      permission.ResourceCarrierIntelligence,
		ResourceID:    "events",
		Operation:     permission.OpUpdate,
		UserID:        tenantInfo.UserID,
		PrincipalType: permissionports.PrincipalTypeUser,
		PrincipalID:   tenantInfo.UserID,
		CurrentState: map[string]any{
			"acknowledged": count,
			"eventIds":     pulid.Map(eventIDs, func(id pulid.ID) string { return id.String() }),
		},
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}, auditservice.WithComment("Carrier intelligence events acknowledged")); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	s.intel.Publish(ctx, tenantInfo, "carrier_intel_events", "updated", pulid.Nil)
	return count, nil
}

type ResolveEventRequest struct {
	TenantInfo pagination.TenantInfo
	EventID    pulid.ID
	Resolution carrierintelevent.EventResolution
	Note       string
}

func PlanResolve(
	event *carrierintelevent.CarrierIntelEvent,
	req *ResolveEventRequest,
	now int64,
) error {
	if err := checkResolveNote(req); err != nil {
		return err
	}

	return event.Resolve(req.TenantInfo.UserID, req.Resolution, strings.TrimSpace(req.Note), now)
}

func checkResolveNote(req *ResolveEventRequest) error {
	note := strings.TrimSpace(req.Note)
	if len(note) > 2000 {
		return errortypes.NewValidationError("note", errortypes.ErrInvalid,
			"Note cannot exceed 2000 characters")
	}
	if req.Resolution == carrierintelevent.EventResolutionFalsePositive && note == "" {
		return errortypes.NewValidationError("note", errortypes.ErrRequired,
			"Explain why this change is a false positive")
	}

	return nil
}

func (s *Service) ResolveEvent(
	ctx context.Context,
	req *ResolveEventRequest,
) (*carrierintelevent.CarrierIntelEvent, error) {
	if err := checkResolveNote(req); err != nil {
		return nil, err
	}

	event, err := s.GetEvent(ctx, req.TenantInfo, req.EventID)
	if err != nil {
		return nil, err
	}
	original := *event

	if err = PlanResolve(event, req, s.now()); err != nil {
		return nil, err
	}
	updated, err := s.eventRepo.Update(ctx, event)
	if err != nil {
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceCarrierIntelligence,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         req.TenantInfo.UserID,
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    req.TenantInfo.UserID,
		CurrentState:   jsonutils.MustToJSON(updated),
		PreviousState:  jsonutils.MustToJSON(&original),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	}, auditservice.WithComment("Carrier intelligence event resolved")); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	s.intel.Publish(ctx, req.TenantInfo, "carrier_intel_events", "updated", updated.ID)
	if s.watchtower != nil {
		s.watchtower.Resolve(
			ctx,
			req.TenantInfo,
			watchtower.SourceCarrierIntelEvent,
			updated.ID.String(),
		)
	}

	return updated, nil
}

type MarkReviewedRequest struct {
	TenantInfo  pagination.TenantInfo
	SubjectType carrierintel.SubjectType
	SubjectID   string
	CarrierID   pulid.ID
	Note        string
}

func (s *Service) MarkReviewed(ctx context.Context, req *MarkReviewedRequest) error {
	note := strings.TrimSpace(req.Note)
	if note == "" {
		return errortypes.NewValidationError("note", errortypes.ErrRequired,
			"Record what was reviewed")
	}
	if len(note) > 2000 {
		return errortypes.NewValidationError("note", errortypes.ErrInvalid,
			"Note cannot exceed 2000 characters")
	}
	subjectType := req.SubjectType
	subjectID := req.SubjectID
	if req.CarrierID.IsNotNil() {
		subjectType = carrierintel.SubjectTypeCarrier
		subjectID = req.CarrierID.String()
	}

	now := s.now()
	if err := s.snapshotRepo.MarkReviewed(ctx, &carrierintelports.MarkSnapshotReviewedRequest{
		TenantInfo:  req.TenantInfo,
		SubjectType: subjectType,
		SubjectID:   subjectID,
		UserID:      req.TenantInfo.UserID,
		Note:        note,
		ReviewedAt:  now,
	}); err != nil {
		return err
	}

	snapshot, err := s.snapshotRepo.GetCurrent(
		ctx,
		req.TenantInfo,
		carrierintelports.CarrierIntelSubjectRef{
			SubjectType: subjectType,
			SubjectID:   subjectID,
		},
	)
	if err != nil {
		return err
	}
	if snapshot != nil && subjectType == carrierintel.SubjectTypeCarrier {
		if err = s.intel.UpdateCarrierSummary(
			ctx,
			req.TenantInfo,
			carrierintelports.CarrierIntelSubject{
				SubjectType: subjectType,
				SubjectID:   subjectID,
				CarrierID:   snapshot.CarrierID,
			},
			snapshot,
		); err != nil {
			return err
		}
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:      permission.ResourceCarrierIntelligence,
		ResourceID:    subjectID,
		Operation:     permission.OpUpdate,
		UserID:        req.TenantInfo.UserID,
		PrincipalType: permissionports.PrincipalTypeUser,
		PrincipalID:   req.TenantInfo.UserID,
		CurrentState: map[string]any{
			"subjectType": subjectType,
			"subjectId":   subjectID,
			"reviewNote":  note,
			"reviewedAt":  now,
		},
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Critical:       true,
	}, auditservice.WithComment("Carrier intelligence findings reviewed")); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	s.intel.Publish(ctx, req.TenantInfo, "carrier_intelligence", "updated", pulid.Nil)
	if req.CarrierID.IsNotNil() {
		s.intel.Publish(ctx, req.TenantInfo, "carriers", "updated", req.CarrierID)
	}
	return nil
}

func (s *Service) ReviewQueue(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	limit int,
) ([]*carrierintelsnapshot.CarrierIntelSnapshot, error) {
	return s.snapshotRepo.ListReviewQueue(
		ctx,
		&carrierintelports.ListCarrierIntelReviewQueueRequest{
			TenantInfo: tenantInfo,
			Limit:      limit,
		},
	)
}

func (s *Service) SnapshotHistory(
	ctx context.Context,
	req *carrierintelports.ListCarrierIntelSnapshotHistoryRequest,
) ([]*carrierintelsnapshot.CarrierIntelSnapshot, error) {
	return s.snapshotRepo.ListHistory(ctx, req)
}

func (s *Service) CurrentSnapshot(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	subject carrierintelports.CarrierIntelSubjectRef,
) (*carrierintelsnapshot.CarrierIntelSnapshot, error) {
	return s.snapshotRepo.GetCurrent(ctx, tenantInfo, subject)
}

func (s *Service) CurrentSnapshotsByCarrierIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierIDs []pulid.ID,
) ([]*carrierintelsnapshot.CarrierIntelSnapshot, error) {
	return s.snapshotRepo.GetCurrentByCarrierIDs(ctx, tenantInfo, carrierIDs)
}

func (s *Service) CurrentSnapshotsBySubjects(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	subjectType carrierintel.SubjectType,
	subjectIDs []string,
) ([]*carrierintelsnapshot.CarrierIntelSnapshot, error) {
	return s.snapshotRepo.GetCurrentBySubjectIDs(ctx, tenantInfo, subjectType, subjectIDs)
}

func (s *Service) OpenEventCountsByCarrierIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierIDs []pulid.ID,
) (map[pulid.ID]int, error) {
	return s.eventRepo.CountOpenByCarrierIDs(ctx, tenantInfo, carrierIDs)
}

func (s *Service) RawPayload(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	snapshotID pulid.ID,
	subject carrierintelports.CarrierIntelSubjectRef,
) (string, error) {
	history, err := s.snapshotRepo.ListHistory(
		ctx,
		&carrierintelports.ListCarrierIntelSnapshotHistoryRequest{
			TenantInfo:  tenantInfo,
			SubjectType: subject.SubjectType,
			SubjectID:   subject.SubjectID,
			Limit:       100,
		},
	)
	if err != nil {
		return "", err
	}
	for _, snap := range history {
		if snap.ID != snapshotID {
			continue
		}
		if snap.RawPayloadID.IsNil() {
			return "", errortypes.NewNotFoundError("No raw payload is retained for this snapshot")
		}
		raw, rawErr := s.rawRepo.GetByID(ctx, tenantInfo, snap.RawPayloadID)
		if rawErr != nil {
			return "", rawErr
		}
		if err = s.auditService.LogAction(&auditports.LogActionParams{
			Resource:       permission.ResourceCarrierIntelligence,
			ResourceID:     snap.ID.String(),
			Operation:      permission.OpExport,
			UserID:         tenantInfo.UserID,
			PrincipalType:  permissionports.PrincipalTypeUser,
			PrincipalID:    tenantInfo.UserID,
			CurrentState:   map[string]any{"rawPayloadId": raw.ID.String()},
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Critical:       true,
		}, auditservice.WithComment("Carrier intelligence raw payload exported")); err != nil {
			s.l.Error("failed to log audit action", zap.Error(err))
		}
		return raw.Payload.String(), nil
	}
	return "", errortypes.NewNotFoundError("Carrier intelligence snapshot not found")
}
