package carrierintelservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	changeFeedPageSize      = 500
	maxChangeFeedPages      = 40
	defaultRefreshBatch     = 200
	recomputeBatchSize      = 500
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
	connector, ok := s.connectors[provider]
	if !ok || !connector.Descriptor().Capabilities.Has(carrierintel.CapabilityNativeMonitoring) {
		return &PollResult{Skipped: true, Reason: "provider has no change feed"}, nil
	}

	key := repositories.CarrierIntelFeedStateKey{
		TenantInfo: tenantInfo,
		Provider:   provider,
		FeedType:   carrierintel.FeedTypeChangeFeed,
	}
	state, err := s.feedRepo.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if state == nil {
		state = newFeedState(key)
	}
	now := s.now()
	if state.IsPaused() {
		return &PollResult{Paused: true, Reason: state.PausedReason.String()}, nil
	}
	if !state.DueAt(now) {
		return &PollResult{Skipped: true, Reason: "not due"}, nil
	}

	bound, err := s.bind(ctx, tenantInfo, provider, false)
	if err != nil {
		return nil, err
	}
	monitor, ok := bound.client.(services.CarrierIntelNativeMonitor)
	if !ok {
		return &PollResult{Skipped: true, Reason: "provider has no change feed"}, nil
	}

	since := now - 2*timeutils.SecondsPerDay
	if state.LastSuccessAt != nil {
		since = timeutils.DayStartUTC(*state.LastSuccessAt) - timeutils.SecondsPerDay
	}

	result := &PollResult{}
	changedRefs := make(map[string]services.CarrierIntelChangedProfile, 64)
	for page := 1; page <= maxChangeFeedPages; page++ {
		var feed *services.CarrierIntelChangeFeedPage
		listErr := s.withRateLimitWait(ctx, func() error {
			var callErr error
			feed, callErr = monitor.ListChanges(ctx, &services.CarrierIntelChangeFeedRequest{
				Since:    since,
				Until:    now,
				Page:     page,
				PageSize: changeFeedPageSize,
			})
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
			_, fetchErr := s.Fetch(ctx, &FetchRequest{
				TenantInfo: tenantInfo,
				Subject:    subject,
				MinDepth:   carrierintel.LookupDepthFMCSA,
				Purpose:    carrierintel.PurposeMonitor,
				Force:      true,
				Source:     carrierintel.EventSourceNativeChangeFeed,
				Control:    control,
			})
			if fetchErr != nil {
				if isPausingError(fetchErr) {
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

	if err = s.confirmUnchangedEnrollments(ctx, tenantInfo, provider, changedRefs, now); err != nil {
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
	state    *carrierintel.CarrierIntelFeedState
	control  *carrierintel.CarrierIntelControl
	err      error
	result   *PollResult
}

func (s *Service) recordFeedFailure(
	ctx context.Context,
	failure *feedFailure,
) (*PollResult, error) {
	now := s.now()
	if reason := pauseReasonFor(failure.err); reason != carrierintel.FeedPauseReasonNone {
		if err := s.pauseFeed(
			ctx, failure.tenant, failure.provider, carrierintel.FeedTypeChangeFeed, reason,
			errorSummary(failure.err),
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
	failure.state.RecordFailure(errorSummary(failure.err), now, now+max(backoff, 300))
	if err := s.feedRepo.Upsert(ctx, failure.state); err != nil {
		return failure.result, err
	}
	return failure.result, failure.err
}

func (s *Service) confirmUnchangedEnrollments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider integration.Type,
	changed map[string]services.CarrierIntelChangedProfile,
	now int64,
) error {
	afterID := pulid.Nil
	for {
		page, err := s.enrollmentRepo.ListActive(ctx, &repositories.ListActiveEnrollmentsRequest{
			TenantInfo: tenantInfo,
			Provider:   provider,
			Mode:       carrierintel.EnrollmentModeNative,
			AfterID:    afterID,
			Limit:      recomputeBatchSize,
		})
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
			if err = s.enrollmentRepo.TouchConfirmed(ctx, &repositories.TouchEnrollmentsConfirmedRequest{
				TenantInfo:  tenantInfo,
				Provider:    provider,
				Mode:        carrierintel.EnrollmentModeNative,
				SubjectIDs:  ids,
				ConfirmedAt: now,
			}); err != nil {
				return err
			}
		}

		if len(page) < recomputeBatchSize {
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

	key := repositories.CarrierIntelFeedStateKey{
		TenantInfo: tenantInfo,
		Provider:   provider,
		FeedType:   carrierintel.FeedTypeSnapshotRefresh,
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
	due, err := s.enrollmentRepo.ListActive(ctx, &repositories.ListActiveEnrollmentsRequest{
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
		fetched, fetchErr := s.Fetch(ctx, &FetchRequest{
			TenantInfo:    tenantInfo,
			Subject:       subject,
			MinDepth:      carrierintel.LookupDepthFMCSA,
			MaxAgeSeconds: control.SnapshotTTLSeconds(),
			Purpose:       carrierintel.PurposeMonitor,
			Source:        carrierintel.EventSourceSnapshotDiff,
			Control:       control,
		})
		if fetchErr != nil {
			if isPausingError(fetchErr) {
				if state == nil {
					state = newFeedState(key)
				}
				if pauseErr := s.pauseFeed(
					ctx, tenantInfo, provider, carrierintel.FeedTypeSnapshotRefresh,
					pauseReasonFor(fetchErr), errorSummary(fetchErr),
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
		if err = s.enrollmentRepo.TouchConfirmed(ctx, &repositories.TouchEnrollmentsConfirmedRequest{
			TenantInfo:  tenantInfo,
			Provider:    provider,
			Mode:        carrierintel.EnrollmentModeSnapshotDiff,
			SubjectIDs:  confirmed,
			ConfirmedAt: now,
		}); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (s *Service) resolveEnrollmentSubjects(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	enrollments []*carrierintel.CarrierMonitoringEnrollment,
) ([]repositories.CarrierIntelSubject, error) {
	subjects := make([]repositories.CarrierIntelSubject, 0, len(enrollments))
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
			subjects = append(subjects, repositories.CarrierIntelSubject{
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
			&repositories.ListCarrierIntelSubjectsRequest{
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

type RecomputeResult struct {
	Evaluated int `json:"evaluated"`
	Changed   int `json:"changed"`
}

func (s *Service) RecomputeStalePolicy(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	deadline time.Time,
) (*RecomputeResult, error) {
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return &RecomputeResult{}, nil
	}

	result := &RecomputeResult{}
	afterID := pulid.Nil
	for time.Now().Before(deadline) {
		page, listErr := s.snapshotRepo.ListForRecompute(
			ctx,
			&repositories.ListSnapshotsForRecomputeRequest{
				TenantInfo:   tenantInfo,
				BelowVersion: control.PolicyVersion,
				AfterID:      afterID,
				Limit:        recomputeBatchSize,
			},
		)
		if listErr != nil {
			return result, listErr
		}
		if len(page) == 0 {
			return result, nil
		}

		for _, snap := range page {
			afterID = snap.ID
			subject, subjectErr := s.subjectForSnapshot(ctx, tenantInfo, snap)
			if subjectErr != nil {
				s.l.Warn("failed to resolve subject for recompute", zap.Error(subjectErr))
				continue
			}
			ingested, ingestErr := s.confirmUnchanged(ctx, &ingestInput{
				tenant:  tenantInfo,
				control: control,
				subject: subject,
				bound:   &boundProvider{provider: provider, tenant: tenantInfo},
				result:  &services.CarrierIntelLookupResult{},
				current: snap,
				source:  carrierintel.EventSourceRuleEvaluation,
				purpose: carrierintel.PurposeReconcile,
			}, snap, s.recomputeFindings(ctx, tenantInfo, control, subject, snap), snap.EffectiveAsOf())
			if ingestErr != nil {
				s.l.Warn("failed to recompute snapshot findings", zap.Error(ingestErr))
				continue
			}
			result.Evaluated++
			if len(ingested.Raised) > 0 {
				result.Changed++
			}
		}
		if len(page) < recomputeBatchSize {
			return result, nil
		}
	}
	return result, nil
}

func (s *Service) recomputeFindings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	control *carrierintel.CarrierIntelControl,
	subject repositories.CarrierIntelSubject,
	snap *carrierintel.CarrierIntelSnapshot,
) []carrierintel.Finding {
	findings, err := s.evaluateSubject(
		ctx,
		tenantInfo,
		control,
		subject,
		snap.Profile,
		snap.NotFound,
	)
	if err != nil {
		return snap.Findings
	}
	return carrierintel.CarryUnconfirmed(snap.Findings, findings)
}

func (s *Service) subjectForSnapshot(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	snap *carrierintel.CarrierIntelSnapshot,
) (repositories.CarrierIntelSubject, error) {
	switch snap.SubjectType {
	case carrierintel.SubjectTypeCarrier:
		subjects, err := s.subjectRepo.ListCarrierSubjects(
			ctx,
			&repositories.ListCarrierIntelSubjectsRequest{
				TenantInfo: tenantInfo,
				CarrierIDs: []pulid.ID{snap.CarrierID},
				Limit:      1,
			},
		)
		if err != nil {
			return repositories.CarrierIntelSubject{}, err
		}
		if len(subjects) == 1 {
			return subjects[0], nil
		}
	case carrierintel.SubjectTypeOrganization:
		org, err := s.subjectRepo.GetOrganizationSubject(ctx, tenantInfo)
		if err != nil {
			return repositories.CarrierIntelSubject{}, err
		}
		if org != nil {
			return *org, nil
		}
	case carrierintel.SubjectTypeCustomer:
		customerID, err := pulid.Parse(snap.SubjectID)
		if err == nil {
			customer, customerErr := s.subjectRepo.GetCustomerSubject(ctx, tenantInfo, customerID)
			if customerErr == nil {
				return *customer, nil
			}
		}
	}
	return repositories.CarrierIntelSubject{
		SubjectType:  snap.SubjectType,
		SubjectID:    snap.SubjectID,
		CarrierID:    snap.CarrierID,
		DOTNumber:    snap.DOTNumber,
		DocketNumber: snap.DocketNumber,
	}, nil
}
