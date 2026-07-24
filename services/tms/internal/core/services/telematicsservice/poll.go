package telematicsservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type TenantPollResult struct {
	PositionsUpserted int `json:"positionsUpserted"`
	HOSStatesUpserted int `json:"hosStatesUpserted"`
	UnmappedVehicles  int `json:"unmappedVehicles"`
	UnmappedDrivers   int `json:"unmappedDrivers"`
}

func (s *Service) PollTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*TenantPollResult, error) {
	provider, err := s.resolveProvider(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	result := new(TenantPollResult)
	posErr := s.pollVehiclePositions(ctx, tenantInfo, provider, result)
	hosErr := s.pollHOSClocks(ctx, tenantInfo, provider, result)
	return result, errors.Join(posErr, hosErr)
}

func (s *Service) pollVehiclePositions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider services.TelematicsProvider,
	result *TenantPollResult,
) error {
	providerType := string(provider.Type())

	tractorsByExternalID, err := s.tractorsByExternalID(ctx, tenantInfo)
	if err != nil {
		return s.recordFeedFailure(ctx, tenantInfo, providerType, err)
	}

	providerPositions, err := provider.ListPositions(ctx)
	if err != nil {
		return s.recordFeedFailure(ctx, tenantInfo, providerType, err)
	}

	now := timeutils.NowUnix()
	positions := make([]*telematics.VehiclePosition, 0, len(providerPositions))
	for i := range providerPositions {
		providerPosition := &providerPositions[i]
		tractorID, ok := tractorsByExternalID[providerPosition.VehicleID]
		if !ok {
			result.UnmappedVehicles++
			continue
		}

		positions = append(positions, &telematics.VehiclePosition{
			OrganizationID:    tenantInfo.OrgID,
			BusinessUnitID:    tenantInfo.BuID,
			TractorID:         tractorID,
			Provider:          providerType,
			ProviderVehicleID: providerPosition.VehicleID,
			Latitude:          providerPosition.Latitude,
			Longitude:         providerPosition.Longitude,
			HeadingDegrees:    providerPosition.HeadingDegrees,
			SpeedMph:          providerPosition.SpeedMph,
			EngineState:       providerPosition.EngineState,
			FuelPercent:       providerPosition.FuelPercent,
			OdometerMeters:    providerPosition.OdometerMeters,
			FormattedLocation: providerPosition.FormattedLocation,
			RecordedAt:        providerPosition.RecordedAt,
			ReceivedAt:        now,
		})
	}

	if err = s.repo.UpsertVehiclePositions(ctx, positions); err != nil {
		return s.recordFeedFailure(ctx, tenantInfo, providerType, err)
	}
	result.PositionsUpserted = len(positions)

	if err = s.recordFeedSuccess(ctx, tenantInfo, providerType); err != nil {
		s.l.Warn("failed to record telematics feed success",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.Error(err))
	}

	if len(positions) > 0 {
		s.publishInvalidation(ctx, tenantInfo, "vehiclePosition")
	}
	return nil
}

func (s *Service) pollHOSClocks(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider services.TelematicsProvider,
	result *TenantPollResult,
) error {
	workersByExternalID, err := s.workersByExternalID(ctx, tenantInfo)
	if err != nil {
		return err
	}
	tractorsByExternalID, err := s.tractorsByExternalID(ctx, tenantInfo)
	if err != nil {
		return err
	}

	clocks, err := provider.ListHOSClocks(ctx)
	if err != nil {
		return err
	}

	providerType := string(provider.Type())
	now := timeutils.NowUnix()
	states := make([]*telematics.WorkerHOSState, 0, len(clocks))
	for i := range clocks {
		clock := &clocks[i]
		workerID, ok := workersByExternalID[clock.DriverID]
		if !ok {
			result.UnmappedDrivers++
			continue
		}

		state := &telematics.WorkerHOSState{
			OrganizationID:          tenantInfo.OrgID,
			BusinessUnitID:          tenantInfo.BuID,
			WorkerID:                workerID,
			Provider:                providerType,
			ProviderDriverID:        clock.DriverID,
			DutyStatus:              clock.DutyStatus,
			DriveRemainingMs:        clock.DriveRemainingMs,
			ShiftRemainingMs:        clock.ShiftRemainingMs,
			CycleRemainingMs:        clock.CycleRemainingMs,
			CycleTomorrowMs:         clock.CycleTomorrowMs,
			BreakRemainingMs:        clock.BreakRemainingMs,
			CycleStartedAt:          clock.CycleStartedAt,
			ShiftDrivingViolationMs: clock.ShiftDrivingViolationMs,
			CycleViolationMs:        clock.CycleViolationMs,
			CurrentVehicleID:        clock.CurrentVehicleID,
			RecordedAt:              now,
			ReceivedAt:              now,
		}
		if tractorID, mapped := tractorsByExternalID[clock.CurrentVehicleID]; mapped {
			state.CurrentTractorID = tractorID
		}
		states = append(states, state)
	}

	if alertErr := s.notifyCriticalHOSTransitions(ctx, tenantInfo, states); alertErr != nil {
		s.l.Warn("failed to evaluate HOS alert transitions",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.Error(alertErr))
	}

	if err = s.repo.UpsertWorkerHOSStates(ctx, states); err != nil {
		return err
	}
	result.HOSStatesUpserted = len(states)

	if len(states) > 0 {
		s.publishInvalidation(ctx, tenantInfo, "workerHosState")
	}
	return nil
}

func (s *Service) tractorsByExternalID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (map[string]pulid.ID, error) {
	mappings, err := s.repo.ListTractorMappings(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	byExternalID := make(map[string]pulid.ID, len(mappings))
	for _, mapping := range mappings {
		if mapping.ExternalID != "" {
			byExternalID[mapping.ExternalID] = mapping.TractorID
		}
	}
	return byExternalID, nil
}

func (s *Service) workersByExternalID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (map[string]pulid.ID, error) {
	mappings, err := s.repo.ListWorkerMappings(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	byExternalID := make(map[string]pulid.ID, len(mappings))
	for _, mapping := range mappings {
		byExternalID[mapping.ExternalID] = mapping.WorkerID
	}
	return byExternalID, nil
}

func (s *Service) recordFeedSuccess(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	providerType string,
) error {
	now := timeutils.NowUnix()
	return s.repo.UpsertFeedState(ctx, &telematics.FeedState{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Provider:       providerType,
		FeedType:       telematics.FeedTypeVehicleStats,
		LastPolledAt:   now,
		LastSuccessAt:  now,
		FailureCount:   0,
	})
}

func (s *Service) recordFeedFailure(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	providerType string,
	cause error,
) error {
	state, err := s.repo.GetFeedState(
		ctx,
		tenantInfo,
		providerType,
		telematics.FeedTypeVehicleStats,
	)
	if err != nil {
		state = nil
		if !errortypes.IsNotFoundError(err) {
			s.l.Warn("failed to load telematics feed state",
				zap.String("organizationId", tenantInfo.OrgID.String()),
				zap.Error(err))
		}
	}

	now := timeutils.NowUnix()
	next := &telematics.FeedState{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Provider:       providerType,
		FeedType:       telematics.FeedTypeVehicleStats,
		LastPolledAt:   now,
		FailureCount:   1,
		LastError:      cause.Error(),
	}
	if state != nil {
		next.LastSuccessAt = state.LastSuccessAt
		next.FailureCount = state.FailureCount + 1
	}
	if upsertErr := s.repo.UpsertFeedState(ctx, next); upsertErr != nil {
		s.l.Warn("failed to record telematics feed failure",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.Error(upsertErr))
	}
	return cause
}
