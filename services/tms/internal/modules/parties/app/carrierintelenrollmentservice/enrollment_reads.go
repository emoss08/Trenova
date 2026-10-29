package carrierintelenrollmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelenrollment"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) EnrollmentsByCarrierIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierIDs []pulid.ID,
) ([]*carrierintelenrollment.CarrierMonitoringEnrollment, error) {
	return s.enrollmentRepo.GetByCarrierIDs(ctx, tenantInfo, carrierIDs)
}

func (s *Service) ListEnrollments(
	ctx context.Context,
	req *carrierintelports.ListEnrollmentConnectionRequest,
) (*pagination.CursorListResult[*carrierintelenrollment.CarrierMonitoringEnrollment], error) {
	return s.enrollmentRepo.ListConnection(ctx, req)
}

func (s *Service) EnrollmentCounts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*carrierintelports.CarrierIntelEnrollmentCounts, error) {
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return &carrierintelports.CarrierIntelEnrollmentCounts{}, nil
	}
	return s.enrollmentRepo.Counts(ctx, tenantInfo, provider)
}

func (s *Service) FeedStates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*carrierintelenrollment.CarrierIntelFeedState, error) {
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return []*carrierintelenrollment.CarrierIntelFeedState{}, nil
	}
	states := make([]*carrierintelenrollment.CarrierIntelFeedState, 0, 2)
	for _, feedType := range []carrierintelenrollment.FeedType{
		carrierintelenrollment.FeedTypeChangeFeed, carrierintelenrollment.FeedTypeSnapshotRefresh,
	} {
		state, getErr := s.feedRepo.Get(ctx, carrierintelports.CarrierIntelFeedStateKey{
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
