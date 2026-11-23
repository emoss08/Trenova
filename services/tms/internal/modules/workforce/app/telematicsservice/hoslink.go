package telematicsservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) WorkerHOSLink(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*telematics.HOSLinkState, error) {
	onPlan, err := s.samsaraOnPlan(ctx)
	if err != nil {
		return nil, err
	}
	if !onPlan {
		return &telematics.HOSLinkState{Link: telematics.HOSLinkNotConnected}, nil
	}
	_, connected, err := s.providerFactory.ConnectedType(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if !connected {
		return &telematics.HOSLinkState{Link: telematics.HOSLinkNotConnected}, nil
	}

	externalID, err := s.workerExternalID(ctx, tenantInfo, workerID)
	if err != nil {
		return nil, err
	}
	if externalID == "" {
		return &telematics.HOSLinkState{Link: telematics.HOSLinkNotLinked}, nil
	}

	state, err := s.repo.GetWorkerHOSState(ctx, telematicsports.GetWorkerHOSStateRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return &telematics.HOSLinkState{Link: telematics.HOSLinkNoData}, nil
		}
		return nil, err
	}
	if err = s.attachDutyStatusSince(
		ctx,
		tenantInfo,
		[]*telematics.WorkerHOSState{state},
	); err != nil {
		return nil, err
	}

	limits := LimitsForRuleset(state.RulesetCycle, state.RulesetShift, state.RulesetJurisdiction)
	link := &telematics.HOSLinkState{
		Link:             telematics.HOSLinkReporting,
		DutyStatusSince:  state.DutyStatusSince,
		DriveRemainingMs: new(state.DriveRemainingMs),
		DriveLimitMs:     new(limits.DriveMs),
		ShiftRemainingMs: new(state.ShiftRemainingMs),
		ShiftLimitMs:     new(limits.ShiftMs),
		CycleRemainingMs: new(state.CycleRemainingMs),
		CycleLimitMs:     new(limits.CycleMs),
		BreakRemainingMs: new(state.BreakRemainingMs),
		BreakLimitMs:     new(limits.BreakMs),
		CurrentTractorID: state.CurrentTractorID,
		RecordedAt:       new(state.RecordedAt),
	}
	if state.DutyStatus != "" {
		link.DutyStatus = new(string(state.DutyStatus))
	}
	return link, nil
}

func (s *Service) attachDutyStatusSince(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	states []*telematics.WorkerHOSState,
) error {
	workerIDs := make([]pulid.ID, 0, len(states))
	for _, state := range states {
		if state != nil && state.DutyStatus != "" {
			workerIDs = append(workerIDs, state.WorkerID)
		}
	}
	if len(workerIDs) == 0 {
		return nil
	}
	open, err := s.repo.ListOpenWorkerHOSLogs(ctx, tenantInfo, workerIDs)
	if err != nil {
		return err
	}
	segments := make(map[pulid.ID]*telematics.WorkerHOSLog, len(open))
	for _, segment := range open {
		segments[segment.WorkerID] = segment
	}
	for _, state := range states {
		if state == nil {
			continue
		}
		segment, ok := segments[state.WorkerID]
		if !ok || segment.DutyStatus != state.DutyStatus {
			state.DutyStatusSince = nil
			continue
		}
		state.DutyStatusSince = new(segment.LogStartAt)
	}
	return nil
}
