package carrierintelservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	enrollmentBatchSize     = 100
	subjectPageSize         = 1000
	watchlistPageSize       = 500
	maxEnrollmentSyncPerRun = 2000
)

func policyReason(policy carrierintel.EnrollmentPolicy) carrierintel.EnrollmentReason {
	switch policy {
	case carrierintel.EnrollmentPolicyAllActive:
		return carrierintel.EnrollmentReasonPolicyAllActive
	case carrierintel.EnrollmentPolicyRecentlyUsed:
		return carrierintel.EnrollmentReasonPolicyRecentUse
	default:
		return carrierintel.EnrollmentReasonManual
	}
}

func isPolicyReason(reason carrierintel.EnrollmentReason) bool {
	switch reason {
	case carrierintel.EnrollmentReasonPolicyAllActive,
		carrierintel.EnrollmentReasonPolicyRecentUse,
		carrierintel.EnrollmentReasonAssignedOrTendered,
		carrierintel.EnrollmentReasonSelfMonitor,
		carrierintel.EnrollmentReasonCustomerBroker:
		return true
	default:
		return false
	}
}

func (s *Service) CarrierSaved(ctx context.Context, event *services.CarrierLifecycleEvent) {
	if event == nil || event.Carrier == nil || event.Carrier.DOTNumber == "" {
		return
	}
	ctx = context.WithoutCancel(ctx)
	control, err := s.controlRepo.GetOrCreate(ctx, event.TenantInfo)
	if err != nil {
		s.l.Warn("failed to load carrier intelligence control", zap.Error(err))
		return
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return
	}

	entity := event.Carrier
	subject := carrierSubject(entity)
	now := s.now()

	switch {
	case !entity.IsActive() && control.AutoUnenrollOnInactive:
		err = s.setDesired(ctx, &desireRequest{
			tenant:       event.TenantInfo,
			control:      control,
			provider:     provider,
			subjects:     []repositories.CarrierIntelSubject{subject},
			state:        carrierintel.DesiredStateNotEnrolled,
			now:          now,
			onlyIfPolicy: true,
		})
	case entity.IsActive() && control.AutoEnrollOnCreate &&
		control.EnrollmentPolicy == carrierintel.EnrollmentPolicyAllActive &&
		(event.Created || event.PreviousStatus != carrier.StatusActive):
		err = s.setDesired(ctx, &desireRequest{
			tenant:   event.TenantInfo,
			control:  control,
			provider: provider,
			subjects: []repositories.CarrierIntelSubject{subject},
			state:    carrierintel.DesiredStateEnrolled,
			reason:   carrierintel.EnrollmentReasonPolicyAllActive,
			now:      now,
		})
	}
	if err != nil {
		s.l.Warn("failed to update carrier monitoring enrollment", zap.Error(err),
			zap.String("carrierId", entity.ID.String()))
	}
}

func (s *Service) CarriersUsed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierIDs []pulid.ID,
) {
	if len(carrierIDs) == 0 {
		return
	}
	ctx = context.WithoutCancel(ctx)
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		s.l.Warn("failed to load carrier intelligence control", zap.Error(err))
		return
	}
	provider, ok := control.PrimaryType()
	if !ok || control.EnrollmentPolicy == carrierintel.EnrollmentPolicyManual {
		return
	}

	subjects, err := s.subjectRepo.ListCarrierSubjects(
		ctx,
		&repositories.ListCarrierIntelSubjectsRequest{
			TenantInfo: tenantInfo,
			CarrierIDs: carrierIDs,
			ActiveOnly: true,
			Limit:      len(carrierIDs),
		},
	)
	if err != nil {
		s.l.Warn("failed to load used carrier subjects", zap.Error(err))
		return
	}
	now := s.now()
	for idx := range subjects {
		subjects[idx].LastUsedAt = &now
	}

	if err = s.setDesired(ctx, &desireRequest{
		tenant:   tenantInfo,
		control:  control,
		provider: provider,
		subjects: subjects,
		state:    carrierintel.DesiredStateEnrolled,
		reason:   carrierintel.EnrollmentReasonAssignedOrTendered,
		now:      now,
	}); err != nil {
		s.l.Warn("failed to enroll used carriers", zap.Error(err))
	}
}

func carrierSubject(entity *carrier.Carrier) repositories.CarrierIntelSubject {
	return repositories.CarrierIntelSubject{
		SubjectType:  carrierintel.SubjectTypeCarrier,
		SubjectID:    entity.ID.String(),
		CarrierID:    entity.ID,
		Name:         entity.Name,
		DOTNumber:    entity.DOTNumber,
		DocketNumber: entity.MCNumber,
		Broker:       entity.CarrierType == carrier.TypeBroker,
		Exempt:       entity.CarrierType == carrier.TypeExempt,
	}
}

type desireRequest struct {
	tenant       pagination.TenantInfo
	control      *carrierintel.CarrierIntelControl
	provider     integration.Type
	subjects     []repositories.CarrierIntelSubject
	state        carrierintel.DesiredState
	reason       carrierintel.EnrollmentReason
	now          int64
	onlyIfPolicy bool
}

func (s *Service) setDesired(ctx context.Context, req *desireRequest) error {
	if len(req.subjects) == 0 {
		return nil
	}
	connector, ok := s.connectors[req.provider]
	if !ok {
		return nil
	}
	mode := connector.Descriptor().Capabilities.MonitoringMode()

	bySubjectType := make(map[carrierintel.SubjectType][]string, 2)
	for _, subject := range req.subjects {
		bySubjectType[subject.SubjectType] = append(
			bySubjectType[subject.SubjectType],
			subject.SubjectID,
		)
	}

	existing := make(map[string]*carrierintel.CarrierMonitoringEnrollment, len(req.subjects))
	for subjectType, ids := range bySubjectType {
		rows, err := s.enrollmentRepo.ListBySubjects(
			ctx,
			&repositories.ListEnrollmentsBySubjectRequest{
				TenantInfo:  req.tenant,
				Provider:    req.provider,
				SubjectType: subjectType,
				SubjectIDs:  ids,
			},
		)
		if err != nil {
			return err
		}
		for _, row := range rows {
			existing[carrierintel.SubjectKey(row.SubjectType, row.SubjectID)] = row
		}
	}

	changed := make([]*carrierintel.CarrierMonitoringEnrollment, 0, len(req.subjects))
	for _, subject := range req.subjects {
		if subject.DOTNumber == "" {
			continue
		}
		key := carrierintel.SubjectKey(subject.SubjectType, subject.SubjectID)
		row, found := existing[key]
		if !found {
			if req.state == carrierintel.DesiredStateNotEnrolled {
				continue
			}
			row = &carrierintel.CarrierMonitoringEnrollment{
				OrganizationID: req.tenant.OrgID,
				BusinessUnitID: req.tenant.BuID,
				SubjectType:    subject.SubjectType,
				SubjectID:      subject.SubjectID,
				Provider:       req.provider,
				Mode:           mode,
				DesiredState:   carrierintel.DesiredStateNotEnrolled,
				VendorState:    carrierintel.VendorStateUnknown,
				Reason:         req.reason,
			}
		}
		if req.onlyIfPolicy && found && !isPolicyReason(row.Reason) &&
			req.state == carrierintel.DesiredStateNotEnrolled {
			continue
		}

		dirty := row.MarkDesired(req.state, req.reason, req.now)
		if row.DOTNumber != subject.DOTNumber || row.DocketNumber != subject.DocketNumber ||
			row.SubjectName != subject.Name || row.Mode != mode {
			row.DOTNumber = subject.DOTNumber
			row.DocketNumber = subject.DocketNumber
			row.SubjectName = subject.Name
			row.Mode = mode
			dirty = true
		}
		if subject.CarrierID.IsNotNil() && row.CarrierID != subject.CarrierID {
			row.CarrierID = subject.CarrierID
			dirty = true
		}
		if subject.LastUsedAt != nil {
			row.LastUsedAt = subject.LastUsedAt
			dirty = true
		}
		if dirty || !found {
			changed = append(changed, row)
		}
	}

	if len(changed) == 0 {
		return nil
	}
	return s.enrollmentRepo.Upsert(ctx, changed)
}

type EnrollSubjectsRequest struct {
	TenantInfo pagination.TenantInfo
	CarrierIDs []pulid.ID
	Enroll     bool
}

func (s *Service) SetManualEnrollment(
	ctx context.Context,
	req *EnrollSubjectsRequest,
) (int, error) {
	if len(req.CarrierIDs) == 0 {
		return 0, errortypes.NewValidationError("carrierIds", errortypes.ErrRequired,
			"Select at least one carrier")
	}
	control, err := s.controlRepo.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return 0, err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return 0, errortypes.NewBusinessError(
			"Enable a carrier intelligence provider before monitoring carriers",
		)
	}
	connector, ok := s.connectors[provider]
	if !ok || !connector.Descriptor().Capabilities.SupportsMonitoring() {
		return 0, errortypes.NewBusinessError(
			"{0} does not support carrier monitoring", provider.String(),
		)
	}

	subjects, err := s.subjectRepo.ListCarrierSubjects(
		ctx,
		&repositories.ListCarrierIntelSubjectsRequest{
			TenantInfo: req.TenantInfo,
			CarrierIDs: req.CarrierIDs,
			Limit:      len(req.CarrierIDs),
		},
	)
	if err != nil {
		return 0, err
	}
	if len(subjects) == 0 {
		return 0, errortypes.NewBusinessError(
			"None of the selected carriers have a DOT number to monitor",
		)
	}

	state := carrierintel.DesiredStateNotEnrolled
	if req.Enroll {
		state = carrierintel.DesiredStateEnrolled
	}
	if err = s.setDesired(ctx, &desireRequest{
		tenant:   req.TenantInfo,
		control:  control,
		provider: provider,
		subjects: subjects,
		state:    state,
		reason:   carrierintel.EnrollmentReasonManual,
		now:      s.now(),
	}); err != nil {
		return 0, err
	}
	s.publish(ctx, req.TenantInfo, "carrier_monitoring_enrollments", "updated", pulid.Nil)
	return len(subjects), nil
}

func (s *Service) desiredSubjects(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	control *carrierintel.CarrierIntelControl,
) ([]repositories.CarrierIntelSubject, error) {
	subjects := make([]repositories.CarrierIntelSubject, 0, 128)

	if control.EnrollmentPolicy != carrierintel.EnrollmentPolicyManual {
		req := &repositories.ListCarrierIntelSubjectsRequest{
			TenantInfo: tenantInfo,
			ActiveOnly: true,
			Limit:      subjectPageSize,
		}
		if control.EnrollmentPolicy == carrierintel.EnrollmentPolicyRecentlyUsed {
			since := s.now() - int64(control.RecentUsageDays)*timeutils.SecondsPerDay
			req.UsedSince = &since
			req.IncludeOpenTenders = control.IncludeOpenTenders
		}
		for {
			page, err := s.subjectRepo.ListCarrierSubjects(ctx, req)
			if err != nil {
				return nil, err
			}
			subjects = append(subjects, page...)
			if len(page) < req.Limit {
				break
			}
			req.AfterID = page[len(page)-1].CarrierID
		}
	}

	if control.SelfMonitoringEnabled {
		org, err := s.subjectRepo.GetOrganizationSubject(ctx, tenantInfo)
		if err != nil {
			return nil, err
		}
		if org != nil && org.DOTNumber != "" {
			subjects = append(subjects, *org)
		}
	}

	brokers, err := s.subjectRepo.ListBrokerCustomerSubjects(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	return append(subjects, brokers...), nil
}

func reasonForSubject(
	control *carrierintel.CarrierIntelControl,
	subject repositories.CarrierIntelSubject,
) carrierintel.EnrollmentReason {
	switch subject.SubjectType {
	case carrierintel.SubjectTypeOrganization:
		return carrierintel.EnrollmentReasonSelfMonitor
	case carrierintel.SubjectTypeCustomer:
		return carrierintel.EnrollmentReasonCustomerBroker
	default:
		return policyReason(control.EnrollmentPolicy)
	}
}

type ReconcileResult struct {
	Desired     int `json:"desired"`
	Enrolled    int `json:"enrolled"`
	Unenrolled  int `json:"unenrolled"`
	DriftAdded  int `json:"driftAdded"`
	DriftRemove int `json:"driftRemoved"`
}

func (s *Service) ReconcileDesired(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*ReconcileResult, error) {
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return &ReconcileResult{}, nil
	}

	subjects, err := s.desiredSubjects(ctx, tenantInfo, control)
	if err != nil {
		return nil, err
	}
	now := s.now()
	result := &ReconcileResult{Desired: len(subjects)}

	byReason := make(map[carrierintel.EnrollmentReason][]repositories.CarrierIntelSubject, 3)
	want := make(map[string]struct{}, len(subjects))
	for _, subject := range subjects {
		reason := reasonForSubject(control, subject)
		byReason[reason] = append(byReason[reason], subject)
		want[carrierintel.SubjectKey(subject.SubjectType, subject.SubjectID)] = struct{}{}
	}
	for reason, group := range byReason {
		if err = s.setDesired(ctx, &desireRequest{
			tenant:   tenantInfo,
			control:  control,
			provider: provider,
			subjects: group,
			state:    carrierintel.DesiredStateEnrolled,
			reason:   reason,
			now:      now,
		}); err != nil {
			return nil, err
		}
	}

	current, err := s.enrollmentRepo.ListDesired(ctx, tenantInfo, provider)
	if err != nil {
		return nil, err
	}
	stale := make([]*carrierintel.CarrierMonitoringEnrollment, 0)
	for _, row := range current {
		if _, keep := want[carrierintel.SubjectKey(row.SubjectType, row.SubjectID)]; keep {
			continue
		}
		if !isPolicyReason(row.Reason) {
			continue
		}
		if row.MarkDesired(carrierintel.DesiredStateNotEnrolled, "", now) {
			stale = append(stale, row)
		}
	}
	if len(stale) > 0 {
		if err = s.enrollmentRepo.Upsert(ctx, stale); err != nil {
			return nil, err
		}
	}
	result.Unenrolled = len(stale)
	return result, nil
}

type SyncResult struct {
	Enrolled   int    `json:"enrolled"`
	Removed    int    `json:"removed"`
	Failed     int    `json:"failed"`
	Paused     bool   `json:"paused"`
	PauseCause string `json:"pauseCause,omitempty"`
}

func (s *Service) SyncPendingEnrollments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*SyncResult, error) {
	ctx = carrierintel.WithPurpose(ctx, carrierintel.PurposeReconcile)
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return &SyncResult{}, nil
	}

	pending, err := s.enrollmentRepo.ListNeedingSync(
		ctx,
		&repositories.ListEnrollmentsNeedingSyncRequest{
			TenantInfo: tenantInfo,
			Provider:   provider,
			Limit:      maxEnrollmentSyncPerRun,
		},
	)
	if err != nil || len(pending) == 0 {
		return &SyncResult{}, err
	}

	now := s.now()
	connector, ok := s.connectors[provider]
	if !ok {
		return &SyncResult{}, nil
	}
	if !connector.Descriptor().Capabilities.Has(carrierintel.CapabilityNativeMonitoring) {
		for _, row := range pending {
			row.MarkSynced(now)
		}
		return &SyncResult{Enrolled: len(pending)}, s.enrollmentRepo.SaveSyncState(ctx, pending)
	}

	bound, err := s.bind(ctx, tenantInfo, provider, false)
	if err != nil {
		return nil, err
	}
	monitor, ok := bound.client.(services.CarrierIntelNativeMonitor)
	if !ok {
		return &SyncResult{}, nil
	}

	if err = s.fillProviderRefs(ctx, tenantInfo, monitor, pending); err != nil {
		return nil, err
	}

	adds := make([]*carrierintel.CarrierMonitoringEnrollment, 0, len(pending))
	removes := make([]*carrierintel.CarrierMonitoringEnrollment, 0, len(pending))
	for _, row := range pending {
		if row.WantsEnrolled() {
			adds = append(adds, row)
		} else {
			removes = append(removes, row)
		}
	}

	result := &SyncResult{}
	if pauseErr := s.syncBatches(ctx, &syncBatch{
		tenant:  tenantInfo,
		control: control,
		bound:   bound,
		rows:    adds,
		add:     true,
		call:    monitor.Enroll,
		result:  result,
	}); pauseErr != nil {
		return s.pauseFromError(
			ctx,
			tenantInfo,
			provider,
			carrierintel.FeedTypeChangeFeed,
			pauseErr,
			result,
		)
	}
	if pauseErr := s.syncBatches(ctx, &syncBatch{
		tenant:  tenantInfo,
		control: control,
		bound:   bound,
		rows:    removes,
		call:    monitor.Unenroll,
		result:  result,
	}); pauseErr != nil {
		return s.pauseFromError(
			ctx,
			tenantInfo,
			provider,
			carrierintel.FeedTypeChangeFeed,
			pauseErr,
			result,
		)
	}

	if result.Enrolled+result.Removed > 0 {
		s.publish(ctx, tenantInfo, "carrier_monitoring_enrollments", "updated", pulid.Nil)
	}
	return result, nil
}

func (s *Service) fillProviderRefs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	monitor services.CarrierIntelNativeMonitor,
	rows []*carrierintel.CarrierMonitoringEnrollment,
) error {
	missing := make(map[carrierintel.SubjectType][]string, 2)
	for _, row := range rows {
		if row.ProviderRef == "" {
			missing[row.SubjectType] = append(missing[row.SubjectType], row.SubjectID)
		}
	}
	profiles := make(map[string]*carrierintel.Profile, len(rows))
	for subjectType, ids := range missing {
		snaps, err := s.snapshotRepo.GetCurrentBySubjectIDs(ctx, tenantInfo, subjectType, ids)
		if err != nil {
			return err
		}
		for _, snap := range snaps {
			profiles[carrierintel.SubjectKey(snap.SubjectType, snap.SubjectID)] = snap.Profile
		}
	}
	for _, row := range rows {
		if row.ProviderRef != "" {
			continue
		}
		row.ProviderRef = monitor.MonitoringRef(
			profiles[carrierintel.SubjectKey(row.SubjectType, row.SubjectID)],
			row.DOTNumber,
			row.DocketNumber,
		)
	}
	return nil
}

type syncBatch struct {
	tenant  pagination.TenantInfo
	control *carrierintel.CarrierIntelControl
	bound   *boundProvider
	rows    []*carrierintel.CarrierMonitoringEnrollment
	add     bool
	call    func(ctx context.Context, refs []string) (*services.CarrierIntelEnrollResult, error)
	result  *SyncResult
}

func (s *Service) syncBatches(ctx context.Context, batch *syncBatch) error {
	for start := 0; start < len(batch.rows); start += enrollmentBatchSize {
		end := min(start+enrollmentBatchSize, len(batch.rows))
		chunk := batch.rows[start:end]
		refs := make([]string, 0, len(chunk))
		byRef := make(map[string]*carrierintel.CarrierMonitoringEnrollment, len(chunk))
		for _, row := range chunk {
			refs = append(refs, row.ProviderRef)
			byRef[row.ProviderRef] = row
		}

		if batch.add {
			for _, row := range chunk {
				if err := s.guardBudget(ctx, &budgetCheck{
					tenant:    batch.tenant,
					control:   batch.control,
					bound:     batch.bound,
					endpoint:  carrierintel.EndpointMonitorAdd,
					dotNumber: row.DOTNumber,
				}); err != nil {
					return err
				}
			}
		}

		outcome, err := batch.call(ctx, refs)
		now := s.now()
		if err != nil {
			if isPausingError(err) {
				return err
			}
			for _, row := range chunk {
				row.MarkFailed(errorSummary(err), now)
			}
			batch.result.Failed += len(chunk)
			if saveErr := s.enrollmentRepo.SaveSyncState(ctx, chunk); saveErr != nil {
				return saveErr
			}
			continue
		}

		failed := make(map[string]string, len(outcome.Failed))
		for _, f := range outcome.Failed {
			failed[f.ProviderRef] = f.Message
		}
		for ref, row := range byRef {
			if message, bad := failed[ref]; bad {
				row.MarkFailed(message, now)
				batch.result.Failed++
				continue
			}
			row.MarkSynced(now)
			if batch.add {
				batch.result.Enrolled++
			} else {
				batch.result.Removed++
			}
		}
		if err = s.enrollmentRepo.SaveSyncState(ctx, chunk); err != nil {
			return err
		}
	}
	return nil
}

func isPausingError(err error) bool {
	if isSpendCapError(err) {
		return true
	}
	kind, ok := services.CarrierIntelErrorKindOf(err)
	return ok && (kind == services.CarrierIntelErrorUnauthorized ||
		kind == services.CarrierIntelErrorPaymentRequired)
}

func pauseReasonFor(err error) carrierintel.FeedPauseReason {
	if isSpendCapError(err) {
		return carrierintel.FeedPauseReasonSpendCap
	}
	kind, _ := services.CarrierIntelErrorKindOf(err)
	switch kind {
	case services.CarrierIntelErrorPaymentRequired:
		return carrierintel.FeedPauseReasonPaymentRequired
	case services.CarrierIntelErrorUnauthorized:
		return carrierintel.FeedPauseReasonUnauthorized
	default:
		return carrierintel.FeedPauseReasonNone
	}
}

func errorSummary(err error) string {
	var providerErr *services.CarrierIntelProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Error()
	}
	var business *errortypes.BusinessError
	if errors.As(err, &business) {
		return business.Error()
	}
	return "provider request failed"
}

func (s *Service) pauseFromError(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider integration.Type,
	feedType carrierintel.FeedType,
	cause error,
	result *SyncResult,
) (*SyncResult, error) {
	reason := pauseReasonFor(cause)
	if reason == carrierintel.FeedPauseReasonNone {
		return result, cause
	}
	if err := s.pauseFeed(ctx, tenantInfo, provider, feedType, reason, errorSummary(cause)); err != nil {
		return result, err
	}
	result.Paused = true
	result.PauseCause = reason.String()
	return result, nil
}

func (s *Service) pauseFeed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider integration.Type,
	feedType carrierintel.FeedType,
	reason carrierintel.FeedPauseReason,
	message string,
) error {
	key := repositories.CarrierIntelFeedStateKey{
		TenantInfo: tenantInfo,
		Provider:   provider,
		FeedType:   feedType,
	}
	state, err := s.feedRepo.Get(ctx, key)
	if err != nil {
		return err
	}
	if state == nil {
		state = newFeedState(key)
	}
	alreadyPaused := state.PausedReason == reason
	state.Pause(reason, message, s.now())
	if err = s.feedRepo.Upsert(ctx, state); err != nil {
		return err
	}
	if alreadyPaused || reason == carrierintel.FeedPauseReasonSpendCap {
		return nil
	}

	s.sendNotification(ctx, &notificationRequest{
		tenant:    tenantInfo,
		eventType: EventCarrierIntelProviderDown,
		correlation: fmt.Sprintf(
			"ci-paused-%s-%s-%s",
			provider,
			reason,
			timeutils.FormatDateKeyUTC(s.now()),
		),
		priority: notification.PriorityCritical,
		title:    "Carrier monitoring paused",
		message: fmt.Sprintf(
			"%s rejected monitoring requests (%s). Carriers are not being watched until the integration is fixed in Integrations.",
			provider,
			reason,
		),
		link: "/admin/integrations?type=" + provider.String(),
	})
	return nil
}

func newFeedState(key repositories.CarrierIntelFeedStateKey) *carrierintel.CarrierIntelFeedState {
	return &carrierintel.CarrierIntelFeedState{
		OrganizationID: key.TenantInfo.OrgID,
		BusinessUnitID: key.TenantInfo.BuID,
		Provider:       key.Provider,
		FeedType:       key.FeedType,
	}
}

func (s *Service) ResumeFeeds(ctx context.Context, tenantInfo pagination.TenantInfo) error {
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return nil
	}
	for _, feedType := range []carrierintel.FeedType{
		carrierintel.FeedTypeChangeFeed, carrierintel.FeedTypeSnapshotRefresh,
	} {
		key := repositories.CarrierIntelFeedStateKey{
			TenantInfo: tenantInfo,
			Provider:   provider,
			FeedType:   feedType,
		}
		state, getErr := s.feedRepo.Get(ctx, key)
		if getErr != nil {
			return getErr
		}
		if state == nil || !state.IsPaused() {
			continue
		}
		state.Resume()
		if err = s.feedRepo.Upsert(ctx, state); err != nil {
			return err
		}
	}
	return nil
}

type CostEstimate struct {
	Provider          integration.Type              `json:"provider"`
	Policy            carrierintel.EnrollmentPolicy `json:"policy"`
	SubjectCount      int                           `json:"subjectCount"`
	MonthlyMonitoring decimal.Decimal               `json:"monthlyMonitoring"`
	PerSubject        decimal.Decimal               `json:"perSubject"`
}

func (s *Service) EstimateEnrollmentCost(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	policy carrierintel.EnrollmentPolicy,
	recentUsageDays int,
	includeOpenTenders bool,
) (*CostEstimate, error) {
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return nil, errortypes.NewBusinessError("Enable a carrier intelligence provider first")
	}
	connector, ok := s.connectors[provider]
	if !ok {
		return nil, errortypes.NewBusinessError("{0} is not available", provider.String())
	}
	if !policy.IsValid() {
		return nil, errortypes.NewValidationError("policy", errortypes.ErrInvalid,
			"Enrollment policy is invalid")
	}

	preview := *control
	preview.EnrollmentPolicy = policy
	if recentUsageDays > 0 {
		preview.RecentUsageDays = recentUsageDays
	}
	preview.IncludeOpenTenders = includeOpenTenders

	subjects, err := s.desiredSubjects(ctx, tenantInfo, &preview)
	if err != nil {
		return nil, err
	}
	perSubject := connector.PriceBook().Price(carrierintel.EndpointMonitorAdd).UnitCost
	return &CostEstimate{
		Provider:          provider,
		Policy:            policy,
		SubjectCount:      len(subjects),
		MonthlyMonitoring: connector.PriceBook().MonitoringMonthlyCost(len(subjects)),
		PerSubject:        perSubject,
	}, nil
}

func (s *Service) DriftCheck(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*ReconcileResult, error) {
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return &ReconcileResult{}, nil
	}
	connector, ok := s.connectors[provider]
	if !ok || !connector.Descriptor().Capabilities.Has(carrierintel.CapabilityNativeMonitoring) {
		return &ReconcileResult{}, nil
	}
	bound, err := s.bind(ctx, tenantInfo, provider, false)
	if err != nil {
		return nil, err
	}
	monitor, ok := bound.client.(services.CarrierIntelNativeMonitor)
	if !ok {
		return &ReconcileResult{}, nil
	}

	vendorRefs := make(map[string]struct{}, 256)
	for page := 1; ; page++ {
		result, listErr := monitor.ListWatchlist(ctx, page, watchlistPageSize)
		if listErr != nil {
			return nil, listErr
		}
		for _, ref := range result.ProviderRefs {
			vendorRefs[ref] = struct{}{}
		}
		if !result.HasMore || len(result.ProviderRefs) == 0 {
			break
		}
	}

	desired, err := s.enrollmentRepo.ListDesired(ctx, tenantInfo, provider)
	if err != nil {
		return nil, err
	}
	ours := make(map[string]struct{}, len(desired))
	resync := make([]*carrierintel.CarrierMonitoringEnrollment, 0)
	for _, row := range desired {
		if row.ProviderRef == "" {
			continue
		}
		ours[row.ProviderRef] = struct{}{}
		if _, present := vendorRefs[row.ProviderRef]; !present &&
			row.VendorState == carrierintel.VendorStateActive {
			row.VendorState = carrierintel.VendorStatePendingAdd
			resync = append(resync, row)
		}
	}
	result := &ReconcileResult{Desired: len(desired), DriftAdded: len(resync)}
	if len(resync) > 0 {
		if err = s.enrollmentRepo.SaveSyncState(ctx, resync); err != nil {
			return nil, err
		}
	}

	if !control.ExclusiveWatchlist {
		return result, nil
	}
	extra := make([]string, 0)
	for ref := range vendorRefs {
		if _, mine := ours[ref]; !mine {
			extra = append(extra, ref)
		}
	}
	for start := 0; start < len(extra); start += enrollmentBatchSize {
		end := min(start+enrollmentBatchSize, len(extra))
		removed, removeErr := monitor.Unenroll(ctx, extra[start:end])
		if removeErr != nil {
			return result, removeErr
		}
		result.DriftRemove += len(removed.Succeeded)
	}
	return result, nil
}
