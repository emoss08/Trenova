package detentionservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	PolicyRepo        repositories.DetentionPolicyRepository
	OccurrenceRepo    repositories.DetentionOccurrenceRepository
	EvidenceRepo      repositories.DetentionEvidenceRepository
	AccessorialRepo   repositories.AccessorialChargeRepository
	PayAssignmentRepo repositories.WorkerPayAssignmentRepository
	PayProfileRepo    repositories.PayProfileRepository
	OrgCacheRepo      repositories.OrganizationCacheRepository
	NoticeRepo        repositories.DetentionNoticeRepository
	AnalyticsRepo     repositories.DetentionAnalyticsRepository
	CustomerRepo      repositories.CustomerRepository
	CommentRepo       repositories.ShipmentCommentRepository
	EmailService      services.EmailService
	AuditService      services.AuditService
}

type Service struct {
	l                 *zap.Logger
	policyRepo        repositories.DetentionPolicyRepository
	occurrenceRepo    repositories.DetentionOccurrenceRepository
	evidenceRepo      repositories.DetentionEvidenceRepository
	accessorialRepo   repositories.AccessorialChargeRepository
	payAssignmentRepo repositories.WorkerPayAssignmentRepository
	payProfileRepo    repositories.PayProfileRepository
	orgCacheRepo      repositories.OrganizationCacheRepository
	noticeRepo        repositories.DetentionNoticeRepository
	analyticsRepo     repositories.DetentionAnalyticsRepository
	customerRepo      repositories.CustomerRepository
	commentRepo       repositories.ShipmentCommentRepository
	emailService      services.EmailService
	auditService      services.AuditService
	now               func() int64
}

func New(p Params) *Service {
	return &Service{
		l:                 p.Logger.Named("service.detention"),
		policyRepo:        p.PolicyRepo,
		occurrenceRepo:    p.OccurrenceRepo,
		evidenceRepo:      p.EvidenceRepo,
		accessorialRepo:   p.AccessorialRepo,
		payAssignmentRepo: p.PayAssignmentRepo,
		payProfileRepo:    p.PayProfileRepo,
		orgCacheRepo:      p.OrgCacheRepo,
		noticeRepo:        p.NoticeRepo,
		analyticsRepo:     p.AnalyticsRepo,
		customerRepo:      p.CustomerRepo,
		commentRepo:       p.CommentRepo,
		emailService:      p.EmailService,
		auditService:      p.AuditService,
		now:               timeutils.NowUnix,
	}
}

// SyncShipmentResult reports what the recalculation did, so callers can decide
// whether shipment charges need rebuilding.
type SyncShipmentResult struct {
	Occurrences []*detention.DetentionOccurrence
	Skipped     int
	TotalAmount decimal.Decimal
}

// SyncShipment recomputes every stop's detention for a shipment.
//
// Frozen occurrences are never rewritten: once a charge has been billed,
// waived, or disputed the customer has seen a number, and silently changing it
// is how carriers lose credibility in a dispute.
func (s *Service) SyncShipment(
	ctx context.Context,
	entity *shipment.Shipment,
) (*SyncShipmentResult, error) {
	if entity == nil {
		return &SyncShipmentResult{TotalAmount: decimal.Zero}, nil
	}

	location := s.tenantLocation(ctx, entity.OrganizationID)

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	log := s.l.With(
		zap.String("operation", "SyncShipment"),
		zap.String("shipmentId", entity.ID.String()),
	)

	result := &SyncShipmentResult{
		Occurrences: make([]*detention.DetentionOccurrence, 0, 4),
		TotalAmount: decimal.Zero,
	}

	commodityIDs := shipmentCommodityIDs(entity)
	dayAccrued := map[string]decimal.Decimal{}
	shipmentAccrued := decimal.Zero
	now := s.now()

	for _, move := range entity.Moves {
		if move == nil {
			continue
		}

		payRate := s.resolveDriverPayRate(ctx, tenantInfo, move, now)

		for _, stop := range move.Stops {
			if !stopIsBillable(stop) {
				continue
			}

			existing, err := s.occurrenceRepo.GetByStop(ctx, &repositories.GetOccurrenceByStopRequest{
				StopID:     stop.ID,
				TenantInfo: tenantInfo,
			})
			if err != nil {
				return nil, err
			}

			if existing != nil && existing.IsFrozen() {
				result.Skipped++
				result.Occurrences = append(result.Occurrences, existing)
				shipmentAccrued = shipmentAccrued.Add(existing.BillableAmount)
				result.TotalAmount = result.TotalAmount.Add(existing.BillableAmount)
				continue
			}

			occurrence, err := s.computeStop(ctx, computeStopParams{
				shipment:        entity,
				move:            move,
				stop:            stop,
				existing:        existing,
				tenantInfo:      tenantInfo,
				commodityIDs:    commodityIDs,
				location:        location,
				payRate:         payRate,
				dayAccrued:      dayAccrued,
				shipmentAccrued: shipmentAccrued,
				now:             now,
			})
			if err != nil {
				log.Error("failed to compute detention for stop",
					zap.String("stopId", stop.ID.String()), zap.Error(err))
				return nil, err
			}

			if occurrence == nil {
				continue
			}

			result.Occurrences = append(result.Occurrences, occurrence)
			shipmentAccrued = shipmentAccrued.Add(occurrence.BillableAmount)
			result.TotalAmount = result.TotalAmount.Add(occurrence.BillableAmount)
		}
	}

	return result, nil
}

type computeStopParams struct {
	shipment        *shipment.Shipment
	move            *shipment.ShipmentMove
	stop            *shipment.Stop
	existing        *detention.DetentionOccurrence
	tenantInfo      pagination.TenantInfo
	commodityIDs    []pulid.ID
	location        *time.Location
	payRate         decimal.NullDecimal
	dayAccrued      map[string]decimal.Decimal
	shipmentAccrued decimal.Decimal
	now             int64
}

//nolint:funlen // one linear pass from stop facts to a persisted occurrence
func (s *Service) computeStop(
	ctx context.Context,
	p computeStopParams,
) (*detention.DetentionOccurrence, error) {
	matchInput := detention.MatchInput{
		CustomerID:     p.shipment.CustomerID,
		LocationID:     p.stop.LocationID,
		ShipmentTypeID: p.shipment.ShipmentTypeID,
		ServiceTypeID:  p.shipment.ServiceTypeID,
		CommodityIDs:   p.commodityIDs,
		StopType:       p.stop.Type,
		ScheduleType:   p.stop.ScheduleType,
		Timestamp:      *p.stop.ActualArrival,
	}

	candidates, err := s.policyRepo.GetResolutionCandidates(
		ctx,
		&repositories.GetResolutionCandidatesRequest{
			TenantInfo: p.tenantInfo,
			CustomerID: p.shipment.CustomerID,
			LocationID: p.stop.LocationID,
		},
	)
	if err != nil {
		return nil, err
	}

	resolution := ResolvePolicy(candidates, matchInput)
	if !resolution.Found() {
		return nil, nil
	}

	policy := resolution.Policy

	accessorial, err := s.accessorialRepo.GetByID(ctx, repositories.GetAccessorialChargeByIDRequest{
		ID:         policy.AccessorialChargeID,
		TenantInfo: &p.tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	snapshot := detention.NewPolicySnapshot(detention.SnapshotInput{
		Policy:          policy,
		StopType:        string(p.stop.Type),
		FreeMinutes:     policy.FreeMinutesForStopType(p.stop.Type),
		AccessorialCode: accessorial.Code,
		FlatRate:        accessorial.Amount,
		FlatRateUnit:    tierUnitFromAccessorial(accessorial.RateUnit),
		CapturedAt:      p.now,
	})

	var noticeSentAt *int64
	if p.existing != nil {
		noticeSentAt = p.existing.NoticeSentAt
	}

	computed := Compute(ComputeInput{
		Snapshot:         snapshot,
		StopType:         p.stop.Type,
		ScheduleType:     p.stop.ScheduleType,
		ArrivedAt:        p.stop.ActualArrival,
		DepartedAt:       p.stop.ActualDeparture,
		AppointmentStart: appointmentStart(p.stop),
		AppointmentEnd:   p.stop.ScheduledWindowEnd,
		NoticeSentAt:     noticeSentAt,
		Now:              p.now,
		Location:         p.location,
		DriverPayRate:    p.payRate,
		DayAccrued:       p.dayAccrued,
		ShipmentAccrued:  p.shipmentAccrued,
	})

	if !computed.Applicable {
		return nil, nil
	}

	for day, amount := range computed.DayAllocation {
		p.dayAccrued[day] = p.dayAccrued[day].Add(amount)
	}

	occurrence := buildOccurrence(p, policy, snapshot, computed)

	saved, err := s.occurrenceRepo.Upsert(ctx, occurrence)
	if err != nil {
		return nil, err
	}

	s.recordRecalculation(ctx, saved, p.existing)
	s.recordChargeComment(ctx, chargeCommentParams{
		occurrence: saved,
		previous:   p.existing,
		stop:       p.stop,
	})

	return saved, nil
}

func buildOccurrence(
	p computeStopParams,
	policy *detention.DetentionPolicy,
	snapshot detention.PolicySnapshot,
	computed ComputeResult,
) *detention.DetentionOccurrence {
	policyID := policy.ID

	occurrence := &detention.DetentionOccurrence{
		OrganizationID:     p.tenantInfo.OrgID,
		BusinessUnitID:     p.tenantInfo.BuID,
		ShipmentID:         p.shipment.ID,
		ShipmentMoveID:     p.move.ID,
		StopID:             p.stop.ID,
		CustomerID:         p.shipment.CustomerID,
		LocationID:         p.stop.LocationID,
		DetentionPolicyID:  &policyID,
		PolicySnapshot:     &snapshot,
		CalculationTrace:   computed.Trace,
		StopType:           p.stop.Type,
		ScheduleType:       p.stop.ScheduleType,
		AppointmentStart:   appointmentStart(p.stop),
		AppointmentEnd:     p.stop.ScheduledWindowEnd,
		ArrivedAt:          p.stop.ActualArrival,
		DepartedAt:         p.stop.ActualDeparture,
		ClockStartAt:       computed.ClockStartAt,
		ClockStopAt:        computed.ClockStopAt,
		FreeTimeExpiresAt:  computed.FreeTimeExpiresAt,
		NoticeDueAt:        computed.NoticeDueAt,
		NoticeDeadlineAt:   computed.NoticeDeadlineAt,
		IsOpen:             computed.IsOpen,
		ArrivedLate:        computed.ArrivedLate,
		LateByMinutes:      computed.LateByMinutes,
		FreeMinutesGranted: computed.FreeMinutesGranted,
		RawDwellMinutes:    computed.RawDwellMinutes,
		BillableMinutes:    computed.BillableMinutes,
		RoundedMinutes:     computed.RoundedMinutes,
		BillableUnits:      computed.BillableUnits,
		GrossAmount:        computed.GrossAmount,
		BillableAmount:     computed.BillableAmount,
		DriverPayMinutes:   computed.DriverPayMinutes,
		DriverPayAmount:    computed.DriverPayAmount,
		NetMargin:          computed.NetMargin,
		CapApplied:         computed.CapApplied,
		ConvertedToLayover: computed.ConvertedToLayover,
		Currency:           snapshot.Currency,
		Status:             computed.Status,
		NotificationStatus: computed.NotificationStatus,
		SuppressedByGate:   computed.SuppressedByGate,
		RequiresApproval:   computed.RequiresApproval,
	}

	if p.existing != nil {
		occurrence.ID = p.existing.ID
		occurrence.NoticeSentAt = p.existing.NoticeSentAt
		occurrence.ApprovedByID = p.existing.ApprovedByID
		occurrence.ApprovedAt = p.existing.ApprovedAt
		occurrence.EvidenceHead = p.existing.EvidenceHead
		occurrence.AdditionalChargeID = p.existing.AdditionalChargeID
		occurrence.CollectabilityScore = p.existing.CollectabilityScore

		if p.existing.Status == detention.OccurrenceStatusApproved &&
			computed.Status == detention.OccurrenceStatusPending &&
			!computed.RequiresApproval {
			occurrence.Status = detention.OccurrenceStatusApproved
		}
	}

	return occurrence
}

// recordRecalculation appends an evidence link whenever a stored amount moves,
// so the reason a number changed is never lost.
func (s *Service) recordRecalculation(
	ctx context.Context,
	saved *detention.DetentionOccurrence,
	existing *detention.DetentionOccurrence,
) {
	if s.evidenceRepo == nil || saved == nil {
		return
	}

	if existing != nil && existing.BillableAmount.Equal(saved.BillableAmount) &&
		existing.Status == saved.Status {
		return
	}

	summary := "Detention recalculated to " + saved.BillableAmount.StringFixed(2) +
		" " + saved.Currency + " (" + string(saved.Status) + ")"

	entry := &detention.DetentionEvidence{
		OrganizationID:        saved.OrganizationID,
		BusinessUnitID:        saved.BusinessUnitID,
		DetentionOccurrenceID: saved.ID,
		Kind:                  detention.EvidenceKindRecalculated,
		Source:                detention.EvidenceSourceSystem,
		Summary:               summary,
		ObservedAt:            s.now(),
		RecordedAt:            s.now(),
	}

	if _, err := s.evidenceRepo.Append(ctx, &repositories.AppendEvidenceRequest{
		Entry: entry,
	}); err != nil {
		s.l.Warn("failed to append detention recalculation evidence",
			zap.String("occurrenceId", saved.ID.String()), zap.Error(err))
	}
}

// resolveDriverPayRate finds the detention rate the assigned driver is paid, so
// the AR and AP sides of the same dwell can be compared on one record.
func (s *Service) resolveDriverPayRate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	move *shipment.ShipmentMove,
	asOf int64,
) decimal.NullDecimal {
	if s.payAssignmentRepo == nil || s.payProfileRepo == nil {
		return decimal.NullDecimal{}
	}

	if move.Assignment == nil || move.Assignment.PrimaryWorkerID == nil ||
		move.Assignment.PrimaryWorkerID.IsNil() {
		return decimal.NullDecimal{}
	}

	assignment, err := s.payAssignmentRepo.GetEffectiveForWorker(
		ctx,
		repositories.GetWorkerPayAssignmentRequest{
			TenantInfo: tenantInfo,
			WorkerID:   *move.Assignment.PrimaryWorkerID,
			AsOf:       asOf,
		},
	)
	if err != nil || assignment == nil {
		return decimal.NullDecimal{}
	}

	profile, err := s.payProfileRepo.GetByID(ctx, repositories.GetPayProfileByIDRequest{
		TenantInfo: tenantInfo,
		ID:         assignment.PayProfileID,
	})
	if err != nil || profile == nil {
		return decimal.NullDecimal{}
	}

	for _, comp := range profile.Components {
		if comp == nil || !comp.IsActive {
			continue
		}
		if comp.Kind == driverpay.ComponentKindDetention {
			return decimal.NewNullDecimal(comp.Rate)
		}
	}

	return decimal.NullDecimal{}
}

func stopIsBillable(stop *shipment.Stop) bool {
	if stop == nil || stop.Status == shipment.StopStatusCanceled {
		return false
	}

	if stop.ActualArrival == nil || *stop.ActualArrival <= 0 {
		return false
	}

	if stop.CountDetentionOverride != nil && !*stop.CountDetentionOverride {
		return false
	}

	return true
}

func appointmentStart(stop *shipment.Stop) *int64 {
	if stop.ScheduledWindowStart <= 0 {
		return nil
	}
	start := stop.ScheduledWindowStart
	return &start
}

func shipmentCommodityIDs(entity *shipment.Shipment) []pulid.ID {
	ids := make([]pulid.ID, 0, len(entity.Commodities))
	for _, commodity := range entity.Commodities {
		if commodity == nil || commodity.CommodityID.IsNil() {
			continue
		}
		ids = append(ids, commodity.CommodityID)
	}
	return ids
}

// tenantLocation resolves the organization's timezone so calendar-day caps
// break at the carrier's local midnight rather than UTC's.
func (s *Service) tenantLocation(ctx context.Context, orgID pulid.ID) *time.Location {
	if s.orgCacheRepo == nil {
		return time.UTC
	}

	org, err := s.orgCacheRepo.GetByID(ctx, orgID)
	if err != nil || org == nil {
		return time.UTC
	}

	loc, err := time.LoadLocation(timeutils.NormalizeTimezone(org.Timezone))
	if err != nil {
		return time.UTC
	}

	return loc
}
