package carrierintelservice

import (
	"context"
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type Optional[T any] struct {
	Set   bool
	Value T
}

func Some[T any](v T) Optional[T] { return Optional[T]{Set: true, Value: v} }

type ControlPatch struct {
	EnrollmentPolicy        Optional[carrierintel.EnrollmentPolicy]
	RecentUsageDays         Optional[int]
	IncludeOpenTenders      Optional[bool]
	AutoEnrollOnCreate      Optional[bool]
	AutoUnenrollOnInactive  Optional[bool]
	ExclusiveWatchlist      Optional[bool]
	PollIntervalMinutes     Optional[int]
	SnapshotTTLHours        Optional[int]
	FullProfileTTLDays      Optional[int]
	PreTenderRefreshEnabled Optional[bool]
	PreTenderMaxAgeHours    Optional[int]
	HardMaxAgeHours         Optional[int]
	ConfirmBlockingChanges  Optional[bool]
	OutagePolicy            Optional[carrierintel.OutagePolicy]
	AutoDisqualifyOnBlock   Optional[bool]
	AutoApplySafetyRating   Optional[bool]
	Rules                   Optional[carrierintel.RuleSettings]
	AutoSyncFields          Optional[[]carrierintel.SyncField]
	MonthlySpendCap         Optional[*decimal.Decimal]
	SoftCapPercent          Optional[int]
	DailyFullProfileCap     Optional[*int]
	RawRetentionDays        Optional[int]
	SnapshotHistoryLimit    Optional[int]
	SelfMonitoringEnabled   Optional[bool]
	ConfirmEstimatedCost    bool
	ExpectedVersion         *int64
}

func applyPatch[T any](field *T, opt Optional[T]) bool {
	if !opt.Set {
		return false
	}
	*field = opt.Value
	return true
}

func (s *Service) UpdateControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	patch *ControlPatch,
) (*carrierintel.CarrierIntelControl, error) {
	log := s.l.With(zap.String("operation", "UpdateControl"))

	original, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if patch.ExpectedVersion != nil && *patch.ExpectedVersion != original.Version {
		return nil, errortypes.NewValidationError("version", errortypes.ErrVersionMismatch,
			"Carrier intelligence settings were changed by someone else. Reload and try again")
	}

	entity := *original
	entity.Rules = maps.Clone(original.Rules)
	entity.AutoSyncFields = slices.Clone(original.AutoSyncFields)

	applyPatch(&entity.EnrollmentPolicy, patch.EnrollmentPolicy)
	applyPatch(&entity.RecentUsageDays, patch.RecentUsageDays)
	applyPatch(&entity.IncludeOpenTenders, patch.IncludeOpenTenders)
	applyPatch(&entity.AutoEnrollOnCreate, patch.AutoEnrollOnCreate)
	applyPatch(&entity.AutoUnenrollOnInactive, patch.AutoUnenrollOnInactive)
	applyPatch(&entity.ExclusiveWatchlist, patch.ExclusiveWatchlist)
	applyPatch(&entity.PollIntervalMinutes, patch.PollIntervalMinutes)
	applyPatch(&entity.SnapshotTTLHours, patch.SnapshotTTLHours)
	applyPatch(&entity.FullProfileTTLDays, patch.FullProfileTTLDays)
	applyPatch(&entity.PreTenderRefreshEnabled, patch.PreTenderRefreshEnabled)
	applyPatch(&entity.PreTenderMaxAgeHours, patch.PreTenderMaxAgeHours)
	applyPatch(&entity.HardMaxAgeHours, patch.HardMaxAgeHours)
	applyPatch(&entity.ConfirmBlockingChanges, patch.ConfirmBlockingChanges)
	applyPatch(&entity.OutagePolicy, patch.OutagePolicy)
	applyPatch(&entity.AutoDisqualifyOnBlock, patch.AutoDisqualifyOnBlock)
	applyPatch(&entity.MonthlySpendCap, patch.MonthlySpendCap)
	applyPatch(&entity.SoftCapPercent, patch.SoftCapPercent)
	applyPatch(&entity.DailyFullProfileCap, patch.DailyFullProfileCap)
	applyPatch(&entity.RawRetentionDays, patch.RawRetentionDays)
	applyPatch(&entity.SnapshotHistoryLimit, patch.SnapshotHistoryLimit)
	applyPatch(&entity.SelfMonitoringEnabled, patch.SelfMonitoringEnabled)

	policyChanged := false
	if applyPatch(&entity.Rules, patch.Rules) {
		if entity.Rules == nil {
			entity.Rules = carrierintel.RuleSettings{}
		}
		policyChanged = !rulesEqual(original.Rules, entity.Rules)
	}
	if applyPatch(&entity.AutoSyncFields, patch.AutoSyncFields) && entity.AutoSyncFields == nil {
		entity.AutoSyncFields = []carrierintel.SyncField{}
	}
	if applyPatch(&entity.AutoApplySafetyRating, patch.AutoApplySafetyRating) &&
		entity.AutoApplySafetyRating != original.AutoApplySafetyRating {
		policyChanged = true
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	enrollmentChanged := entity.EnrollmentPolicy != original.EnrollmentPolicy ||
		entity.RecentUsageDays != original.RecentUsageDays ||
		entity.IncludeOpenTenders != original.IncludeOpenTenders ||
		entity.SelfMonitoringEnabled != original.SelfMonitoringEnabled

	if entity.EnrollmentPolicy == carrierintel.EnrollmentPolicyAllActive &&
		original.EnrollmentPolicy != carrierintel.EnrollmentPolicyAllActive &&
		!patch.ConfirmEstimatedCost {
		return nil, errortypes.NewBusinessError(
			"Monitoring every active carrier is billed per carrier each month. Review the cost estimate and confirm to continue",
		).WithParam("requiresCostConfirmation", "true")
	}

	if policyChanged {
		entity.PolicyVersion = original.PolicyVersion + 1
	}

	updated, err := s.controlRepo.Update(ctx, &entity)
	if err != nil {
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceCarrierIntelligence,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpManage,
		UserID:         tenantInfo.UserID,
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    tenantInfo.UserID,
		CurrentState:   jsonutils.MustToJSON(updated),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
		Critical:       true,
	},
		auditservice.WithComment("Carrier intelligence settings updated"),
		auditservice.WithDiff(original, updated),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	if enrollmentChanged {
		if _, err = s.ReconcileDesired(ctx, tenantInfo); err != nil {
			log.Warn(
				"failed to reconcile monitoring enrollment after settings change",
				zap.Error(err),
			)
		}
	}
	if original.MonthlySpendCap != nil && (updated.MonthlySpendCap == nil ||
		updated.MonthlySpendCap.GreaterThan(*original.MonthlySpendCap)) {
		if err = s.ResumeFeeds(ctx, tenantInfo); err != nil {
			log.Warn("failed to resume paused feeds", zap.Error(err))
		}
	}

	s.publish(ctx, tenantInfo, "carrier_intelligence_control", "updated", updated.ID)
	return updated, nil
}

func rulesEqual(a, b carrierintel.RuleSettings) bool {
	if len(a) != len(b) {
		return false
	}
	for code, left := range a {
		right, ok := b[code]
		if !ok || left.Action != right.Action || !maps.Equal(left.Params, right.Params) {
			return false
		}
	}
	return true
}

func (s *Service) RuleCatalog() []carrierintel.RuleDefinition {
	return carrierintel.Catalog()
}

func (s *Service) ProviderDescriptor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (integration.Type, carrierintel.ProviderDescriptor, bool, error) {
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return "", carrierintel.ProviderDescriptor{}, false, err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return "", carrierintel.ProviderDescriptor{}, false, nil
	}
	connector, ok := s.connectors[provider]
	if !ok {
		return provider, carrierintel.ProviderDescriptor{}, false, nil
	}
	return provider, connector.Descriptor(), true, nil
}

func (s *Service) SwitchPrimaryProvider(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider integration.Type,
) (*carrierintel.CarrierIntelControl, error) {
	if !provider.SupportsCarrierIntelligence() {
		return nil, errortypes.NewValidationError("provider", errortypes.ErrInvalid,
			"Provider is not a carrier intelligence provider")
	}
	if _, ok := s.connectors[provider]; !ok {
		return nil, errortypes.NewBusinessError("{0} is not available", provider.String())
	}
	runtime, err := s.integrations.GetRuntimeConfig(ctx, tenantInfo, provider)
	if err != nil {
		return nil, err
	}
	if !runtime.Ready {
		return nil, errortypes.NewBusinessError(
			"Enable and configure {0} in Integrations before making it primary", provider.String(),
		)
	}

	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	previous, hadPrevious := control.PrimaryType()
	if hadPrevious && previous == provider {
		return control, nil
	}

	if hadPrevious {
		s.releaseWatchlist(ctx, tenantInfo, previous)
	}

	original := *control
	control.PrimaryProvider = &provider
	if control.FallbackProvider != nil && *control.FallbackProvider == provider {
		control.FallbackProvider = nil
	}
	control.PolicyVersion++
	updated, err := s.controlRepo.Update(ctx, control)
	if err != nil {
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceCarrierIntelligence,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpManage,
		UserID:         tenantInfo.UserID,
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    tenantInfo.UserID,
		CurrentState:   jsonutils.MustToJSON(updated),
		PreviousState:  jsonutils.MustToJSON(&original),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
		Critical:       true,
	}, auditservice.WithComment("Carrier intelligence primary provider switched to "+provider.String())); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	if _, err = s.ReconcileDesired(ctx, tenantInfo); err != nil {
		s.l.Warn("failed to reconcile enrollment after provider switch", zap.Error(err))
	}
	s.publish(ctx, tenantInfo, "carrier_intelligence_control", "updated", updated.ID)
	return updated, nil
}

func (s *Service) releaseWatchlist(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider integration.Type,
) {
	rows, err := s.enrollmentRepo.ListDesired(ctx, tenantInfo, provider)
	if err != nil {
		s.l.Warn("failed to list enrollments to release", zap.Error(err))
		return
	}
	if len(rows) == 0 {
		return
	}
	now := s.now()
	for _, row := range rows {
		row.MarkDesired(carrierintel.DesiredStateNotEnrolled, "", now)
	}
	if err = s.enrollmentRepo.Upsert(ctx, rows); err != nil {
		s.l.Warn("failed to mark enrollments for release", zap.Error(err))
		return
	}

	connector, ok := s.connectors[provider]
	if !ok || !connector.Descriptor().Capabilities.Has(carrierintel.CapabilityNativeMonitoring) {
		for _, row := range rows {
			row.MarkSynced(now)
		}
		if err = s.enrollmentRepo.SaveSyncState(ctx, rows); err != nil {
			s.l.Warn("failed to release snapshot enrollments", zap.Error(err))
		}
		return
	}

	bound, err := s.bind(ctx, tenantInfo, provider, false)
	if err != nil {
		s.l.Warn("previous provider cannot be reached to release its watchlist", zap.Error(err))
		return
	}
	monitor, ok := bound.client.(services.CarrierIntelNativeMonitor)
	if !ok {
		return
	}
	result := &SyncResult{}
	if err = s.syncBatches(ctx, &syncBatch{
		tenant: tenantInfo,
		bound:  bound,
		rows:   withRefs(rows),
		call:   monitor.Unenroll,
		result: result,
	}); err != nil {
		s.l.Warn("failed to release previous provider watchlist", zap.Error(err))
	}
}

func withRefs(
	rows []*carrierintel.CarrierMonitoringEnrollment,
) []*carrierintel.CarrierMonitoringEnrollment {
	out := make([]*carrierintel.CarrierMonitoringEnrollment, 0, len(rows))
	for _, row := range rows {
		if row.ProviderRef != "" {
			out = append(out, row)
		}
	}
	return out
}
