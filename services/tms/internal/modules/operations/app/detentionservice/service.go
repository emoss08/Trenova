package detentionservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/watchtowersources"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/identity/app/orgzone"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
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
	PolicyRepo        detentionports.DetentionPolicyRepository
	OccurrenceRepo    detentionports.DetentionOccurrenceRepository
	EvidenceRepo      detentionports.DetentionEvidenceRepository
	AccessorialRepo   accessorialchargeports.AccessorialChargeRepository
	AgreementRepo     rateagreementports.RateAgreementRepository
	PayAssignmentRepo driverpayports.WorkerPayAssignmentRepository
	PayProfileRepo    driverpayports.PayProfileRepository
	Organizations     tenantports.OrganizationRepository
	NoticeRepo        detentionports.DetentionNoticeRepository
	AnalyticsRepo     detentionports.DetentionAnalyticsRepository
	CustomerRepo      customerports.CustomerRepository
	CommentRepo       shipmentcommentports.ShipmentCommentRepository
	DocumentTypeRepo  documenttypeports.DocumentTypeRepository
	EmailService      emailports.EmailService
	UploadService     documentuploadports.DocumentUploadService
	WorkflowStarter   kernelports.WorkflowStarter
	Templates         documenttemplateports.DocumentTemplateResolver
	ContextBuilder    *ContextBuilder
	AuditService      auditports.AuditService
	// Watchtower puts a running detention clock on the feed and takes it
	// off once the occurrence closes.
	Watchtower watchtowerports.WatchtowerProjector `optional:"true"`
	// Publisher wakes whichever agent covers detention when a clock starts
	// and when a notice is left for a person to send.
	Publisher agentports.AgentEventPublisher `optional:"true"`
	// Realtime refreshes the billing queue when a charge starts or stops
	// holding its shipment off an invoice.
	Realtime realtimeports.RealtimeService `optional:"true"`
	// Senders names who a notice would go out as, for its preview.
	Senders       emailports.EmailSenderResolver   `optional:"true"`
	DocumentStore documentports.DocumentStore      `optional:"true"`
	Documents     documentports.DocumentRepository `optional:"true"`
}

type Service struct {
	l                 *zap.Logger
	policyRepo        detentionports.DetentionPolicyRepository
	occurrenceRepo    detentionports.DetentionOccurrenceRepository
	evidenceRepo      detentionports.DetentionEvidenceRepository
	accessorialRepo   accessorialchargeports.AccessorialChargeRepository
	agreementRepo     rateagreementports.RateAgreementRepository
	payAssignmentRepo driverpayports.WorkerPayAssignmentRepository
	payProfileRepo    driverpayports.PayProfileRepository
	organizations     tenantports.OrganizationRepository
	noticeRepo        detentionports.DetentionNoticeRepository
	analyticsRepo     detentionports.DetentionAnalyticsRepository
	customerRepo      customerports.CustomerRepository
	commentRepo       shipmentcommentports.ShipmentCommentRepository
	documentTypeRepo  documenttypeports.DocumentTypeRepository
	emailService      emailports.EmailService
	uploadService     documentuploadports.DocumentUploadService
	workflowStarter   kernelports.WorkflowStarter
	templates         documenttemplateports.DocumentTemplateResolver
	contextBuilder    *ContextBuilder
	auditService      auditports.AuditService
	watchtower        watchtowerports.WatchtowerProjector
	publisher         agentports.AgentEventPublisher
	realtime          realtimeports.RealtimeService
	senders           emailports.EmailSenderResolver
	documentStore     documentports.DocumentStore
	documents         documentports.DocumentRepository
	now               func() int64
}

var _ detentionports.DetentionNoticeService = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		l:                 p.Logger.Named("service.detention"),
		policyRepo:        p.PolicyRepo,
		occurrenceRepo:    p.OccurrenceRepo,
		evidenceRepo:      p.EvidenceRepo,
		accessorialRepo:   p.AccessorialRepo,
		agreementRepo:     p.AgreementRepo,
		payAssignmentRepo: p.PayAssignmentRepo,
		payProfileRepo:    p.PayProfileRepo,
		organizations:     p.Organizations,
		noticeRepo:        p.NoticeRepo,
		analyticsRepo:     p.AnalyticsRepo,
		customerRepo:      p.CustomerRepo,
		commentRepo:       p.CommentRepo,
		documentTypeRepo:  p.DocumentTypeRepo,
		emailService:      p.EmailService,
		uploadService:     p.UploadService,
		workflowStarter:   p.WorkflowStarter,
		templates:         p.Templates,
		contextBuilder:    p.ContextBuilder,
		watchtower:        p.Watchtower,
		publisher:         p.Publisher,
		realtime:          p.Realtime,
		senders:           p.Senders,
		documentStore:     p.DocumentStore,
		documents:         p.Documents,
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

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	location, err := s.tenantLocation(ctx, tenantInfo)
	if err != nil {
		return nil, err
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

	// The contract's own accessorial prices, read once for the whole shipment.
	prices := s.resolveContractPrices(ctx, entity, tenantInfo, now)

	for _, move := range entity.Moves {
		if move == nil {
			continue
		}

		payRate := s.resolveDriverPayRate(ctx, tenantInfo, move, now)

		for _, stop := range move.Stops {
			if !stopIsBillable(stop) {
				continue
			}

			existing, err := s.occurrenceRepo.GetByStop(
				ctx,
				&detentionports.GetOccurrenceByStopRequest{
					StopID:     stop.ID,
					TenantInfo: tenantInfo,
				},
			)
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
				prices:          prices,
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
	// prices is what the shipment's own contract charges, when a contract
	// priced it. Nil means the organization defaults apply.
	prices *contractPrices
	now    int64
}

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
		&detentionports.GetResolutionCandidatesRequest{
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

	accessorial, err := s.accessorialRepo.GetByID(
		ctx,
		accessorialchargeports.GetAccessorialChargeByIDRequest{
			ID:         policy.AccessorialChargeID,
			TenantInfo: &p.tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	// The contract's price wins over the organization default, which is what
	// makes the rate confirmation and the invoice agree on detention.
	flatRate := accessorial.Amount
	rateUnit := accessorial.RateUnit

	if contractRate, contractUnit, priced := p.prices.priceFor(accessorial); priced {
		flatRate = contractRate
		rateUnit = contractUnit
	}

	snapshot := detention.NewPolicySnapshot(detention.SnapshotInput{
		Policy:          policy,
		StopType:        string(p.stop.Type),
		FreeMinutes:     policy.FreeMinutesForStopType(p.stop.Type),
		AccessorialCode: accessorial.Code,
		FlatRate:        flatRate,
		FlatRateUnit:    tierUnitFromAccessorial(rateUnit),
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
	s.publishBillingHoldChange(ctx, p.existing, saved, pulid.Nil)
	s.recordChargeComment(ctx, chargeCommentParams{
		occurrence: saved,
		previous:   p.existing,
		stop:       p.stop,
	})
	s.projectToWatchtower(ctx, saved, p)

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
		DriverPayAmount:    decimal.NewNullDecimal(computed.DriverPayAmount),
		NetMargin:          decimal.NewNullDecimal(computed.NetMargin),
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

	if _, err := s.evidenceRepo.Append(ctx, &detentionports.AppendEvidenceRequest{
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
		driverpayports.GetWorkerPayAssignmentRequest{
			TenantInfo: tenantInfo,
			WorkerID:   *move.Assignment.PrimaryWorkerID,
			AsOf:       asOf,
		},
	)
	if err != nil || assignment == nil {
		return decimal.NullDecimal{}
	}

	profile, err := s.payProfileRepo.GetByID(ctx, driverpayports.GetPayProfileByIDRequest{
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
func (s *Service) tenantLocation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*time.Location, error) {
	return orgzone.Location(ctx, s.organizations, tenantInfo)
}

// projectToWatchtower puts a running detention clock on the feed and takes
// it off once the occurrence closes. The names come from what the
// recalculation already loaded, so the live item reads the same as the one
// the nightly snapshot would write without a second round of queries.
func (s *Service) projectToWatchtower(
	ctx context.Context,
	saved *detention.DetentionOccurrence,
	p computeStopParams,
) {
	if s.watchtower == nil || saved == nil {
		return
	}

	if !saved.IsOpen {
		s.watchtower.Resolve(
			ctx,
			p.tenantInfo,
			watchtower.SourceDetentionOccurrence,
			saved.ID.String(),
		)

		return
	}

	described := *saved
	if described.ShipmentProNumber == "" && p.shipment != nil {
		described.ShipmentProNumber = p.shipment.ProNumber
	}
	if described.CustomerName == "" && p.shipment != nil && p.shipment.Customer != nil {
		described.CustomerName = p.shipment.Customer.Name
	}
	if described.LocationName == "" && p.stop != nil && p.stop.Location != nil {
		described.LocationName = p.stop.Location.Name
	}

	s.watchtower.Upsert(ctx, watchtowersources.DescribeDetentionOccurrence(&described))
}

// publishOccurrenceOpened wakes the detention desk the moment a clock
// starts, and only then. Recalculation runs on every stop update, so an
// occurrence that was already running is not news; the event fires on the
// transition into open, not on every pass over one that already is.
func (s *Service) publishOccurrenceOpened(
	ctx context.Context,
	saved *detention.DetentionOccurrence,
	p computeStopParams,
) {
	if saved == nil || !saved.IsOpen {
		return
	}
	if p.existing != nil && p.existing.IsOpen {
		return
	}

	agentports.PublishAgentEvent(ctx, s.publisher, agentports.AgentEvent{
		Kind:       agent.EventDetentionOccurrenceOpened,
		SubjectID:  saved.ID,
		TenantInfo: p.tenantInfo,
	})
}
