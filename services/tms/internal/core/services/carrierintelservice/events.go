package carrierintelservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const maxBulkEventIDs = 500

func (s *Service) ListEvents(
	ctx context.Context,
	req *repositories.ListCarrierIntelEventsRequest,
) (*pagination.CursorListResult[*carrierintel.CarrierIntelEvent], error) {
	return s.eventRepo.ListConnection(ctx, req)
}

func (s *Service) EventCounts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*repositories.CarrierIntelEventCounts, error) {
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
		&repositories.BulkAcknowledgeCarrierIntelEventsRequest{
			TenantInfo:     tenantInfo,
			EventIDs:       eventIDs,
			UserID:         tenantInfo.UserID,
			AcknowledgedAt: s.now(),
		},
	)
	if err != nil {
		return 0, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:      permission.ResourceCarrierIntelligence,
		ResourceID:    "events",
		Operation:     permission.OpUpdate,
		UserID:        tenantInfo.UserID,
		PrincipalType: services.PrincipalTypeUser,
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

	s.publish(ctx, tenantInfo, "carrier_intel_events", "updated", pulid.Nil)
	return count, nil
}

type ResolveEventRequest struct {
	TenantInfo pagination.TenantInfo
	EventID    pulid.ID
	Resolution carrierintel.EventResolution
	Note       string
}

func (s *Service) ResolveEvent(
	ctx context.Context,
	req *ResolveEventRequest,
) (*carrierintel.CarrierIntelEvent, error) {
	note := strings.TrimSpace(req.Note)
	if len(note) > 2000 {
		return nil, errortypes.NewValidationError("note", errortypes.ErrInvalid,
			"Note cannot exceed 2000 characters")
	}
	if req.Resolution == carrierintel.EventResolutionFalsePositive && note == "" {
		return nil, errortypes.NewValidationError("note", errortypes.ErrRequired,
			"Explain why this change is a false positive")
	}

	events, err := s.eventRepo.GetByIDs(ctx, req.TenantInfo, []pulid.ID{req.EventID})
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, errortypes.NewNotFoundError("Carrier intelligence event not found")
	}
	event := events[0]
	original := *event

	if err = event.Resolve(req.TenantInfo.UserID, req.Resolution, note, s.now()); err != nil {
		return nil, err
	}
	updated, err := s.eventRepo.Update(ctx, event)
	if err != nil {
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceCarrierIntelligence,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         req.TenantInfo.UserID,
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    req.TenantInfo.UserID,
		CurrentState:   jsonutils.MustToJSON(updated),
		PreviousState:  jsonutils.MustToJSON(&original),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	}, auditservice.WithComment("Carrier intelligence event resolved")); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	s.publish(ctx, req.TenantInfo, "carrier_intel_events", "updated", updated.ID)
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
	if err := s.snapshotRepo.MarkReviewed(ctx, &repositories.MarkSnapshotReviewedRequest{
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
		repositories.CarrierIntelSubjectRef{
			SubjectType: subjectType,
			SubjectID:   subjectID,
		},
	)
	if err != nil {
		return err
	}
	if snapshot != nil && subjectType == carrierintel.SubjectTypeCarrier {
		if err = s.updateCarrierSummary(ctx, req.TenantInfo, repositories.CarrierIntelSubject{
			SubjectType: subjectType,
			SubjectID:   subjectID,
			CarrierID:   snapshot.CarrierID,
		}, snapshot); err != nil {
			return err
		}
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:      permission.ResourceCarrierIntelligence,
		ResourceID:    subjectID,
		Operation:     permission.OpUpdate,
		UserID:        req.TenantInfo.UserID,
		PrincipalType: services.PrincipalTypeUser,
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

	s.publish(ctx, req.TenantInfo, "carrier_intelligence", "updated", pulid.Nil)
	if req.CarrierID.IsNotNil() {
		s.publish(ctx, req.TenantInfo, "carriers", "updated", req.CarrierID)
	}
	return nil
}

func (s *Service) ReviewQueue(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	limit int,
) ([]*carrierintel.CarrierIntelSnapshot, error) {
	return s.snapshotRepo.ListReviewQueue(ctx, &repositories.ListCarrierIntelReviewQueueRequest{
		TenantInfo: tenantInfo,
		Limit:      limit,
	})
}

func (s *Service) SnapshotHistory(
	ctx context.Context,
	req *repositories.ListCarrierIntelSnapshotHistoryRequest,
) ([]*carrierintel.CarrierIntelSnapshot, error) {
	return s.snapshotRepo.ListHistory(ctx, req)
}

func (s *Service) CurrentSnapshot(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	subject repositories.CarrierIntelSubjectRef,
) (*carrierintel.CarrierIntelSnapshot, error) {
	return s.snapshotRepo.GetCurrent(ctx, tenantInfo, subject)
}

func (s *Service) CurrentSnapshotsByCarrierIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierIDs []pulid.ID,
) ([]*carrierintel.CarrierIntelSnapshot, error) {
	return s.snapshotRepo.GetCurrentByCarrierIDs(ctx, tenantInfo, carrierIDs)
}

func (s *Service) CurrentSnapshotsBySubjects(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	subjectType carrierintel.SubjectType,
	subjectIDs []string,
) ([]*carrierintel.CarrierIntelSnapshot, error) {
	return s.snapshotRepo.GetCurrentBySubjectIDs(ctx, tenantInfo, subjectType, subjectIDs)
}

func (s *Service) OpenEventCountsByCarrierIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierIDs []pulid.ID,
) (map[pulid.ID]int, error) {
	return s.eventRepo.CountOpenByCarrierIDs(ctx, tenantInfo, carrierIDs)
}

func (s *Service) EnrollmentsByCarrierIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierIDs []pulid.ID,
) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
	return s.enrollmentRepo.GetByCarrierIDs(ctx, tenantInfo, carrierIDs)
}

func (s *Service) ListEnrollments(
	ctx context.Context,
	req *repositories.ListEnrollmentConnectionRequest,
) (*pagination.CursorListResult[*carrierintel.CarrierMonitoringEnrollment], error) {
	return s.enrollmentRepo.ListConnection(ctx, req)
}

func (s *Service) EnrollmentCounts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*repositories.CarrierIntelEnrollmentCounts, error) {
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return &repositories.CarrierIntelEnrollmentCounts{}, nil
	}
	return s.enrollmentRepo.Counts(ctx, tenantInfo, provider)
}

func (s *Service) FeedStates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*carrierintel.CarrierIntelFeedState, error) {
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return []*carrierintel.CarrierIntelFeedState{}, nil
	}
	states := make([]*carrierintel.CarrierIntelFeedState, 0, 2)
	for _, feedType := range []carrierintel.FeedType{
		carrierintel.FeedTypeChangeFeed, carrierintel.FeedTypeSnapshotRefresh,
	} {
		state, getErr := s.feedRepo.Get(ctx, repositories.CarrierIntelFeedStateKey{
			TenantInfo: tenantInfo, Provider: provider, FeedType: feedType,
		})
		if getErr != nil {
			return nil, getErr
		}
		if state != nil {
			states = append(states, state)
		}
	}
	return states, nil
}

func (s *Service) RawPayload(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	snapshotID pulid.ID,
	subject repositories.CarrierIntelSubjectRef,
) (string, error) {
	history, err := s.snapshotRepo.ListHistory(
		ctx,
		&repositories.ListCarrierIntelSnapshotHistoryRequest{
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
		if err = s.auditService.LogAction(&services.LogActionParams{
			Resource:       permission.ResourceCarrierIntelligence,
			ResourceID:     snap.ID.String(),
			Operation:      permission.OpExport,
			UserID:         tenantInfo.UserID,
			PrincipalType:  services.PrincipalTypeUser,
			PrincipalID:    tenantInfo.UserID,
			CurrentState:   map[string]any{"rawPayloadId": raw.ID.String()},
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Critical:       true,
		}, auditservice.WithComment("Carrier intelligence raw payload exported")); err != nil {
			s.l.Error("failed to log audit action", zap.Error(err))
		}
		return raw.Payload, nil
	}
	return "", errortypes.NewNotFoundError("Carrier intelligence snapshot not found")
}
