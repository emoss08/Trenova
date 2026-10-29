package carrierintelmonitorservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelenrollmentservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelcontrol"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelenrollment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelevent"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	changeFeedPageSize      = 500
	maxChangeFeedPages      = 40
	defaultRefreshBatch     = 200
	confirmBatchSize        = 500
	feedFailureBackoffSteps = 6
)

type PollResult struct {
	Skipped   bool   `json:"skipped"`
	Paused    bool   `json:"paused"`
	Changed   int    `json:"changed"`
	Refreshed int    `json:"refreshed"`
	Failed    int    `json:"failed"`
	Reason    string `json:"reason,omitempty"`
}

type HeartbeatFunc func(details ...any)

func (s *Service) PollChangeFeed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	heartbeat HeartbeatFunc,
) (*PollResult, error) {
	ctx = carrierintel.WithPurpose(ctx, carrierintel.PurposeMonitor)
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return &PollResult{Skipped: true, Reason: "no provider"}, nil
	}
	connector, ok := s.intel.Connector(provider)
	if !ok || !connector.Descriptor().Capabilities.Has(carrierintel.CapabilityNativeMonitoring) {
		return &PollResult{Skipped: true, Reason: "provider has no change feed"}, nil
	}

	key := carrierintelports.CarrierIntelFeedStateKey{
		TenantInfo: tenantInfo,
		Provider:   provider,
		FeedType:   carrierintelenrollment.FeedTypeChangeFeed,
	}
	state, err := s.feedRepo.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if state == nil {
		state = carrierintelenrollmentservice.NewFeedState(key)
	}
	now := s.now()
	if state.IsPaused() {
		return &PollResult{Paused: true, Reason: state.PausedReason.String()}, nil
	}
	if !state.DueAt(now) {
		return &PollResult{Skipped: true, Reason: "not due"}, nil
	}

	bound, err := s.intel.Bind(ctx, tenantInfo, provider, false)
	if err != nil {
		return nil, err
	}
	monitor, ok := bound.Client().(carrierintelports.CarrierIntelNativeMonitor)
	if !ok {
		return &PollResult{Skipped: true, Reason: "provider has no change feed"}, nil
	}

	since := now - 2*timeutils.SecondsPerDay
	if state.LastSuccessAt != nil {
		since = timeutils.DayStartUTC(*state.LastSuccessAt) - timeutils.SecondsPerDay
	}

	result := &PollResult{}
	changedRefs := make(map[string]carrierintelports.CarrierIntelChangedProfile, 64)
	for page := 1; page <= maxChangeFeedPages; page++ {
		var feed *carrierintelports.CarrierIntelChangeFeedPage
		listErr := s.intel.WithRateLimitWait(ctx, func() error {
			var callErr error
			feed, callErr = monitor.ListChanges(
				ctx,
				&carrierintelports.CarrierIntelChangeFeedRequest{
					Since:    since,
					Until:    now,
					Page:     page,
					PageSize: changeFeedPageSize,
				},
			)
			return callErr
		})
		if listErr != nil {
			return s.recordFeedFailure(ctx, &feedFailure{
				tenant:   tenantInfo,
				provider: provider,
				state:    state,
				control:  control,
				err:      listErr,
				result:   result,
			})
		}
		for _, item := range feed.Items {
			changedRefs[item.ProviderRef] = item
		}
		if heartbeat != nil {
			heartbeat(page, len(changedRefs))
		}
		if !feed.HasMore || len(feed.Items) == 0 {
			break
		}
	}

	if len(changedRefs) > 0 {
		refs := make([]string, 0, len(changedRefs))
		for ref := range changedRefs {
			refs = append(refs, ref)
		}
		enrollments, listErr := s.enrollmentRepo.ListByProviderRefs(ctx, tenantInfo, provider, refs)
		if listErr != nil {
			return nil, listErr
		}
		result.Changed = len(enrollments)

		subjects, resolveErr := s.resolveEnrollmentSubjects(ctx, tenantInfo, enrollments)
		if resolveErr != nil {
			return nil, resolveErr
		}
		for _, subject := range subjects {
			if ctx.Err() != nil {
				return result, ctx.Err()
			}
			_, fetchErr := s.intel.Fetch(ctx, &carrierintelservice.FetchRequest{
				TenantInfo: tenantInfo,
				Subject:    subject,
				MinDepth:   carrierintel.LookupDepthFMCSA,
				Purpose:    carrierintel.PurposeMonitor,
				Force:      true,
				Source:     carrierintelevent.EventSourceNativeChangeFeed,
				Control:    control,
			})
			if fetchErr != nil {
				if carrierintelenrollmentservice.IsPausingError(fetchErr) {
					return s.recordFeedFailure(ctx, &feedFailure{
						tenant:   tenantInfo,
						provider: provider,
						state:    state,
						control:  control,
						err:      fetchErr,
						result:   result,
					})
				}
				result.Failed++
				s.l.Info("change feed refresh failed",
					zap.String("subjectId", subject.SubjectID), zap.Error(fetchErr))
				continue
			}
			result.Refreshed++
			if heartbeat != nil {
				heartbeat(result.Refreshed)
			}
		}
	}

	if err = s.confirmUnchangedEnrollments(
		ctx,
		tenantInfo,
		provider,
		changedRefs,
		now,
	); err != nil {
		s.l.Warn("failed to confirm unchanged monitored carriers", zap.Error(err))
	}

	state.RecordSuccess(now, now+int64(control.PollIntervalMinutes)*60)
	if err = s.feedRepo.Upsert(ctx, state); err != nil {
		return nil, err
	}
	return result, nil
}

type feedFailure struct {
	tenant   pagination.TenantInfo
	provider integration.Type
	state    *carrierintelenrollment.CarrierIntelFeedState
	control  *carrierintelcontrol.CarrierIntelControl
	err      error
	result   *PollResult
}

func (s *Service) recordFeedFailure(
	ctx context.Context,
	failure *feedFailure,
) (*PollResult, error) {
	now := s.now()
	reason := carrierintelenrollmentservice.PauseReasonFor(failure.err)
	if reason != carrierintelenrollment.FeedPauseReasonNone {
		if err := s.enrollment.PauseFeed(
			ctx, failure.tenant, failure.provider, carrierintelenrollment.FeedTypeChangeFeed, reason,
			carrierintelservice.ErrorSummary(failure.err),
		); err != nil {
			return failure.result, err
		}
		failure.result.Paused = true
		failure.result.Reason = reason.String()
		return failure.result, nil
	}

	steps := min(failure.state.FailureCount+1, feedFailureBackoffSteps)
	backoff := int64(
		failure.control.PollIntervalMinutes,
	) * 60 / int64(
		feedFailureBackoffSteps,
	) * int64(
		steps,
	)
	failure.state.RecordFailure(
		carrierintelservice.ErrorSummary(failure.err),
		now,
		now+max(backoff, 300),
	)
	if err := s.feedRepo.Upsert(ctx, failure.state); err != nil {
		return failure.result, err
	}
	return failure.result, failure.err
}

func (s *Service) confirmUnchangedEnrollments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider integration.Type,
	changed map[string]carrierintelports.CarrierIntelChangedProfile,
	now int64,
) error {
	afterID := pulid.Nil
	for {
		page, err := s.enrollmentRepo.ListActive(
			ctx,
			&carrierintelports.ListActiveEnrollmentsRequest{
				TenantInfo: tenantInfo,
				Provider:   provider,
				Mode:       carrierintel.EnrollmentModeNative,
				AfterID:    afterID,
				Limit:      confirmBatchSize,
			},
		)
		if err != nil {
			return err
		}
		if len(page) == 0 {
			return nil
		}

		bySubjectType := make(map[carrierintel.SubjectType][]string, 3)
		for _, row := range page {
			if _, wasChanged := changed[row.ProviderRef]; wasChanged {
				continue
			}
			bySubjectType[row.SubjectType] = append(bySubjectType[row.SubjectType], row.SubjectID)
		}
		for subjectType, ids := range bySubjectType {
			snaps, snapErr := s.snapshotRepo.GetCurrentBySubjectIDs(
				ctx,
				tenantInfo,
				subjectType,
				ids,
			)
			if snapErr != nil {
				return snapErr
			}
			snapIDs := make([]pulid.ID, 0, len(snaps))
			for _, snap := range snaps {
				if snap.Provider == provider {
					snapIDs = append(snapIDs, snap.ID)
				}
			}
			if err = s.snapshotRepo.TouchConfirmed(ctx, tenantInfo, snapIDs, now); err != nil {
				return err
			}
			if err = s.enrollmentRepo.TouchConfirmed(
				ctx,
				&carrierintelports.TouchEnrollmentsConfirmedRequest{
					TenantInfo:  tenantInfo,
					Provider:    provider,
					Mode:        carrierintel.EnrollmentModeNative,
					SubjectIDs:  ids,
					ConfirmedAt: now,
				},
			); err != nil {
				return err
			}
		}

		if len(page) < confirmBatchSize {
			return nil
		}
		afterID = page[len(page)-1].ID
	}
}

func (s *Service) RefreshDueSnapshots(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	limit int,
	heartbeat HeartbeatFunc,
) (*PollResult, error) {
	ctx = carrierintel.WithPurpose(ctx, carrierintel.PurposeMonitor)
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return &PollResult{Skipped: true, Reason: "no provider"}, nil
	}

	key := carrierintelports.CarrierIntelFeedStateKey{
		TenantInfo: tenantInfo,
		Provider:   provider,
		FeedType:   carrierintelenrollment.FeedTypeSnapshotRefresh,
	}
	state, err := s.feedRepo.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if state != nil && state.IsPaused() {
		return &PollResult{Paused: true, Reason: state.PausedReason.String()}, nil
	}

	if limit <= 0 {
		limit = defaultRefreshBatch
	}
	now := s.now()
	confirmedBefore := now - control.SnapshotTTLSeconds()
	due, err := s.enrollmentRepo.ListActive(ctx, &carrierintelports.ListActiveEnrollmentsRequest{
		TenantInfo:      tenantInfo,
		Provider:        provider,
		Mode:            carrierintel.EnrollmentModeSnapshotDiff,
		ConfirmedBefore: &confirmedBefore,
		Limit:           limit,
	})
	if err != nil {
		return nil, err
	}
	result := &PollResult{Changed: len(due)}
	if len(due) == 0 {
		return result, nil
	}

	subjects, err := s.resolveEnrollmentSubjects(ctx, tenantInfo, due)
	if err != nil {
		return nil, err
	}

	confirmed := make([]string, 0, len(subjects))
	for _, subject := range subjects {
		if ctx.Err() != nil {
			break
		}
		fetched, fetchErr := s.intel.Fetch(ctx, &carrierintelservice.FetchRequest{
			TenantInfo:    tenantInfo,
			Subject:       subject,
			MinDepth:      carrierintel.LookupDepthFMCSA,
			MaxAgeSeconds: control.SnapshotTTLSeconds(),
			Purpose:       carrierintel.PurposeMonitor,
			Source:        carrierintelevent.EventSourceSnapshotDiff,
			Control:       control,
		})
		if fetchErr != nil {
			if carrierintelenrollmentservice.IsPausingError(fetchErr) {
				if state == nil {
					state = carrierintelenrollmentservice.NewFeedState(key)
				}
				if pauseErr := s.enrollment.PauseFeed(
					ctx, tenantInfo, provider, carrierintelenrollment.FeedTypeSnapshotRefresh,
					carrierintelenrollmentservice.PauseReasonFor(fetchErr),
					carrierintelservice.ErrorSummary(fetchErr),
				); pauseErr != nil {
					return result, pauseErr
				}
				result.Paused = true
				break
			}
			result.Failed++
			continue
		}
		if !fetched.FromCache {
			result.Refreshed++
		}
		confirmed = append(confirmed, subject.SubjectID)
		if heartbeat != nil {
			heartbeat(result.Refreshed)
		}
	}

	if len(confirmed) > 0 {
		if err = s.enrollmentRepo.TouchConfirmed(
			ctx,
			&carrierintelports.TouchEnrollmentsConfirmedRequest{
				TenantInfo:  tenantInfo,
				Provider:    provider,
				Mode:        carrierintel.EnrollmentModeSnapshotDiff,
				SubjectIDs:  confirmed,
				ConfirmedAt: now,
			},
		); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (s *Service) resolveEnrollmentSubjects(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	enrollments []*carrierintelenrollment.CarrierMonitoringEnrollment,
) ([]carrierintelports.CarrierIntelSubject, error) {
	subjects := make([]carrierintelports.CarrierIntelSubject, 0, len(enrollments))
	carrierIDs := make([]pulid.ID, 0, len(enrollments))
	for _, row := range enrollments {
		switch row.SubjectType {
		case carrierintel.SubjectTypeCarrier:
			carrierIDs = append(carrierIDs, row.CarrierID)
		case carrierintel.SubjectTypeOrganization:
			org, err := s.subjectRepo.GetOrganizationSubject(ctx, tenantInfo)
			if err != nil {
				return nil, err
			}
			if org != nil {
				subjects = append(subjects, *org)
			}
		case carrierintel.SubjectTypeCustomer:
			customerID, err := pulid.Parse(row.SubjectID)
			if err != nil {
				continue
			}
			customer, err := s.subjectRepo.GetCustomerSubject(ctx, tenantInfo, customerID)
			if err != nil {
				continue
			}
			if customer.DOTNumber != "" {
				subjects = append(subjects, *customer)
			}
		case carrierintel.SubjectTypeProspect:
			subjects = append(subjects, carrierintelports.CarrierIntelSubject{
				SubjectType:  row.SubjectType,
				SubjectID:    row.SubjectID,
				Name:         row.SubjectName,
				DOTNumber:    row.DOTNumber,
				DocketNumber: row.DocketNumber,
			})
		}
	}
	if len(carrierIDs) > 0 {
		carriers, err := s.subjectRepo.ListCarrierSubjects(
			ctx,
			&carrierintelports.ListCarrierIntelSubjectsRequest{
				TenantInfo: tenantInfo,
				CarrierIDs: carrierIDs,
				Limit:      len(carrierIDs),
			},
		)
		if err != nil {
			return nil, err
		}
		subjects = append(subjects, carriers...)
	}
	return subjects, nil
}
